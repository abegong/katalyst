package inspect_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
)

// renderInput is two evidence records from different families, built directly
// so the renderer is exercised without depending on any inspector's internals.
func renderInput() []inspect.Evidence {
	return []inspect.Evidence{
		{Inspector: "document_shape", Scope: "books", N: 3, Data: map[string]any{"classes": []any{}, "outliers": []any{}}},
		{Inspector: "object_fields", Scope: "books", N: 3, Data: map[string]any{"title": map[string]any{"present": 3}}},
	}
}

func TestRenderMarkdown_groupsByFamilyWithCounts(t *testing.T) {
	md := inspect.RenderMarkdown(renderInput(), 0)
	for _, want := range []string{
		"## Structural",
		"### document_shape (n=3)",
		"## Object",
		"### object_fields (n=3)",
		"- present: 3",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\n---\n%s", want, md)
		}
	}
}

func TestRenderMarkdown_includesDescription(t *testing.T) {
	md := inspect.RenderMarkdown(renderInput(), 0)
	want := "_" + inspect.Summary("object_fields") + "_"
	if !strings.Contains(md, want) {
		t.Errorf("markdown missing description %q\n---\n%s", want, md)
	}
}

func TestRenderMarkdown_truncatesPerInspector(t *testing.T) {
	data := map[string]any{}
	for i := 0; i < 30; i++ {
		data[string(rune('a'+i))] = i
	}
	ev := inspect.Evidence{Inspector: "object_fields", Scope: "x", N: 1, Data: data}

	truncated := inspect.RenderMarkdown([]inspect.Evidence{ev}, 5)
	if !strings.Contains(truncated, "more line(s) truncated") {
		t.Errorf("expected truncation notice\n%s", truncated)
	}

	full := inspect.RenderMarkdown([]inspect.Evidence{ev}, 0)
	if strings.Contains(full, "truncated") {
		t.Errorf("maxLines=0 should not truncate\n%s", full)
	}
}

func TestRenderMarkdown_fileTreeSmallTree(t *testing.T) {
	ev := inspect.Evidence{Inspector: "file_tree", Scope: "repo", N: 3, Data: map[string]any{
		"file_count": 3,
		"dir_count":  2,
		"max_depth":  2,
		"extensions": map[string]any{".md": 3},
		"top_level_regions": []any{
			map[string]any{"path": "books/", "file_count": 3, "extensions": map[string]any{".md": 3}},
		},
		"tree_entries":         []any{".", "└── books/", "    ├── dune.md", "    └── it.md"},
		"representative_paths": []any{"books/dune.md", "books/it.md"},
		"directory_summaries":  []any{},
		"naming": map[string]any{
			"dominant_bucket":  "lowercase",
			"dominant_count":   3,
			"comparable_count": 3,
			"buckets":          map[string]any{"lowercase": 3},
			"exceptions":       []any{},
		},
	}}
	md := inspect.RenderMarkdown([]inspect.Evidence{ev}, 20)
	for _, want := range []string{
		"summary:",
		"  files        : 3",
		"  dominant type: .md (3 of 3 files)",
		"----------------------------------------",
		"tree:",
		"└── books/",
		"file types:",
		"  TYPE  FILES",
		"  .md   3",
		"naming:",
		"  pattern: lowercase",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("file_tree markdown missing %q\n%s", want, md)
		}
	}
}

func TestRenderMarkdown_fileTreeVerboseShowsExpandedEvidence(t *testing.T) {
	ev := inspect.Evidence{Inspector: "file_tree", Scope: "repo", N: 1, Data: map[string]any{
		"file_count":        1,
		"dir_count":         1,
		"max_depth":         5,
		"extensions":        map[string]any{".md": 1, ".png": 1, ".css": 1, ".txt": 1, ".yml": 1, ".json": 1},
		"tree_entries":      []any{},
		"top_level_regions": []any{},
		"naming": map[string]any{
			"dominant_bucket":  "",
			"dominant_count":   0,
			"comparable_count": 1,
			"buckets":          map[string]any{"mixed/other": 1},
			"exceptions":       []any{},
		},
		"directory_summaries": []any{
			map[string]any{"path": ".", "descendant_file_count": 1, "direct_file_count": 0, "markdown_heavy": false},
		},
		"deep_paths":           []any{"a/b/c/d/e.md"},
		"representative_paths": []any{"a/b/c/d/e.md"},
	}}
	defaultMD := inspect.RenderMarkdown([]inspect.Evidence{ev}, 20)
	if strings.Contains(defaultMD, "directory density:") {
		t.Errorf("default file_tree output should stay skimmable\n%s", defaultMD)
	}
	verbose := inspect.RenderMarkdown([]inspect.Evidence{ev}, 0)
	for _, want := range []string{"directory density:", "deep paths:", "  mixed/other  1"} {
		if !strings.Contains(verbose, want) {
			t.Errorf("verbose file_tree markdown missing %q\n%s", want, verbose)
		}
	}
}

func TestRenderJSON_roundTrips(t *testing.T) {
	out, err := inspect.RenderJSON(renderInput())
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal(out, &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	first := records[0]
	for _, key := range []string{"inspector", "description", "scope", "n", "evidence"} {
		if _, ok := first[key]; !ok {
			t.Errorf("record missing %q: %v", key, first)
		}
	}
	if first["inspector"] != "document_shape" {
		t.Errorf("inspector = %v, want document_shape", first["inspector"])
	}
	if first["description"] != inspect.Summary("document_shape") {
		t.Errorf("description = %v, want %q", first["description"], inspect.Summary("document_shape"))
	}
}

func TestRenderJSON_emptyIsArray(t *testing.T) {
	out, err := inspect.RenderJSON(nil)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	if strings.TrimSpace(string(out)) != "[]" {
		t.Errorf("empty render = %q, want []", out)
	}
}
