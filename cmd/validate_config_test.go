package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Set up an init-scaffolded repo and chdir into it.
func setupScaffoldRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	chdir(t, dir)
	return dir
}

func TestValidate_usesConfigWhenSchemaFlagOmitted(t *testing.T) {
	dir := setupScaffoldRepo(t)

	_, stderr, err := runRoot(t, "validate", filepath.Join(dir, "notes/example.md"))
	if err != nil {
		t.Fatalf("validate via config failed: %v\nstderr: %s", err, stderr)
	}
}

func TestValidate_inlineSchemaKeyTakesPrecedence(t *testing.T) {
	dir := setupScaffoldRepo(t)

	// Add a second schema and a doc that asks for it inline. The config
	// rules (`notes/**` -> book) would otherwise apply.
	strict := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["title", "year", "isbn"],
  "properties": {
    "title": { "type": "string" },
    "year":  { "type": "integer" },
    "isbn":  { "type": "string", "pattern": "^[0-9-]+$" }
  }
}`
	mustWrite(t, filepath.Join(dir, "schemas/strict-book.json"), strict)
	mustWrite(t, filepath.Join(dir, "katabridge.yaml"), `schemas:
  book:        ./schemas/book.json
  strict-book: ./schemas/strict-book.json
rules:
  - paths: "notes/**/*.md"
    schema: book
`)

	docPath := filepath.Join(dir, "notes/strict.md")
	mustWrite(t, docPath, `---
schema: strict-book
title: Dune
year: 1965
---
# Body
`)

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected validation failure (missing isbn under strict-book)")
	}
	if !strings.Contains(stderr, "isbn") {
		t.Errorf("expected stderr to mention 'isbn', got: %q", stderr)
	}
}

func TestValidate_unmatchedFileIsError(t *testing.T) {
	dir := setupScaffoldRepo(t)

	// `elsewhere/` is outside the `notes/**` rule. No inline schema.
	outsider := filepath.Join(dir, "elsewhere/random.md")
	mustWrite(t, outsider, "---\ntitle: x\nyear: 1\n---\n# Body\n")

	_, stderr, err := runRoot(t, "validate", outsider)
	if err == nil {
		t.Fatalf("expected error for unmatched file")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(stderr, "no schema") && !strings.Contains(stderr, "unmatched") {
		t.Errorf("expected stderr to explain why the file was unmatched, got: %q", stderr)
	}
}

func TestValidate_schemaFlagWinsOverConfig(t *testing.T) {
	dir := setupScaffoldRepo(t)

	loose := `{"type":"object"}`
	loosePath := filepath.Join(dir, "schemas/loose.json")
	mustWrite(t, loosePath, loose)

	docPath := filepath.Join(dir, "notes/missing-required.md")
	mustWrite(t, docPath, "---\nfoo: bar\n---\n# Body\n")

	// Config would apply `book` which requires title+year, but --schema
	// trumps it and `loose` accepts anything.
	if _, _, err := runRoot(t, "validate", "--schema", loosePath, docPath); err != nil {
		t.Fatalf("--schema should have overridden config rules: %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

