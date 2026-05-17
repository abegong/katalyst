package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMkdirMvRm_basicFilesystemFlow(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "src")
	dstDir := filepath.Join(dir, "dst")
	file := filepath.Join(srcDir, "a.txt")
	moved := filepath.Join(dstDir, "a.txt")

	if _, _, err := runRoot(t, "mkdir", "-p", srcDir, dstDir); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runRoot(t, "mv", file, moved); err != nil {
		t.Fatalf("mv: %v", err)
	}
	if _, err := os.Stat(moved); err != nil {
		t.Fatalf("expected moved file to exist: %v", err)
	}

	if _, _, err := runRoot(t, "rm", moved); err != nil {
		t.Fatalf("rm file: %v", err)
	}
	if _, err := os.Stat(moved); !os.IsNotExist(err) {
		t.Fatalf("expected moved file removed, stat err=%v", err)
	}
}

func TestRm_requiresRecursiveForDirectories(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sub")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := runRoot(t, "rm", target)
	if err == nil {
		t.Fatalf("expected rm directory without -r to fail")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Fatalf("expected usage exit code 2, got err=%v", err)
	}

	if _, _, err := runRoot(t, "rm", "-r", target); err != nil {
		t.Fatalf("rm -r: %v", err)
	}
}

func TestCp_strictValidationRejectsInvalidMarkdownWrite(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	src := filepath.Join(dir, "notes", "src.md")
	dst := filepath.Join(dir, "notes", "dst.md")
	mustWrite(t, src, "---\ntitle: Dune\nyear: 1965\n---\n# Body\n")

	_, _, err := runRoot(t, "cp", "--schema", strictSchema, src, dst)
	if err == nil {
		t.Fatalf("expected cp to fail strict validation")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Fatalf("expected validation exit code 1, got err=%v", err)
	}
	if !strings.Contains(err.Error(), "isbn") {
		t.Fatalf("expected validation error mentioning isbn, err=%q", err.Error())
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("expected destination to not be written on failure")
	}
}

func TestCp_noValidateBypassesStrictCheck(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	src := filepath.Join(dir, "notes", "src.md")
	dst := filepath.Join(dir, "notes", "dst.md")
	mustWrite(t, src, "---\ntitle: Dune\nyear: 1965\n---\n# Body\n")

	if _, _, err := runRoot(t, "cp", "--schema", strictSchema, "--no-validate", src, dst); err != nil {
		t.Fatalf("expected cp with --no-validate to succeed: %v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("expected destination to exist: %v", err)
	}
}

func TestCp_schemaResolutionPrecedenceForWriteValidation(t *testing.T) {
	dir := setupScaffoldRepo(t)
	mustWrite(t, filepath.Join(dir, "schemas/strict-book.json"), strictBookSchemaFixture)
	mustWrite(t, filepath.Join(dir, "katabridge.yaml"), strictBookConfigFixture)

	// Config defaults notes/** -> book, but inline schema should win.
	src := filepath.Join(dir, "notes", "src.md")
	dst := filepath.Join(dir, "notes", "dst.md")
	mustWrite(t, src, `---
schema: strict-book
title: Dune
year: 1965
---
# Body
`)

	_, _, err := runRoot(t, "cp", src, dst)
	if err == nil {
		t.Fatalf("expected cp to fail: inline strict-book should win over config rule")
	}
	if !strings.Contains(err.Error(), "isbn") {
		t.Fatalf("expected strict-book validation failure, got: %v", err)
	}

	loose := filepath.Join(dir, "schemas", "loose.json")
	mustWrite(t, loose, `{"type":"object"}`)
	if _, _, err := runRoot(t, "cp", "--schema", loose, src, dst); err != nil {
		t.Fatalf("--schema should override inline schema during write validation: %v", err)
	}
}

func TestSet_strictValidationRejectsInvalidResult(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	path := filepath.Join(dir, "notes", "example.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = runRoot(t, "set", "--schema", strictSchema, path, "title=Changed")
	if err == nil {
		t.Fatalf("expected strict set validation failure")
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
		t.Fatalf("set modified file despite strict validation failure")
	}
}

func TestSet_noValidateAllowsWrite(t *testing.T) {
	dir := setupScaffoldRepo(t)
	strictSchema := filepath.Join(dir, "schemas/strict-book.json")
	mustWrite(t, strictSchema, strictBookSchemaFixture)

	path := filepath.Join(dir, "notes", "example.md")
	if _, _, err := runRoot(t, "set", "--schema", strictSchema, "--no-validate", path, "title=Changed"); err != nil {
		t.Fatalf("expected set --no-validate to succeed: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "title: Changed") {
		t.Fatalf("expected updated title in file, got:\n%s", string(got))
	}
}
