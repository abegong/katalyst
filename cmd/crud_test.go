package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRead_returnsFileBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.md")
	want := "---\ntitle: Dune\n---\n# Body\n"
	mustWrite(t, path, want)

	stdout, _, err := runRoot(t, "read", path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if stdout != want {
		t.Fatalf("read output mismatch:\n got: %q\nwant: %q", stdout, want)
	}
}

func TestDelete_removesFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "to-delete.md")
	mustWrite(t, path, "x\n")

	if _, _, err := runRoot(t, "delete", path); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err=%v", err)
	}
}

func TestDelete_forceIgnoresMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.md")
	if _, _, err := runRoot(t, "delete", "-f", path); err != nil {
		t.Fatalf("delete -f should ignore missing file: %v", err)
	}
}

func TestDelete_rejectsDirectories(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := runRoot(t, "delete", p)
	if err == nil {
		t.Fatalf("expected delete directory to fail")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Fatalf("expected usage exit code 2, got err=%v", err)
	}
}

func TestCreate_strictValidationRejectsInvalidMarkdownWrite(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	path := filepath.Join(dir, "notes", "new.md")
	_, _, err := runRoot(t, "create", "--schema", strictSchema, path, "title=Dune", "year=1965")
	if err == nil {
		t.Fatalf("expected create to fail strict validation")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Fatalf("expected validation exit code 1, got err=%v", err)
	}
	if !strings.Contains(err.Error(), "isbn") {
		t.Fatalf("expected validation error mentioning isbn, err=%q", err.Error())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected destination to not be written on failure")
	}
}

func TestCreate_noValidateBypassesStrictCheck(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	path := filepath.Join(dir, "notes", "new.md")
	if _, _, err := runRoot(t, "create", "--schema", strictSchema, "--no-validate", path, "title=Dune", "year=1965"); err != nil {
		t.Fatalf("expected create with --no-validate to succeed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected destination to exist: %v", err)
	}
}

func TestCreate_rejectsExistingPathByDefault(t *testing.T) {
	dir := setupScaffoldRepo(t)
	path := filepath.Join(dir, "notes", "existing.md")
	mustWrite(t, path, "---\ntitle: x\nyear: 1\n---\n")
	if _, _, err := runRoot(t, "create", path, "title=changed"); err == nil {
		t.Fatalf("expected create to reject existing path")
	}
}

func TestCreate_schemaResolutionPrecedenceForWriteValidation(t *testing.T) {
	dir := setupScaffoldRepo(t)
	mustWrite(t, filepath.Join(dir, "schemas/strict-book.json"), strictBookSchemaFixture)
	mustWrite(t, filepath.Join(dir, "katabridge.yaml"), strictBookConfigFixture)

	// Config defaults notes/** -> book. Inline schema in create args should win.
	path := filepath.Join(dir, "notes", "new.md")
	_, _, err := runRoot(t, "create", path, "schema=strict-book", "title=Dune", "year=1965")
	if err == nil {
		t.Fatalf("expected create to fail: inline strict-book should win over config rule")
	}
	if !strings.Contains(err.Error(), "isbn") {
		t.Fatalf("expected strict-book validation failure, got: %v", err)
	}

	loose := filepath.Join(dir, "schemas", "loose.json")
	mustWrite(t, loose, `{"type":"object"}`)
	if _, _, err := runRoot(t, "create", "--schema", loose, path, "schema=strict-book", "title=Dune", "year=1965"); err != nil {
		t.Fatalf("--schema should override inline schema during write validation: %v", err)
	}
}

func TestUpdate_strictValidationRejectsInvalidResult(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	path := filepath.Join(dir, "notes", "example.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = runRoot(t, "update", "--schema", strictSchema, path, "title=Changed")
	if err == nil {
		t.Fatalf("expected strict update validation failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Fatalf("expected validation exit code 1, got err=%v", err)
	}
	if !strings.Contains(err.Error(), "isbn") {
		t.Fatalf("expected validation error mentioning isbn, err=%q", err.Error())
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("update modified file despite strict validation failure")
	}
}

func TestUpdate_noValidateAllowsWrite(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	path := filepath.Join(dir, "notes", "example.md")
	if _, _, err := runRoot(t, "update", "--schema", strictSchema, "--no-validate", path, "title=Changed"); err != nil {
		t.Fatalf("expected update --no-validate to succeed: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "title: Changed") {
		t.Fatalf("expected updated title in file, got:\n%s", string(got))
	}
}
