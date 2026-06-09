package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katabase-ai/katalyst/internal/config"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	p := filepath.Join(dir, "katalyst.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

// realPath returns dir with symlinks resolved. macOS's $TMPDIR is
// /var/folders/... which is a symlink to /private/var/folders/...;
// Load canonicalizes via EvalSymlinks, so tests must compare against
// the resolved form.
func realPath(t *testing.T, dir string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return dir
	}
	return r
}

func TestLoad_parsesSchemasAndRules(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `schemas:
  book:   ./schemas/book.json
  person: ./schemas/person.json
rules:
  - paths: "notes/books/**/*.md"
    schema: book
  - paths: "notes/people/**/*.md"
    schema: person
`)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	wantRoot := realPath(t, dir)
	if cfg.Root != wantRoot {
		t.Errorf("Root = %q, want %q", cfg.Root, wantRoot)
	}
	if len(cfg.Schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(cfg.Schemas))
	}
	if got := cfg.SchemaPath("book"); got != filepath.Join(wantRoot, "schemas/book.json") {
		t.Errorf("SchemaPath(book) = %q", got)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].Schema != "book" {
		t.Errorf("rule[0].Schema = %q, want book", cfg.Rules[0].Schema)
	}
}

func TestLoad_ascendsToFindConfig(t *testing.T) {
	repo := t.TempDir()
	writeConfig(t, repo, "schemas: {}\nrules: []\n")
	deep := filepath.Join(repo, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(deep)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantRoot := realPath(t, repo)
	if cfg.Root != wantRoot {
		t.Errorf("Root = %q, want %q", cfg.Root, wantRoot)
	}
}

func TestLoad_notFound(t *testing.T) {
	dir := t.TempDir()
	_, err := config.Load(dir)
	if !errors.Is(err, config.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLoad_rejectsUnknownSchemaInRule(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `schemas:
  book: ./schemas/book.json
rules:
  - paths: "**/*.md"
    schema: nonexistent
`)
	_, err := config.Load(dir)
	if err == nil {
		t.Fatalf("expected error for rule referencing unknown schema")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the bad name: %v", err)
	}
}

func TestMatch_firstMatchingRuleWins(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `schemas:
  book:    ./schemas/book.json
  generic: ./schemas/generic.json
rules:
  - paths: "notes/books/**/*.md"
    schema: book
  - paths: "**/*.md"
    schema: generic
`)
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		path string
		want string
	}{
		{filepath.Join(dir, "notes/books/dune.md"), "book"},
		{filepath.Join(dir, "notes/random.md"), "generic"},
		{filepath.Join(dir, "elsewhere/x.md"), "generic"},
	}
	for _, tc := range cases {
		got, ok := cfg.Match(tc.path)
		if !ok {
			t.Errorf("Match(%q) returned no match", tc.path)
			continue
		}
		if got != tc.want {
			t.Errorf("Match(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestMatch_noMatch(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `schemas:
  book: ./schemas/book.json
rules:
  - paths: "books/**/*.md"
    schema: book
`)
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Match(filepath.Join(dir, "notes/whatever.md")); ok {
		t.Errorf("expected no match")
	}
}

func TestList_returnsSortedNames(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `schemas:
  zebra:  ./z.json
  apple:  ./a.json
  middle: ./m.json
rules: []
`)
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.SchemaNames()
	want := []string{"apple", "middle", "zebra"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("SchemaNames = %v, want %v", got, want)
	}
}
