package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/katabase-ai/katalyst/cmd"
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

// setupScaffoldRepo runs `init` into a temp dir and chdirs into it.
func setupScaffoldRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	chdir(t, dir)
	return dir
}

// writeConfigDir writes the two-schema book-and-person config and its
// schemas into a fresh temp dir, returning the dir.
func writeConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), bookAndPersonConfigFixture)
	mustWrite(t, filepath.Join(dir, "schemas/book.json"), bookSchemaFixture)
	mustWrite(t, filepath.Join(dir, "schemas/person.json"), personSchemaFixture)
	return dir
}
