package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katabase-ai/katabridge/cmd"
)

func runRoot(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := cmd.NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestInit_scaffoldsConfigSchemaAndExample(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runRoot(t, "init", "--dir", dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	for _, want := range []string{
		"katabridge.yaml",
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
	cfg := filepath.Join(dir, "katabridge.yaml")
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

// The scaffold must already be in `katabridge fmt` canonical form,
// otherwise a brand-new repo fails `fmt --check` in CI, which is a
// nasty first-impression.
func TestInit_scaffoldIsCanonical(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, _, err := runRoot(t, "fmt", "--check", filepath.Join(dir, "notes/example.md")); err != nil {
		t.Errorf("scaffolded example.md is not in fmt canonical form: %v", err)
	}
}

// TestInit_scaffoldValidatesCleanly_viaExplicitSchema asserts the
// scaffold is internally consistent (the example doc satisfies the
// example schema) using --schema. A stronger version that also
// exercises config discovery lives in validate_config_test.go once
// `validate` learns to read the config.
func TestInit_scaffoldValidatesCleanly_viaExplicitSchema(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}

	schema := filepath.Join(dir, "schemas/book.json")
	doc := filepath.Join(dir, "notes/example.md")

	_, stderr, err := runRoot(t, "validate", "--schema", schema, doc)
	if err != nil {
		t.Fatalf("validate on scaffolded example failed: %v\nstderr: %s", err, stderr)
	}
}
