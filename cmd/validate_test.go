package cmd_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katabase-ai/katabridge/cmd"
)

const testSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["title", "year"],
  "properties": {
    "title": { "type": "string" },
    "year":  { "type": "integer" }
  }
}`

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func runValidate(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := cmd.NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"validate"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestValidateCmd_validFile(t *testing.T) {
	dir := t.TempDir()
	schemaPath := writeFile(t, dir, "schema.json", testSchema)
	mdPath := writeFile(t, dir, "good.md",
		"---\ntitle: Dune\nyear: 1965\n---\n# Body\n")

	stdout, stderr, err := runValidate(t, "--schema", schemaPath, mdPath)
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("expected OK in stdout, got: %q", stdout)
	}
}

func TestValidateCmd_invalidFile_returnsExitCode1(t *testing.T) {
	dir := t.TempDir()
	schemaPath := writeFile(t, dir, "schema.json", testSchema)
	mdPath := writeFile(t, dir, "bad.md",
		"---\ntitle: Dune\n---\n# Body\n") // missing year

	_, stderr, err := runValidate(t, "--schema", schemaPath, mdPath)
	if err == nil {
		t.Fatalf("expected error for invalid file")
	}

	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got err=%v", err)
	}
	if !strings.Contains(stderr, "year") {
		t.Errorf("expected stderr to mention missing 'year', got: %q", stderr)
	}
}

func TestValidateCmd_includesLineNumberWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	schemaPath := writeFile(t, dir, "schema.json", testSchema)
	// year is on line 3 (line 1 = "---", line 2 = "title: Dune",
	// line 3 = "year: not-a-number", line 4 = "---")
	mdPath := writeFile(t, dir, "bad.md",
		"---\ntitle: Dune\nyear: \"not a number\"\n---\n# Body\n")

	_, stderr, err := runValidate(t, "--schema", schemaPath, mdPath)
	if err == nil {
		t.Fatalf("expected validation failure")
	}
	// Format: <path>:<line>: <pointer>: <message>
	if !strings.Contains(stderr, mdPath+":3:") {
		t.Errorf("expected stderr to contain %q with line number 3, got: %q", mdPath, stderr)
	}
}

func TestValidateCmd_missingSchemaFlag(t *testing.T) {
	dir := t.TempDir()
	mdPath := writeFile(t, dir, "x.md", "---\ntitle: x\n---\n")

	_, _, err := runValidate(t, mdPath)
	if err == nil {
		t.Fatalf("expected usage error when --schema omitted")
	}

	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2 (usage), got err=%v", err)
	}
}

func TestValidateCmd_fileWithoutFrontmatter(t *testing.T) {
	dir := t.TempDir()
	schemaPath := writeFile(t, dir, "schema.json", testSchema)
	mdPath := writeFile(t, dir, "no-fm.md", "# Just a heading\n")

	_, stderr, err := runValidate(t, "--schema", schemaPath, mdPath)
	if err == nil {
		t.Fatalf("expected error when file has no frontmatter")
	}
	if !strings.Contains(stderr, "no frontmatter") {
		t.Errorf("expected 'no frontmatter' message, got: %q", stderr)
	}
}
