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
// directory — a raw store for the source layer.
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
	for _, want := range []string{"# Inspection report:", "### document_shape", "### file_tree"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("source-layer output missing %q\n%s", want, stdout)
		}
	}
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
	stdout, _, err := runRoot(t, "inspect", "--json", "--inspector", "document_shape", dir)
	if err != nil {
		t.Fatalf("inspect --inspector: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if len(records) != 1 || records[0]["inspector"] != "document_shape" {
		t.Errorf("expected only document_shape, got %v", records)
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

func TestInspect_collapseParamsMutuallyExclusive(t *testing.T) {
	dir := inspectRepo(t)
	_, _, err := runRoot(t, "inspect", "--detail", "coarse", "--max-classes", "2", dir)
	var coded interface{ Code() int }
	if err == nil || !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit 2 for mutually-exclusive collapse flags, got: %v", err)
	}
}

func TestInspect_outputIncludesDescriptions(t *testing.T) {
	stdout, _, err := runRoot(t, "inspect", inspectRepo(t))
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !strings.Contains(stdout, "Cluster files into candidate collections") {
		t.Errorf("output missing inspector description\n%s", stdout)
	}
}

func TestInspect_truncatesLongOutputAndVerboseShowsAll(t *testing.T) {
	dir := t.TempDir()
	// Ten files with disjoint frontmatter keys + sections → ten singleton
	// document_shape classes, enough lines to exceed a small --max-lines.
	for i := 0; i < 10; i++ {
		writeFile(t, dir, fmt.Sprintf("docs/f%02d.md", i),
			fmt.Sprintf("---\nk%02d: v\n---\n# H\n\n## S%02d\n", i, i))
	}

	truncated, _, err := runRoot(t, "inspect", "--inspector", "document_shape", "--max-lines", "5", dir)
	if err != nil {
		t.Fatalf("inspect --max-lines: %v", err)
	}
	if !strings.Contains(truncated, "truncated") {
		t.Errorf("expected a truncation notice with --max-lines 5\n%s", truncated)
	}

	full, _, err := runRoot(t, "inspect", "--inspector", "document_shape", "-v", dir)
	if err != nil {
		t.Fatalf("inspect -v: %v", err)
	}
	if strings.Contains(full, "truncated") {
		t.Errorf("-v should not truncate\n%s", full)
	}
	if got := strings.Count(full, "label=docs/f"); got != 10 {
		t.Errorf("-v rendered %d outliers, want 10\n%s", got, full)
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
