package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katabridge.yaml"), bookAndPersonConfigFixture)
	mustWrite(t, filepath.Join(dir, "schemas/book.json"), bookSchemaFixture)
	mustWrite(t, filepath.Join(dir, "schemas/person.json"), personSchemaFixture)
	return dir
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestSchemaList_printsSortedNamesAndPaths(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)

	stdout, _, err := runRoot(t, "schema", "list")
	if err != nil {
		t.Fatalf("schema list: %v", err)
	}

	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d:\n%s", len(lines), stdout)
	}
	if !strings.HasPrefix(lines[0], "book") {
		t.Errorf("expected first line to start with 'book', got: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "person") {
		t.Errorf("expected second line to start with 'person', got: %q", lines[1])
	}
	if !strings.Contains(lines[0], "schemas/book.json") {
		t.Errorf("expected first line to contain path, got: %q", lines[0])
	}
}

func TestSchemaList_noConfig(t *testing.T) {
	chdir(t, t.TempDir())

	_, _, err := runRoot(t, "schema", "list")
	if err == nil {
		t.Fatalf("expected error when no config found")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2 (usage), got: %v", err)
	}
}

func TestSchemaShow_printsSchemaContents(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)

	stdout, _, err := runRoot(t, "schema", "show", "book")
	if err != nil {
		t.Fatalf("schema show: %v", err)
	}
	if !strings.Contains(stdout, `"title": "book"`) && !strings.Contains(stdout, `"title":"book"`) {
		t.Errorf("expected schema contents in output, got: %q", stdout)
	}
}

func TestSchemaShow_unknownName(t *testing.T) {
	dir := writeConfigDir(t)
	chdir(t, dir)

	_, _, err := runRoot(t, "schema", "show", "nonexistent")
	if err == nil {
		t.Fatalf("expected error for unknown schema")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention the missing name, got: %v", err)
	}
}
