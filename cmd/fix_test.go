package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixNotesConfig = `path: notes
checks:
  - kind: markdown_requires_h1
`

func setupFixRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": fixNotesConfig}),
	})
	chdir(t, dir)
	return dir
}

func setupFixRepoWith(t *testing.T, notesConfig string) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": notesConfig}),
	})
	chdir(t, dir)
	return dir
}

func TestFix_textForbidsFix_rewritesOnlyMatch(t *testing.T) {
	dir := setupFixRepoWith(t, `path: notes
checks:
  - kind: text_forbids
    target: first-line
    pattern: '\.(\s*)$'
    fix: '$1'
`)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nt: 1\n---\n# Title.\nkeep this.\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	// First line loses its period; the later "keep this." line is untouched.
	want := "---\nt: 1\n---\n# Title\nkeep this.\n"
	if string(got) != want {
		t.Errorf("after fix:\n got: %q\nwant: %q", got, want)
	}
}

func TestFix_textForbidsFix_badTemplateFails(t *testing.T) {
	dir := setupFixRepoWith(t, `path: notes
checks:
  - kind: text_forbids
    pattern: TODO
    fix: TODO-DONE
`)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nt: 1\n---\nhas TODO here\n")

	_, stderr, err := runRoot(t, "fix", "notes/doc")
	if err == nil {
		t.Fatal("expected fix to fail on a template that does not resolve the violation")
	}
	if !strings.Contains(stderr, "fix did not resolve the violation") {
		t.Errorf("expected re-check failure message, got stderr: %q", stderr)
	}
	got, _ := os.ReadFile(p)
	if !strings.Contains(string(got), "has TODO here") {
		t.Errorf("file must be untouched on failure, got %q", got)
	}
}

func TestFix_textForbidsWithoutFix_preservesBody(t *testing.T) {
	dir := setupFixRepoWith(t, `path: notes
checks:
  - kind: text_forbids
    pattern: TODO
`)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\n# Body TODO\nkeep\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	// Frontmatter is still canonicalized; the body (TODO and all) is verbatim.
	want := "---\napple: 2\nzebra: 1\n---\n# Body TODO\nkeep\n"
	if string(got) != want {
		t.Errorf("after fix:\n got: %q\nwant: %q", got, want)
	}
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
