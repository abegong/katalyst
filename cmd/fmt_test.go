package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFmt_writesNormalizedContent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\nbody\n")

	_, _, err := runRoot(t, "fmt", p)
	if err != nil {
		t.Fatalf("fmt: %v", err)
	}

	got, _ := os.ReadFile(p)
	want := "---\napple: 2\nzebra: 1\n---\nbody\n"
	if string(got) != want {
		t.Errorf("file content after fmt:\n got: %q\nwant: %q", got, want)
	}
}

func TestFmt_isIdempotent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.md")
	mustWrite(t, p, "---\napple: 2\nzebra: 1\n---\nbody\n")

	if _, _, err := runRoot(t, "fmt", p); err != nil {
		t.Fatalf("first fmt: %v", err)
	}
	first, _ := os.ReadFile(p)

	if _, _, err := runRoot(t, "fmt", p); err != nil {
		t.Fatalf("second fmt: %v", err)
	}
	second, _ := os.ReadFile(p)

	if string(first) != string(second) {
		t.Errorf("fmt is not idempotent.\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestFmt_checkFlag_reportsDiffWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.md")
	original := "---\nzebra: 1\napple: 2\n---\nbody\n"
	mustWrite(t, p, original)

	_, _, err := runRoot(t, "fmt", "--check", p)
	if err == nil {
		t.Fatalf("expected --check to fail when file needs formatting")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}

	got, _ := os.ReadFile(p)
	if string(got) != original {
		t.Errorf("--check rewrote the file; content changed to: %q", got)
	}
}

func TestFmt_checkFlag_succeedsOnAlreadyFormatted(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "doc.md")
	mustWrite(t, p, "---\napple: 2\nzebra: 1\n---\nbody\n")

	if _, _, err := runRoot(t, "fmt", "--check", p); err != nil {
		t.Fatalf("--check on formatted file should succeed: %v", err)
	}
}

func TestFmt_reportsListOfChangedFiles(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.md")
	p2 := filepath.Join(dir, "b.md")
	mustWrite(t, p1, "---\nz: 1\na: 2\n---\nbody\n")
	mustWrite(t, p2, "---\nalready: ok\n---\nbody\n")

	stdout, _, err := runRoot(t, "fmt", p1, p2)
	if err != nil {
		t.Fatalf("fmt: %v", err)
	}
	if !strings.Contains(stdout, p1) {
		t.Errorf("expected stdout to list changed file %s, got: %q", p1, stdout)
	}
	if strings.Contains(stdout, p2) {
		t.Errorf("expected stdout to NOT list unchanged file %s, got: %q", p2, stdout)
	}
}
