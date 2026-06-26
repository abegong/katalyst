package sqlite_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection"
	sqlitestore "github.com/abegong/katalyst/internal/storage/collection/sqlite"

	_ "modernc.org/sqlite"
)

func setupDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "content.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE notes (
		slug TEXT PRIMARY KEY,
		title TEXT,
		year INTEGER,
		status TEXT,
		author_first TEXT,
		author_last TEXT,
		body TEXT
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO notes (slug, title, year, status, author_first, author_last, body) VALUES
		('dune', 'Dune', 1965, 'published', 'Frank', 'Herbert', '# Dune'),
		('messiah', 'Dune Messiah', 1969, 'published', 'Frank', 'Herbert', '# Dune Messiah')`); err != nil {
		t.Fatalf("seed rows: %v", err)
	}
	return path
}

func notesCollection() collection.Collection {
	return collection.Collection{
		Name:          "notes",
		Table:         "notes",
		IDColumn:      "slug",
		ContentKind:   "markdown",
		ContentColumn: "body",
		Attributes: map[string]collection.AttributeCapture{
			"title":  {Column: "title"},
			"year":   {Column: "year"},
			"status": {Column: "status"},
			"author": {
				Columns: map[string]string{
					"first": "author_first",
					"last":  "author_last",
				},
			},
		},
	}
}

func TestDefinition_Items_sortedIDsAndLabels(t *testing.T) {
	path := setupDB(t)
	c := notesCollection()
	def := sqlitestore.New(path, []collection.Collection{c})

	items, err := def.Items(c)
	if err != nil {
		t.Fatalf("Items: %v", err)
	}
	if len(items) != 2 || items[0].ID != "dune" || items[1].ID != "messiah" {
		t.Fatalf("expected [dune messiah], got %+v", items)
	}
	if items[0].Path != path+":notes/dune" {
		t.Fatalf("item path = %q, want %q", items[0].Path, path+":notes/dune")
	}
}

func TestDefinition_Read_capturesAttributesAndContent(t *testing.T) {
	path := setupDB(t)
	c := notesCollection()
	def := sqlitestore.New(path, []collection.Collection{c})

	raw, doc, err := def.Read(collection.Item{Collection: c, ID: "dune"})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(string(raw), "title: Dune") {
		t.Fatalf("raw document missing attributes:\n%s", raw)
	}
	if doc.Meta["title"] != "Dune" || fmt.Sprint(doc.Meta["year"]) != "1965" {
		t.Fatalf("unexpected scalar attributes: %#v", doc.Meta)
	}
	author, ok := doc.Meta["author"].(map[string]any)
	if !ok {
		t.Fatalf("author attribute = %#v, want object", doc.Meta["author"])
	}
	if author["first"] != "Frank" || author["last"] != "Herbert" {
		t.Fatalf("author = %#v, want Frank Herbert", author)
	}
	if string(doc.Body) != "# Dune" {
		t.Fatalf("body = %q, want # Dune", doc.Body)
	}
}

func TestDefinition_Read_withoutAttributesCapturesScalarFallback(t *testing.T) {
	path := setupDB(t)
	c := collection.Collection{
		Name:          "notes",
		Table:         "notes",
		IDColumn:      "slug",
		ContentColumn: "body",
	}
	def := sqlitestore.New(path, []collection.Collection{c})

	_, doc, err := def.Read(collection.Item{Collection: c, ID: "dune"})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, ok := doc.Meta["slug"]; ok {
		t.Fatalf("fallback attributes included id column: %#v", doc.Meta)
	}
	if _, ok := doc.Meta["body"]; ok {
		t.Fatalf("fallback attributes included content column: %#v", doc.Meta)
	}
	if doc.Meta["title"] != "Dune" || doc.Meta["author_first"] != "Frank" {
		t.Fatalf("fallback attributes = %#v", doc.Meta)
	}
}

func TestDefinition_AddUpdateDelete_translateAttributeColumns(t *testing.T) {
	path := setupDB(t)
	c := notesCollection()
	def := sqlitestore.New(path, []collection.Collection{c})

	if err := def.Add(c, "children", map[string]any{"title": "Children of Dune", "year": 1976}, []byte("# Children")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got := scalar(t, path, `SELECT title FROM notes WHERE slug = 'children'`); got != "Children of Dune" {
		t.Fatalf("inserted title = %#v", got)
	}
	if got := scalar(t, path, `SELECT body FROM notes WHERE slug = 'children'`); got != "# Children" {
		t.Fatalf("inserted body = %#v", got)
	}

	_, doc, err := def.Read(collection.Item{Collection: c, ID: "children"})
	if err != nil {
		t.Fatalf("Read inserted row: %v", err)
	}
	doc.Meta["year"] = 1977
	doc.Meta["status"] = "draft"
	if err := def.Update(c, "children", doc.Meta, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := scalar(t, path, `SELECT year FROM notes WHERE slug = 'children'`); got != int64(1977) {
		t.Fatalf("updated year = %#v", got)
	}
	if got := scalar(t, path, `SELECT status FROM notes WHERE slug = 'children'`); got != "draft" {
		t.Fatalf("updated status = %#v", got)
	}

	if err := def.Delete(c, "children"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got := scalar(t, path, `SELECT COUNT(1) FROM notes WHERE slug = 'children'`); got != int64(0) {
		t.Fatalf("deleted count = %#v", got)
	}
}

func TestDefinition_Add_rejectsStructuredAttributeWrites(t *testing.T) {
	path := setupDB(t)
	c := notesCollection()
	def := sqlitestore.New(path, []collection.Collection{c})

	err := def.Add(c, "bad", map[string]any{"author": map[string]any{"first": "Frank"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "structured") {
		t.Fatalf("expected structured attribute write error, got %v", err)
	}
}

func TestDefinition_Read_validatesConfiguredColumns(t *testing.T) {
	path := setupDB(t)
	c := notesCollection()
	c.Attributes["subtitle"] = collection.AttributeCapture{Column: "subtitle"}
	def := sqlitestore.New(path, []collection.Collection{c})

	_, _, err := def.Read(collection.Item{Collection: c, ID: "dune"})
	if err == nil || !strings.Contains(err.Error(), `attribute "subtitle" column "subtitle" does not exist`) {
		t.Fatalf("expected missing attribute column error, got %v", err)
	}
}

func TestDefinition_Scope_unitIsCollection(t *testing.T) {
	if g := sqlitestore.New("", nil).Scope(); g != storage.UnitIsCollection {
		t.Fatalf("Scope = %v, want UnitIsCollection", g)
	}
}

func scalar(t *testing.T, path, query string) any {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	var v any
	if err := db.QueryRow(query).Scan(&v); err != nil {
		t.Fatalf("query scalar: %v", err)
	}
	return v
}
