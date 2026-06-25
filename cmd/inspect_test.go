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

// inspectRepo scaffolds a small markdown tree (no .katalyst) and returns its
// directory, a raw store for the source layer.
func inspectRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, dir, "books/dune.md", "---\ntitle: Dune\nstatus: read\n---\n# Dune\n## Review\n")
	writeFile(t, dir, "books/it.md", "---\ntitle: It\nstatus: read\n---\n# It\n## Review\n")
	return dir
}

func TestInspect_rawPathRunsSourceLayer(t *testing.T) {
	dir := inspectRepo(t)
	stdout, _, err := runRoot(t, "inspect", dir)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	// The report header embeds the inspected path; normTmp makes it stable.
	snapshot(t, "inspect/source-report.txt", stdout, normTmp(dir))
	if strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		t.Errorf("default output looks like JSON, want Markdown")
	}
}

func TestInspect_collectionLayerWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".katalyst/storage/local.yaml", `type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
`)
	writeFile(t, dir, "notes/dune.md", "---\ntitle: Dune\nrating: 5\n---\n# Dune\n")
	writeFile(t, dir, "notes/it.md", "---\ntitle: It\nrating: 4\n---\n# It\n")
	chdir(t, dir)

	stdout, _, err := runRoot(t, "inspect", "--json", "notes")
	if err != nil {
		t.Fatalf("inspect notes: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatalf("bad json: %v\n%s", err, stdout)
	}
	names := map[string]bool{}
	for _, r := range records {
		names[r["inspector"].(string)] = true
		if r["scope"] != "notes" {
			t.Errorf("scope = %v, want notes", r["scope"])
		}
	}
	if !names["object_fields"] || !names["markdown_body"] {
		t.Errorf("collection layer should run object_fields + markdown_body, got %v", names)
	}
	if names["file_tree"] {
		t.Errorf("collection layer should not run source inspectors, got %v", names)
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
	stdout, _, err := runRoot(t, "inspect", "--json", "--inspector", "file_tree", dir)
	if err != nil {
		t.Fatalf("inspect --inspector: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(records) != 1 || records[0]["inspector"] != "file_tree" {
		t.Errorf("expected only file_tree, got %v", records)
	}
}

func TestInspect_selectRunsFileContentShape(t *testing.T) {
	dir := inspectRepo(t)
	writeFile(t, dir, "data/books.csv", "title,rating\nDune,5\n")
	stdout, _, err := runRoot(t, "inspect", "--json", "--inspector", "file_content_shape", "--select", `ext = ".csv"`, dir)
	if err != nil {
		t.Fatalf("inspect --select: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(records) != 1 || records[0]["inspector"] != "file_content_shape" {
		t.Fatalf("expected only file_content_shape, got %v", records)
	}
	ev := records[0]["evidence"].(map[string]any)
	if got := ev["file_count"].(float64); got != 1 {
		t.Errorf("file_count = %v, want 1 selected CSV file", got)
	}
	if got := ev["selector"].(string); got != `ext = ".csv"` {
		t.Errorf("selector = %q", got)
	}
}

func TestInspect_selectRejectsInvalidCombinations(t *testing.T) {
	dir := inspectRepo(t)
	tests := [][]string{
		{"inspect", "--select", "books", dir},
		{"inspect", "--inspector", "file_tree", "--select", "books", dir},
		{"inspect", "--inspector", "file_content_shape", "--inspector", "file_tree", "--select", "books", dir},
	}
	for _, args := range tests {
		_, _, err := runRoot(t, args...)
		var coded interface{ Code() int }
		if err == nil || !errors.As(err, &coded) || coded.Code() != 2 {
			t.Errorf("%v: expected exit 2, got %v", args, err)
		}
	}
}

func TestInspect_selectRejectsCollectionTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".katalyst/storage/local.yaml", `type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
`)
	writeFile(t, dir, "notes/dune.md", "---\ntitle: Dune\n---\n# Dune\n")
	chdir(t, dir)

	_, _, err := runRoot(t, "inspect", "--inspector", "file_content_shape", "--select", "notes", "notes")
	var coded interface{ Code() int }
	if err == nil || !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit 2 for --select with collection target, got %v", err)
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
	if !strings.Contains(stdout, "Profile selected files by text") {
		t.Errorf("output missing inspector description\n%s", stdout)
	}
}

func TestInspect_truncatesLongOutputAndVerboseShowsAll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".katalyst/storage/local.yaml", `type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
`)
	for i := 0; i < 10; i++ {
		writeFile(t, dir, fmt.Sprintf("notes/f%02d.md", i),
			fmt.Sprintf("---\nk%02d: v\n---\n# H\n", i))
	}
	chdir(t, dir)

	truncated, _, err := runRoot(t, "inspect", "--inspector", "object_fields", "--max-lines", "5", "notes")
	if err != nil {
		t.Fatalf("inspect --max-lines: %v", err)
	}
	if !strings.Contains(truncated, "truncated") {
		t.Errorf("expected a truncation notice with --max-lines 5\n%s", truncated)
	}

	full, _, err := runRoot(t, "inspect", "--inspector", "object_fields", "-v", "notes")
	if err != nil {
		t.Fatalf("inspect -v: %v", err)
	}
	if strings.Contains(full, "truncated") {
		t.Errorf("-v should not truncate\n%s", full)
	}
	if !strings.Contains(full, "k09") {
		t.Errorf("-v should render all object field evidence\n%s", full)
	}
}

func TestInspect_missingArgumentGivesHelpfulError(t *testing.T) {
	_, stderr, err := runRoot(t, "inspect")
	var coded interface{ Code() int }
	if err == nil || !errors.As(err, &coded) || coded.Code() != 2 {
		t.Fatalf("expected exit code 2, got: %v", err)
	}
	combined := err.Error() + stderr
	if !strings.Contains(combined, "usage: katalyst inspect <path-or-collection>") {
		t.Errorf("error should carry a usage hint: %q", combined)
	}
	if strings.Contains(combined, "arg(s)") {
		t.Errorf("should not surface Cobra's default arity message: %q", combined)
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
