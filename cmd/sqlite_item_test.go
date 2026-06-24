package cmd_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupSQLiteRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	projecttest := map[string]string{
		"storage/db.yaml": `type: sqlite
path: content.sqlite
collections:
  notes:
    table: notes
    id: slug
    body: body
    checks:
      - kind: object_required_field
        field: title
`,
	}
	writeProject(t, dir, projecttest)

	db, err := sql.Open("sqlite", filepath.Join(dir, "content.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE notes (slug TEXT PRIMARY KEY, title TEXT, year INTEGER, status TEXT, body TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO notes (slug, title, year, status, body) VALUES
		('dune', 'Dune', 1965, 'published', '# Dune'),
		('bad', NULL, 2025, 'draft', '# Bad')`); err != nil {
		t.Fatalf("seed rows: %v", err)
	}
	chdir(t, dir)
	return dir
}

func TestSQLiteItemGetListCheckAndInspect(t *testing.T) {
	setupSQLiteRepo(t)

	got, _, err := runRoot(t, "item", "get", "--frontmatter", "notes/dune")
	if err != nil {
		t.Fatalf("sqlite item get: %v", err)
	}
	if !strings.Contains(got, "title: Dune") || !strings.Contains(got, "year: 1965") {
		t.Fatalf("frontmatter did not include row metadata:\n%s", got)
	}

	body, _, err := runRoot(t, "item", "get", "--body", "notes/dune")
	if err != nil {
		t.Fatalf("sqlite item get --body: %v", err)
	}
	if body != "# Dune" {
		t.Fatalf("body = %q, want %q", body, "# Dune")
	}

	list, _, err := runRoot(t, "item", "list", "notes", "--filter", "status=published")
	if err != nil {
		t.Fatalf("sqlite item list: %v", err)
	}
	if got := strings.Join(listIDs(list), ","); got != "dune" {
		t.Fatalf("filtered ids = %q, want dune\n%s", got, list)
	}

	stdout, stderr, err := runRoot(t, "check", "notes/dune")
	if err != nil {
		t.Fatalf("sqlite check valid row: %v\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "notes/dune: OK") {
		t.Fatalf("expected OK for sqlite row, got stdout=%q stderr=%q", stdout, stderr)
	}

	_, stderr, err = runRoot(t, "check", "notes/bad")
	if err == nil {
		t.Fatalf("expected sqlite check failure")
	}
	if !strings.Contains(stderr, "missing required field") {
		t.Fatalf("expected missing required field, got: %s", stderr)
	}

	report, _, err := runRoot(t, "inspect", "notes", "--inspector", "object_fields", "-v")
	if err != nil {
		t.Fatalf("sqlite inspect: %v", err)
	}
	if !strings.Contains(report, "title") || !strings.Contains(report, "year") {
		t.Fatalf("inspect report missing sqlite fields:\n%s", report)
	}
}

func TestSQLiteItemAddUpdateDelete(t *testing.T) {
	dir := setupSQLiteRepo(t)

	if _, _, err := runRoot(t, "item", "add", "notes/hobbit", "title=Hobbit", "year=1937"); err != nil {
		t.Fatalf("sqlite item add: %v", err)
	}
	if got := sqliteScalar(t, dir, `SELECT title FROM notes WHERE slug = 'hobbit'`); got != "Hobbit" {
		t.Fatalf("inserted title = %q", got)
	}

	if _, _, err := runRoot(t, "item", "update", "notes/hobbit", "year=1938", "status=published"); err != nil {
		t.Fatalf("sqlite item update: %v", err)
	}
	if got := sqliteScalar(t, dir, `SELECT year FROM notes WHERE slug = 'hobbit'`); got != int64(1938) {
		t.Fatalf("updated year = %#v", got)
	}
	if got := sqliteScalar(t, dir, `SELECT status FROM notes WHERE slug = 'hobbit'`); got != "published" {
		t.Fatalf("updated status = %#v", got)
	}

	if _, _, err := runRoot(t, "item", "delete", "notes/hobbit"); err != nil {
		t.Fatalf("sqlite item delete: %v", err)
	}
	if got := sqliteScalar(t, dir, `SELECT COUNT(1) FROM notes WHERE slug = 'hobbit'`); got != int64(0) {
		t.Fatalf("deleted row count = %#v", got)
	}
}

func TestSQLiteRejectsFilesystemChecksAtLoad(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"storage/db.yaml": `type: sqlite
path: content.sqlite
collections:
  notes:
    table: notes
    id: slug
    checks:
      - kind: filesystem_name_case
        style: kebab
`,
	})
	chdir(t, dir)
	_, _, err := runRoot(t, "item", "list", "notes")
	if err == nil || !strings.Contains(err.Error(), "does not support filesystem check") {
		t.Fatalf("expected unsupported filesystem check error, got: %v", err)
	}
}

func sqliteScalar(t *testing.T, dir, query string) any {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "content.sqlite"))
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

func TestSQLiteStorageFileCreatedBySetup(t *testing.T) {
	dir := setupSQLiteRepo(t)
	if _, err := os.Stat(filepath.Join(dir, "content.sqlite")); err != nil {
		t.Fatalf("expected sqlite db: %v", err)
	}
}
