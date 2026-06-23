package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/abegong/katalyst/cmd"
)

// runRoot builds a fresh command tree, runs it with args, and captures
// stdout/stderr. A fresh tree per call keeps tests hermetic.
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

// mustWrite writes content to path, creating parent directories.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeFile writes content to dir/name and returns the full path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	mustWrite(t, p, content)
	return p
}

// chdir switches into dir for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

// schemaFormatJSON is a config.yaml that tells the schema loader to scan
// for *.json files. The shared schema fixtures are JSON, so test projects
// opt into the JSON format rather than the default YAML.
const schemaFormatJSON = "schemas:\n  format: json\n"

// writeProject scaffolds a .katalyst/ tree. Keys are paths relative to the
// .katalyst/ directory (e.g. "schemas/book.json", "storage/local.yaml",
// "config.yaml"); values are file contents.
func writeProject(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		mustWrite(t, filepath.Join(dir, ".katalyst", rel), content)
	}
}

// storageLocal builds a .katalyst/storage/local.yaml body: a filesystem
// instance rooted at the project, declaring the given collections. Each value
// is the collection's YAML body, re-indented under its name. Collections now
// live inside their storage instance, so tests scaffold them this way instead
// of one file per collection.
func storageLocal(collections map[string]string) string {
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

// writeConfigDir writes the two-schema book-and-person project (book and
// person schemas, books and people collections) into a fresh temp dir,
// returning the dir.
func writeConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml":         schemaFormatJSON,
		"schemas/book.json":   bookSchemaFixture,
		"schemas/person.json": personSchemaFixture,
		"storage/local.yaml": storageLocal(map[string]string{
			"books":  "path: notes/books\nschema: book\n",
			"people": "path: notes/people\nschema: person\n",
		}),
	})
	return dir
}
