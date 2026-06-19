package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_scaffoldsConfigSchemaAndExample(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runRoot(t, "init", "--dir", dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	for _, want := range []string{
		"katalyst.yaml",
		"schemas/book.json",
		"notes/example.md",
	} {
		p := filepath.Join(dir, want)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
}

func TestInit_refusesToOverwrite(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "katalyst.yaml")
	if err := os.WriteFile(cfg, []byte("existing: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runRoot(t, "init", "--dir", dir)
	if err == nil {
		t.Fatalf("expected error when config exists")
	}
	if !strings.Contains(err.Error(), "exists") && !strings.Contains(stderr, "exists") {
		t.Errorf("expected message about pre-existing config, got err=%v stderr=%q", err, stderr)
	}

	body, _ := os.ReadFile(cfg)
	if !strings.Contains(string(body), "existing: true") {
		t.Errorf("init clobbered an existing file: %q", body)
	}
}

// The scaffold must already be in `katalyst fix` canonical form, otherwise
// a brand-new repo fails `fix --check` in CI.
func TestInit_scaffoldIsCanonical(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	chdir(t, dir)
	if _, _, err := runRoot(t, "fix", "--check"); err != nil {
		t.Errorf("scaffolded project is not in fix canonical form: %v", err)
	}
}

// The scaffold is internally consistent: the example item satisfies the
// configured checks, discovered via the scaffolded katalyst.yaml.
func TestInit_scaffoldChecksCleanly(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	chdir(t, dir)
	if _, stderr, err := runRoot(t, "check"); err != nil {
		t.Fatalf("check on scaffolded project failed: %v\nstderr: %s", err, stderr)
	}
}
