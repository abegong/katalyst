package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupItemRepo creates a repo with a single `notes` collection backed by
// the book object schema (title+year required), and chdirs in.
func setupItemRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml":              schemaFormatJSON,
		"schemas/book.json":        bookSchemaFixture,
		"schemas/strict-book.json": strictBookSchemaFixture,
		"collections/notes.yaml":   objectNotesConfig,
	})
	chdir(t, dir)
	return dir
}

func TestItemAdd_writesFrontmatterAndEmptyBody(t *testing.T) {
	dir := setupItemRepo(t)
	if _, _, err := runRoot(t, "item", "add", "notes/dune", "title=Dune", "year=1965"); err != nil {
		t.Fatalf("item add: %v", err)
	}
	p := filepath.Join(dir, "notes/dune.md")
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("expected file written: %v", err)
	}
	// YAML-scalar typing: year is an integer, not a quoted string.
	if !strings.Contains(string(got), "year: 1965") {
		t.Errorf("expected integer year, got:\n%s", got)
	}
	// Empty body: nothing after the closing fence.
	if !strings.HasSuffix(string(got), "---\n") {
		t.Errorf("expected empty body after frontmatter, got:\n%s", got)
	}
}

func TestItemAdd_refusesExisting_exit2(t *testing.T) {
	dir := setupItemRepo(t)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: x\nyear: 1\n---\n")
	_, _, err := runRoot(t, "item", "add", "notes/dune", "title=Changed", "year=2")
	if err == nil {
		t.Fatalf("expected refuse-overwrite error")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestItemAdd_validationFailureWritesNothing_exit1(t *testing.T) {
	dir := setupItemRepo(t)
	strict := filepath.Join(dir, ".katalyst/schemas/strict-book.json")
	_, _, err := runRoot(t, "item", "add", "--schema", strict, "notes/dune", "title=Dune", "year=1965")
	if err == nil {
		t.Fatalf("expected strict validation failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(err.Error(), "isbn") {
		t.Errorf("expected isbn in error, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes/dune.md")); !os.IsNotExist(err) {
		t.Errorf("expected nothing written on validation failure")
	}
}

func TestItemAdd_noValidateBypasses(t *testing.T) {
	dir := setupItemRepo(t)
	strict := filepath.Join(dir, ".katalyst/schemas/strict-book.json")
	if _, _, err := runRoot(t, "item", "add", "--schema", strict, "--no-validate", "notes/dune", "title=Dune"); err != nil {
		t.Fatalf("expected --no-validate to succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes/dune.md")); err != nil {
		t.Errorf("expected file written: %v", err)
	}
}

func TestItemGet_defaultPrintsFrontmatterAndBody(t *testing.T) {
	dir := setupItemRepo(t)
	content := "---\ntitle: Dune\nyear: 1965\n---\n# Dune\nbody\n"
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), content)

	stdout, _, err := runRoot(t, "item", "get", "notes/dune")
	if err != nil {
		t.Fatalf("item get: %v", err)
	}
	if stdout != content {
		t.Errorf("default get mismatch:\n got: %q\nwant: %q", stdout, content)
	}
}

func TestItemGet_frontmatterAndBodyFlags(t *testing.T) {
	dir := setupItemRepo(t)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\nbody\n")

	fm, _, err := runRoot(t, "item", "get", "--frontmatter", "notes/dune")
	if err != nil {
		t.Fatalf("get --frontmatter: %v", err)
	}
	if !strings.Contains(fm, "title: Dune") || strings.Contains(fm, "# Dune") {
		t.Errorf("--frontmatter should print only frontmatter, got: %q", fm)
	}

	body, _, err := runRoot(t, "item", "get", "--body", "notes/dune")
	if err != nil {
		t.Fatalf("get --body: %v", err)
	}
	if !strings.Contains(body, "# Dune") || strings.Contains(body, "title:") {
		t.Errorf("--body should print only body, got: %q", body)
	}
}

func TestItemGet_unknownItem_exit2(t *testing.T) {
	setupItemRepo(t)
	_, _, err := runRoot(t, "item", "get", "notes/ghost")
	if err == nil {
		t.Fatalf("expected error for unknown item")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestItemUpdate_mergesKeysBodyUntouched(t *testing.T) {
	dir := setupItemRepo(t)
	p := filepath.Join(dir, "notes/dune.md")
	mustWrite(t, p, "---\ntitle: Dune\nyear: 1965\n---\n# Dune\noriginal body\n")

	if _, _, err := runRoot(t, "item", "update", "notes/dune", "year=1969"); err != nil {
		t.Fatalf("item update: %v", err)
	}
	got, _ := os.ReadFile(p)
	if !strings.Contains(string(got), "year: 1969") {
		t.Errorf("expected updated year, got:\n%s", got)
	}
	if !strings.Contains(string(got), "original body") {
		t.Errorf("expected body untouched, got:\n%s", got)
	}
}

func TestItemUpdate_strictFailureLeavesFileUnchanged(t *testing.T) {
	dir := setupItemRepo(t)
	p := filepath.Join(dir, "notes/dune.md")
	before := "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n"
	mustWrite(t, p, before)

	strict := filepath.Join(dir, ".katalyst/schemas/strict-book.json")
	_, _, err := runRoot(t, "item", "update", "--schema", strict, "notes/dune", "title=Changed")
	if err == nil {
		t.Fatalf("expected strict update failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	after, _ := os.ReadFile(p)
	if string(after) != before {
		t.Errorf("file modified despite validation failure:\n%s", after)
	}
}

func TestItemDelete_removesOneAndMany(t *testing.T) {
	dir := setupItemRepo(t)
	a := filepath.Join(dir, "notes/a.md")
	b := filepath.Join(dir, "notes/b.md")
	mustWrite(t, a, "---\ntitle: A\nyear: 1\n---\n")
	mustWrite(t, b, "---\ntitle: B\nyear: 2\n---\n")

	if _, _, err := runRoot(t, "item", "delete", "notes/a", "notes/b"); err != nil {
		t.Fatalf("item delete: %v", err)
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected %s removed", p)
		}
	}
}

func TestItemDelete_missing_exit2(t *testing.T) {
	setupItemRepo(t)
	_, _, err := runRoot(t, "item", "delete", "notes/ghost")
	if err == nil {
		t.Fatalf("expected error for missing item")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestItemList_showsIdsAndStatus(t *testing.T) {
	dir := setupItemRepo(t)
	mustWrite(t, filepath.Join(dir, "notes/good.md"), "---\ntitle: Good\nyear: 1\n---\n# Good\n")
	mustWrite(t, filepath.Join(dir, "notes/bad.md"), "---\ntitle: Bad\n---\n# Bad\n") // missing year

	stdout, _, err := runRoot(t, "item", "list", "notes")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	if !strings.Contains(stdout, "good") || !strings.Contains(stdout, "ok") {
		t.Errorf("expected good ok, got: %q", stdout)
	}
	if !strings.Contains(stdout, "bad") || !strings.Contains(stdout, "error") {
		t.Errorf("expected bad error status, got: %q", stdout)
	}
}

func TestItemList_wrongDepth_exit2(t *testing.T) {
	setupItemRepo(t)
	_, _, err := runRoot(t, "item", "list", "notes/dune")
	if err == nil {
		t.Fatalf("expected wrong-depth usage error")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}
