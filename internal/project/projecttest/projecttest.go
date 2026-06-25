// Package projecttest holds shared test helpers for constructing temporary
// Katalyst projects.
package projecttest

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// MinimalSchema is a placeholder schema body for tests that only need the
// config layer to record a schema path.
const MinimalSchema = "type: object\n"

// WriteProject scaffolds a .katalyst/ tree. Keys are paths relative to the
// .katalyst/ directory, such as "schemas/book.yaml" or "config.yaml".
func WriteProject(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".katalyst"), 0o755); err != nil {
		t.Fatalf("mkdir .katalyst: %v", err)
	}
	for rel, content := range files {
		p := filepath.Join(dir, ".katalyst", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// LocalBase builds a .katalyst/bases/local.yaml body with a filesystem base
// rooted at the project and the given collection YAML bodies.
func LocalBase(collections map[string]string) string {
	var b strings.Builder
	b.WriteString("type: filesystem\nroot: .\ncollections:\n")
	names := make([]string, 0, len(collections))
	for n := range collections {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b.WriteString("  " + n + ":\n")
		for _, line := range strings.Split(strings.TrimRight(collections[n], "\n"), "\n") {
			if line == "" {
				b.WriteString("\n")
				continue
			}
			b.WriteString("    " + line + "\n")
		}
	}
	return b.String()
}

// RealPath returns dir with symlinks resolved. macOS's $TMPDIR is
// /var/folders/... which is a symlink to /private/var/folders/...; Load
// canonicalizes via EvalSymlinks, so tests compare against the resolved form.
func RealPath(t *testing.T, dir string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return dir
	}
	return r
}
