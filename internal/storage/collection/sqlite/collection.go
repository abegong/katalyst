// Package sqlite maps one SQLite table to one Katalyst collection: each row is
// one item, the configured id column is the item id, scalar columns are
// metadata, and an optional body column supplies body text.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection"
	"gopkg.in/yaml.v3"

	_ "modernc.org/sqlite"
)

var identRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Definition maps SQLite tables onto collections.
type Definition struct {
	path        string
	collections []collection.Collection
}

// New builds a SQLite definition for path and collections.
func New(path string, collections []collection.Collection) *Definition {
	return &Definition{path: path, collections: collections}
}

// Granularity is UnitIsCollection: one table is a collection and rows are items.
func (d *Definition) Granularity() storage.Granularity { return storage.UnitIsCollection }

// Collections returns the collections this definition maps.
func (d *Definition) Collections() []collection.Collection { return d.collections }

// Items lists row ids in a collection table.
func (d *Definition) Items(c collection.Collection) ([]collection.Item, error) {
	db, err := d.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	table, id, err := tableAndID(c)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", id, table, id))
	if err != nil {
		return nil, fmt.Errorf("collection %q: %w", c.Name, err)
	}
	defer rows.Close()

	var out []collection.Item
	for rows.Next() {
		var raw any
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		itemID := fmt.Sprint(normalize(raw))
		out = append(out, collection.Item{Collection: c, ID: itemID, Path: d.label(c, itemID)})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Unmatched is empty for the first SQLite shape: a configured table has no
// adjacent references to classify.
func (d *Definition) Unmatched(collection.Collection) ([]storage.Reference, error) {
	return nil, nil
}

// Reference reconstructs the row locator for an item id.
func (d *Definition) Reference(c collection.Collection, id string) (storage.Reference, error) {
	if _, _, err := tableAndID(c); err != nil {
		return "", err
	}
	return storage.Reference(d.label(c, id)), nil
}

// Read loads one SQLite row and decodes it into a markdownbodytext.Document.
func (d *Definition) Read(item collection.Item) ([]byte, *markdownbodytext.Document, error) {
	db, err := d.open()
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()
	return d.read(db, item.Collection, item.ID)
}

// Exists reports whether a row exists.
func (d *Definition) Exists(c collection.Collection, id string) (bool, error) {
	db, err := d.open()
	if err != nil {
		return false, err
	}
	defer db.Close()
	table, idCol, err := tableAndID(c)
	if err != nil {
		return false, err
	}
	var n int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE %s = ?", table, idCol), id).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Add inserts a row.
func (d *Definition) Add(c collection.Collection, id string, meta map[string]any, body []byte) error {
	db, err := d.open()
	if err != nil {
		return err
	}
	defer db.Close()
	cols, err := d.columns(db, c)
	if err != nil {
		return err
	}
	table, _, err := tableAndID(c)
	if err != nil {
		return err
	}
	idCol := c.IDColumn

	values := map[string]any{idCol: id}
	for k, v := range meta {
		values[k] = v
	}
	values[idCol] = id
	if c.BodyColumn != "" {
		values[c.BodyColumn] = string(body)
	}
	names, args, err := orderedValues(values, cols, true)
	if err != nil {
		return err
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(names)), ",")
	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, quoteList(names), placeholders), args...)
	if err != nil {
		return fmt.Errorf("collection %q: add %q: %w", c.Name, id, err)
	}
	return nil
}

// Update updates metadata columns, preserving body unless body is non-nil.
func (d *Definition) Update(c collection.Collection, id string, meta map[string]any, body []byte) error {
	db, err := d.open()
	if err != nil {
		return err
	}
	defer db.Close()
	cols, err := d.columns(db, c)
	if err != nil {
		return err
	}
	table, idQ, err := tableAndID(c)
	if err != nil {
		return err
	}
	idCol := c.IDColumn

	values := map[string]any{}
	for k, v := range meta {
		if k != idCol {
			values[k] = v
		}
	}
	if body != nil && c.BodyColumn != "" {
		values[c.BodyColumn] = string(body)
	}
	names, args, err := orderedValues(values, cols, false)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return nil
	}
	sets := make([]string, len(names))
	for i, name := range names {
		sets[i] = quote(name) + " = ?"
	}
	args = append(args, id)
	res, err := db.Exec(fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, strings.Join(sets, ", "), idQ), args...)
	if err != nil {
		return fmt.Errorf("collection %q: update %q: %w", c.Name, id, err)
	}
	return expectOne(res, c.Name, id)
}

// Delete deletes a row.
func (d *Definition) Delete(c collection.Collection, id string) error {
	db, err := d.open()
	if err != nil {
		return err
	}
	defer db.Close()
	table, idCol, err := tableAndID(c)
	if err != nil {
		return err
	}
	res, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, idCol), id)
	if err != nil {
		return fmt.Errorf("collection %q: delete %q: %w", c.Name, id, err)
	}
	return expectOne(res, c.Name, id)
}

func (d *Definition) open() (*sql.DB, error) {
	if d.path == "" {
		return nil, errors.New("sqlite storage path is empty")
	}
	return sql.Open("sqlite", d.path)
}

func (d *Definition) read(db *sql.DB, c collection.Collection, id string) ([]byte, *markdownbodytext.Document, error) {
	table, idCol, err := tableAndID(c)
	if err != nil {
		return nil, nil, err
	}
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", table, idCol), id)
	if err != nil {
		return nil, nil, fmt.Errorf("collection %q: read %q: %w", c.Name, id, err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil, fmt.Errorf("unknown item %q in collection %q", id, c.Name)
	}
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, nil, err
	}
	if rows.Next() {
		return nil, nil, fmt.Errorf("collection %q: id %q matched more than one row", c.Name, id)
	}

	meta := map[string]any{}
	var body []byte
	for i, col := range cols {
		v := normalize(vals[i])
		if col == c.BodyColumn {
			if v != nil {
				body = []byte(fmt.Sprint(v))
			}
			continue
		}
		if v != nil {
			meta[col] = v
		}
	}
	raw, frontmatter, err := rawDocument(meta, body)
	if err != nil {
		return nil, nil, err
	}
	doc := &markdownbodytext.Document{
		HasFrontmatter: true,
		Format:         markdownbodytext.KindYAML,
		Meta:           meta,
		Body:           body,
		BodyLine:       1,
		Lines:          map[string]int{},
		Frontmatter:    frontmatter,
	}
	return raw, doc, nil
}

func (d *Definition) columns(db *sql.DB, c collection.Collection) (map[string]bool, error) {
	table, _, err := tableAndID(c)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, fmt.Errorf("collection %q: %w", c.Name, err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[name] = true
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("collection %q: table %q has no columns or does not exist", c.Name, c.Table)
	}
	if !out[c.IDColumn] {
		return nil, fmt.Errorf("collection %q: id column %q does not exist", c.Name, c.IDColumn)
	}
	if c.BodyColumn != "" && !out[c.BodyColumn] {
		return nil, fmt.Errorf("collection %q: body column %q does not exist", c.Name, c.BodyColumn)
	}
	return out, rows.Err()
}

func tableAndID(c collection.Collection) (string, string, error) {
	table, err := quoteIdent(c.Table)
	if err != nil {
		return "", "", fmt.Errorf("collection %q: table: %w", c.Name, err)
	}
	id, err := quoteIdent(c.IDColumn)
	if err != nil {
		return "", "", fmt.Errorf("collection %q: id: %w", c.Name, err)
	}
	return table, id, nil
}

func orderedValues(values map[string]any, columns map[string]bool, requireAny bool) ([]string, []any, error) {
	names := make([]string, 0, len(values))
	for name := range values {
		if _, err := quoteIdent(name); err != nil {
			return nil, nil, err
		}
		if !columns[name] {
			return nil, nil, fmt.Errorf("unknown sqlite column %q", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if requireAny && len(names) == 0 {
		return nil, nil, errors.New("no sqlite columns to write")
	}
	args := make([]any, len(names))
	for i, name := range names {
		args[i] = values[name]
	}
	return names, args, nil
}

func quoteIdent(s string) (string, error) {
	if !identRE.MatchString(s) {
		return "", fmt.Errorf("invalid identifier %q", s)
	}
	return quote(s), nil
}

func quote(s string) string { return `"` + s + `"` }

func quoteList(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = quote(name)
	}
	return strings.Join(quoted, ", ")
}

func rawDocument(meta map[string]any, body []byte) ([]byte, []byte, error) {
	fm, err := yaml.Marshal(meta)
	if err != nil {
		return nil, nil, err
	}
	out := append([]byte("---\n"), fm...)
	if !strings.HasSuffix(string(fm), "\n") {
		out = append(out, '\n')
	}
	out = append(out, []byte("---\n")...)
	out = append(out, body...)
	return out, fm, nil
}

func normalize(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return x
	}
}

func expectOne(res sql.Result, collectionName, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("unknown item %q in collection %q", id, collectionName)
	}
	return nil
}

func (d *Definition) label(c collection.Collection, id string) string {
	return fmt.Sprintf("%s:%s/%s", d.path, c.Name, id)
}
