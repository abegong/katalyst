package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_preparesKatalystDir(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}

	for _, want := range []string{
		".katalyst",
		".katalyst/schemas",
		".katalyst/bases",
		".katalyst/bases/local.yaml",
		".katalyst/config.yaml",
	} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
}

// init prepares the directory only; it must not scaffold example schemas,
// collections, or documents.
func TestInit_writesNoExampleContent(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}

	for _, unwanted := range []string{
		"katalyst.yaml",
		"schemas",
		"notes",
		".katalyst/schemas/book.yaml",
		".katalyst/bases/local/notes.yaml",
	} {
		if _, err := os.Stat(filepath.Join(dir, unwanted)); err == nil {
			t.Errorf("did not expect %s to exist", unwanted)
		}
	}
}

func TestInit_refusesWhenKatalystDirExists(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, ".katalyst")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(existing, "config.yaml")
	if err := os.WriteFile(sentinel, []byte("existing: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runRoot(t, "init", "--dir", dir)
	if err == nil {
		t.Fatalf("expected error when .katalyst/ exists")
	}
	if !strings.Contains(err.Error(), "exists") && !strings.Contains(stderr, "exists") {
		t.Errorf("expected message about pre-existing .katalyst, got err=%v stderr=%q", err, stderr)
	}

	body, _ := os.ReadFile(sentinel)
	if !strings.Contains(string(body), "existing: true") {
		t.Errorf("init clobbered an existing file: %q", body)
	}
}

// A freshly-prepared project must satisfy `fix --check` (nothing to
// format) and `check` (no collections, nothing to validate).
func TestInit_freshProjectIsClean(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	chdir(t, dir)
	if _, _, err := runRoot(t, "fix", "--check"); err != nil {
		t.Errorf("fix --check on a fresh project failed: %v", err)
	}
	if _, stderr, err := runRoot(t, "check"); err != nil {
		t.Fatalf("check on a fresh project failed: %v\nstderr: %s", err, stderr)
	}
}

func TestInit_refusesImplicitNestedProject(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("parent init: %v", err)
	}
	child := filepath.Join(dir, "ongoing", "blog")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := runRoot(t, "init", "--dir", child)
	if err == nil {
		t.Fatalf("expected nested init refusal")
	}
	if got := exitCode(err); got != 2 {
		t.Fatalf("exit code = %d, want 2 (err: %v)", got, err)
	}
	if _, statErr := os.Stat(filepath.Join(child, ".katalyst")); !os.IsNotExist(statErr) {
		t.Fatalf("plain init wrote child config, stat err: %v", statErr)
	}
}

func TestInit_nestedCreatesChildAndPrintsDelegationSnippet(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("parent init: %v", err)
	}
	child := filepath.Join(dir, "ongoing", "blog")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runRoot(t, "init", "--nested", "--dir", child)
	if err != nil {
		t.Fatalf("nested init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(child, ".katalyst", "config.yaml")); err != nil {
		t.Fatalf("expected child config: %v", err)
	}
	for _, want := range []string{
		"nested project created",
		"path: ongoing/blog",
		"collections: file_nearest",
		"collectionChecks: file_nearest",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected stdout to contain %q, got:\n%s", want, stdout)
		}
	}
}
