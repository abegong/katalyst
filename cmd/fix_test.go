package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixNotesConfig = `schemas: {}
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
`

func setupFixRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), fixNotesConfig)
	chdir(t, dir)
	return dir
}

func TestFix_normalizesAndPreservesBody(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\n# Body\nverbatim\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	want := "---\napple: 2\nzebra: 1\n---\n# Body\nverbatim\n"
	if string(got) != want {
		t.Errorf("after fix:\n got: %q\nwant: %q", got, want)
	}
}

func TestFix_isIdempotent(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\n# Body\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("first fix: %v", err)
	}
	first, _ := os.ReadFile(p)
	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("second fix: %v", err)
	}
	second, _ := os.ReadFile(p)
	if string(first) != string(second) {
		t.Errorf("fix not idempotent:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestFix_checkFlag_reportsWithoutWriting(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	original := "---\nzebra: 1\napple: 2\n---\n# Body\n"
	mustWrite(t, p, original)

	stdout, _, err := runRoot(t, "fix", "--check", "notes/doc")
	if err == nil {
		t.Fatalf("expected --check to exit non-zero when a change is pending")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(stdout, "doc.md") {
		t.Errorf("expected the would-change path in stdout, got: %q", stdout)
	}
	got, _ := os.ReadFile(p)
	if string(got) != original {
		t.Errorf("--check modified the file: %q", got)
	}
}

func TestFix_checkFlag_cleanExitsZero(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\napple: 2\nzebra: 1\n---\n# Body\n")

	if _, _, err := runRoot(t, "fix", "--check", "notes/doc"); err != nil {
		t.Fatalf("--check on canonical file should exit 0: %v", err)
	}
}

// D3 guardrail: fix normalizes but never injects a value for a missing
// required key.
func TestFix_neverInjectsMissingKeys(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\ntitle: Dune\n---\n# Dune\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), "year") {
		t.Errorf("fix injected a missing key; content: %q", got)
	}
}

func TestFix_wholeProjectAndReportsChangedOnly(t *testing.T) {
	dir := setupFixRepo(t)
	dirty := filepath.Join(dir, "notes/dirty.md")
	clean := filepath.Join(dir, "notes/clean.md")
	mustWrite(t, dirty, "---\nz: 1\na: 2\n---\n# Body\n")
	mustWrite(t, clean, "---\na: 2\nz: 1\n---\n# Body\n")

	stdout, _, err := runRoot(t, "fix")
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	if !strings.Contains(stdout, "dirty.md") {
		t.Errorf("expected changed file listed, got: %q", stdout)
	}
	if strings.Contains(stdout, "clean.md") {
		t.Errorf("did not expect unchanged file listed, got: %q", stdout)
	}
}
