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
	mustWrite(t, filepath.Join(dir, "schemas/strict-book.json"), strictBookSchemaFixture)
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), strictBookConfigFixture)

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
	mustWrite(t, docPath, "---\nslug: missing-required\ntitle: Missing Required\n---\n# Missing Required\n")

	// Config would apply object schema `book` (title+year required), but
	// --schema overrides object checks while leaving markdown/filesystem
	// checks active.
	if _, _, err := runRoot(t, "validate", "--schema", loosePath, docPath); err != nil {
		t.Fatalf("--schema should have overridden config rules: %v", err)
	}

	nonObjectFailure := filepath.Join(dir, "notes/non-object-fail.md")
	mustWrite(t, nonObjectFailure, "---\nslug: wrong-slug\ntitle: Non Object Fail\n---\n# Non Object Fail\n")
	_, stderr, err := runRoot(t, "validate", "--schema", loosePath, nonObjectFailure)
	if err == nil {
		t.Fatalf("expected non-object checks to still fail with --schema")
	}
	if !strings.Contains(stderr, "slug") {
		t.Fatalf("expected slug mismatch in stderr, got: %q", stderr)
	}
}

func TestValidate_objectCheck_reportsTypeError(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), objectCheckConfigFixture)
	mustWrite(t, filepath.Join(dir, "schemas/book.json"), bookSchemaFixture)
	chdir(t, dir)

	docPath := filepath.Join(dir, "notes/bad.md")
	mustWrite(t, docPath, "---\ntitle: Dune\nyear: not-a-number\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected object check failure")
	}
	if !strings.Contains(stderr, "/year") {
		t.Fatalf("expected /year in stderr, got: %q", stderr)
	}
}

func TestValidate_markdownCheck_reportsTitleMismatch(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), markdownCheckConfigFixture)
	chdir(t, dir)

	docPath := filepath.Join(dir, "notes/dune.md")
	mustWrite(t, docPath, "---\ntitle: Dune\n---\n# Children of Dune\n")

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected markdown check failure")
	}
	if !strings.Contains(stderr, "does not match first H1") {
		t.Fatalf("expected title/H1 mismatch message, got: %q", stderr)
	}
}

func TestValidate_filesystemCheck_reportsSlugMismatch(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), filesystemCheckConfigFixture)
	chdir(t, dir)

	docPath := filepath.Join(dir, "notes/dune.md")
	mustWrite(t, docPath, "---\nslug: dune-messiah\n---\n# Dune Messiah\n")

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected filesystem check failure")
	}
	if !strings.Contains(stderr, "must match filename") {
		t.Fatalf("expected slug/filename mismatch message, got: %q", stderr)
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
