package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/abegong/katalyst/cmd"
	"github.com/abegong/katalyst/internal/project/projecttest"
)

// runRoot builds a fresh command tree, runs it with args, and captures
// stdout/stderr. A fresh tree per call keeps tests hermetic.
func runRoot(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := cmd.NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	if args == nil {
		args = []string{}
	}
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
// .katalyst/ directory (e.g. "schemas/book.json", "bases/local.yaml",
// "config.yaml"); values are file contents.
func writeProject(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	projecttest.WriteProject(t, dir, files)
}

// baseLocal builds a .katalyst/bases/local.yaml body: a filesystem base rooted
// at the project, declaring the given collections. Each value
// is the collection's YAML body, re-indented under its name. Collections now
// live inside their base, so tests scaffold them this way instead
// of one file per collection.
func baseLocal(collections map[string]string) string {
	return projecttest.LocalBase(collections)
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
		"bases/local.yaml": baseLocal(map[string]string{
			"books":  "path: notes/books\nschema: book\n",
			"people": "path: notes/people\nschema: person\n",
		}),
	})
	return dir
}
