package cmd_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectPlan_noNestedConfigs(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	chdir(t, dir)

	stdout, stderr, err := runRoot(t, "project", "plan")
	if err != nil {
		t.Fatalf("project plan: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "nested configs: none") {
		t.Fatalf("expected no nested configs, got:\n%s", stdout)
	}
}

func TestProjectPlan_printsDelegatedAuthority(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        filesystemChecks: compose
        schemas: file_nearest
`,
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	chdir(t, dir)

	stdout, stderr, err := runRoot(t, "project", "plan")
	if err != nil {
		t.Fatalf("project plan: %v\nstderr: %s", err, stderr)
	}
	for _, want := range []string{
		"ongoing/blog/.katalyst",
		"status: active",
		"collections: file_nearest",
		"collectionChecks: file_nearest",
		"filesystemChecks: compose",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected stdout to contain %q, got:\n%s", want, stdout)
		}
	}
}
