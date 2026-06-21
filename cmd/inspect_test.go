package cmd_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inspectRepo scaffolds a small markdown corpus and returns its directory.
func inspectRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "books/dune.md", "---\ntitle: Dune\nstatus: read\n---\n# Dune\n## Review\n")
	writeFile(t, dir, "books/it.md", "---\ntitle: It\nstatus: reading\n---\n# It\n")
	return dir
}

func TestInspect_defaultOutputIsMarkdown(t *testing.T) {
	dir := inspectRepo(t)
	stdout, _, err := runRoot(t, "inspect", dir)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	for _, want := range []string{"# Inspection report:", "## Structural", "### object_field_frequency"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("markdown output missing %q\n%s", want, stdout)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		t.Errorf("default output looks like JSON, want Markdown")
	}
}

func TestInspect_jsonEmitsSameEvidence(t *testing.T) {
	dir := inspectRepo(t)
	stdout, _, err := runRoot(t, "inspect", "--json", dir)
	if err != nil {
		t.Fatalf("inspect --json: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout)
	}
	if len(records) == 0 {
		t.Fatal("no evidence records emitted")
	}
	for _, rec := range records {
		for _, key := range []string{"inspector", "scope", "n", "evidence"} {
			if _, ok := rec[key]; !ok {
				t.Errorf("record missing %q: %v", key, rec)
			}
		}
	}
}

func TestInspect_outputFileMatchesStdout(t *testing.T) {
	dir := inspectRepo(t)
	stdout, _, err := runRoot(t, "inspect", dir)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	out := filepath.Join(t.TempDir(), "report.md")
	if _, _, err := runRoot(t, "inspect", "-o", out, dir); err != nil {
		t.Fatalf("inspect -o: %v", err)
	}
	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if string(saved) != stdout {
		t.Errorf("-o bytes differ from stdout\n--- file ---\n%s\n--- stdout ---\n%s", saved, stdout)
	}
}

func TestInspect_inspectorFlagNarrows(t *testing.T) {
	dir := inspectRepo(t)
	stdout, _, err := runRoot(t, "inspect", "--json", "--inspector", "object_field_frequency", dir)
	if err != nil {
		t.Fatalf("inspect --inspector: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(records) != 1 || records[0]["inspector"] != "object_field_frequency" {
		t.Errorf("expected only object_field_frequency, got %v", records)
	}
}

func TestInspect_writesNothingUnderScope(t *testing.T) {
	dir := inspectRepo(t)
	before := countFiles(t, dir)
	if _, _, err := runRoot(t, "inspect", dir); err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if after := countFiles(t, dir); after != before {
		t.Errorf("inspect changed file count: %d → %d", before, after)
	}
	if _, err := os.Stat(filepath.Join(dir, ".katalyst")); err == nil {
		t.Errorf("inspect created a .katalyst directory")
	}
}

func TestInspect_missingPathIsUsageError(t *testing.T) {
	_, _, err := runRoot(t, "inspect", filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestInspect_unknownInspectorIsUsageError(t *testing.T) {
	_, _, err := runRoot(t, "inspect", "--inspector", "no_such_inspector", inspectRepo(t))
	var coded interface{ Code() int }
	if err == nil || !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2 for unknown inspector, got: %v", err)
	}
}

func TestInspect_outputIncludesDescriptions(t *testing.T) {
	stdout, _, err := runRoot(t, "inspect", inspectRepo(t))
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	// The one-line description of object_field_frequency's results.
	if !strings.Contains(stdout, "Report, per frontmatter key, how many files contain it.") {
		t.Errorf("output missing inspector description\n%s", stdout)
	}
}

func TestInspect_truncatesLongOutputAndVerboseShowsAll(t *testing.T) {
	dir := t.TempDir()
	var body strings.Builder
	body.WriteString("---\ntitle: A\n---\n# A\n")
	for i := 0; i < 30; i++ {
		body.WriteString(fmt.Sprintf("## Section %02d\n", i))
	}
	writeFile(t, dir, "a.md", body.String())

	truncated, _, err := runRoot(t, "inspect", "--inspector", "markdown_sections", "--max-lines", "5", dir)
	if err != nil {
		t.Fatalf("inspect --max-lines: %v", err)
	}
	if !strings.Contains(truncated, "truncated") {
		t.Errorf("expected a truncation notice with --max-lines 5\n%s", truncated)
	}

	full, _, err := runRoot(t, "inspect", "--inspector", "markdown_sections", "-v", dir)
	if err != nil {
		t.Fatalf("inspect -v: %v", err)
	}
	if strings.Contains(full, "truncated") {
		t.Errorf("-v should not truncate\n%s", full)
	}
	if got := strings.Count(full, "- Section "); got != 30 {
		t.Errorf("-v rendered %d sections, want 30", got)
	}
}

// countFiles counts regular files under root.
func countFiles(t *testing.T, root string) int {
	t.Helper()
	n := 0
	err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return n
}
