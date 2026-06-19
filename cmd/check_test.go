package cmd_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// setupNotesRepo writes a project with the book schema and a single
// `notes` collection defined by the given collection body, then chdirs in.
// The book schema requires title+year.
func setupNotesRepo(t *testing.T, notesCollection string) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml":            schemaFormatJSON,
		"schemas/book.json":      bookSchemaFixture,
		"collections/notes.yaml": notesCollection,
	})
	chdir(t, dir)
	return dir
}

const objectNotesConfig = `path: notes
schema: book
`

func TestCheck_validItem_OK(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	stdout, _, err := runRoot(t, "check", "notes/dune")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("expected OK, got: %q", stdout)
	}
}

func TestCheck_invalidItem_exit1WithPointer(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	// year is a string → type error at /year on line 3.
	mustWrite(t, filepath.Join(dir, "notes/bad.md"), "---\ntitle: Dune\nyear: \"not a number\"\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/bad")
	if err == nil {
		t.Fatalf("expected validation failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(stderr, "/year") {
		t.Errorf("expected /year pointer in stderr, got: %q", stderr)
	}
	bad := filepath.Join(dir, "notes/bad.md")
	if !strings.Contains(stderr, bad+":3:") {
		t.Errorf("expected line 3 in stderr, got: %q", stderr)
	}
}

func TestCheck_missingRequired_fallsBackToAncestorLine(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/missing.md"), "---\ntitle: Dune\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/missing")
	if err == nil {
		t.Fatalf("expected failure for missing required field")
	}
	if !strings.Contains(stderr, "year") {
		t.Errorf("expected mention of missing 'year', got: %q", stderr)
	}
	// The error is attributed to a known ancestor line (the document has
	// a frontmatter root), so a line number is present.
	missing := filepath.Join(dir, "notes/missing.md")
	if !strings.Contains(stderr, missing+":") {
		t.Errorf("expected a path:line prefix, got: %q", stderr)
	}
}

func TestCheck_wholeProjectWhenNoSelector(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/a.md"), "---\ntitle: A\nyear: 1\n---\n# A\n")
	mustWrite(t, filepath.Join(dir, "notes/b.md"), "---\ntitle: B\nyear: 2\n---\n# B\n")

	stdout, _, err := runRoot(t, "check")
	if err != nil {
		t.Fatalf("check (whole project): %v", err)
	}
	if strings.Count(stdout, "OK") != 2 {
		t.Errorf("expected 2 OK lines, got: %q", stdout)
	}
}

func TestCheck_unmatchedFileInCollectionDir_isError(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/ok.md"), "---\ntitle: Ok\nyear: 1\n---\n# Ok\n")
	// A non-matching file sitting in the collection directory.
	mustWrite(t, filepath.Join(dir, "notes/stray.txt"), "not markdown\n")

	_, stderr, err := runRoot(t, "check", "notes")
	if err == nil {
		t.Fatalf("expected unmatched file to cause failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(stderr, "stray.txt") || !strings.Contains(stderr, "unmatched") {
		t.Errorf("expected unmatched stray.txt error, got: %q", stderr)
	}
}

func TestCheck_unknownSelector_exit2(t *testing.T) {
	setupNotesRepo(t, objectNotesConfig)
	_, _, err := runRoot(t, "check", "ghosts")
	if err == nil {
		t.Fatalf("expected usage error for unknown collection")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestCheck_noConfig_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "check")
	if err == nil {
		t.Fatalf("expected usage error when no config found")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestCheck_schemaFlagOverridesForAllItems(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	// Config would require title+year (book). A loose --schema overrides
	// object validation for every selected item.
	loose := filepath.Join(dir, "schemas/loose.json")
	mustWrite(t, loose, `{"type":"object"}`)
	mustWrite(t, filepath.Join(dir, "notes/missing.md"), "---\ntitle: Missing\n---\n# Missing\n")

	if _, _, err := runRoot(t, "check", "--schema", loose, "notes/missing"); err != nil {
		t.Fatalf("--schema should have overridden config object checks: %v", err)
	}
}

func TestCheck_inlineSchemaKeyTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml":              schemaFormatJSON,
		"schemas/book.json":        bookSchemaFixture,
		"schemas/strict-book.json": strictBookSchemaFixture,
		"collections/notes.yaml":   "path: notes\nschema: book\n",
	})
	chdir(t, dir)

	// Collection maps notes → book, but the doc opts into strict-book,
	// which additionally requires isbn.
	mustWrite(t, filepath.Join(dir, "notes/strict.md"),
		"---\nschema: strict-book\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/strict")
	if err == nil {
		t.Fatalf("expected inline strict-book to fail (missing isbn)")
	}
	if !strings.Contains(stderr, "isbn") {
		t.Errorf("expected isbn in stderr, got: %q", stderr)
	}
}

func TestCheck_markdownAndFilesystemChecks(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"collections/notes.yaml": "path: notes\nchecks:\n  - kind: markdown_title_matches_h1\n    field: title\n",
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: Dune\n---\n# Children of Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/dune")
	if err == nil {
		t.Fatalf("expected markdown title/H1 mismatch")
	}
	if !strings.Contains(stderr, "does not match first H1") {
		t.Errorf("expected title/H1 message, got: %q", stderr)
	}
}
