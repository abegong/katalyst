package inspect_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/katabase-ai/katalyst/internal/inspect"
)

func renderInput(t *testing.T) []inspect.Evidence {
	t.Helper()
	c := sampleCorpus(t)
	return []inspect.Evidence{
		inspect.WalkParse{}.Inspect(c),
		inspect.ObjectFieldFrequency{}.Inspect(c),
	}
}

func TestRenderMarkdown_groupsByFamilyWithCounts(t *testing.T) {
	md := inspect.RenderMarkdown(renderInput(t), 0)

	for _, want := range []string{
		"## Structural",
		"### walk_parse (n=4)",
		"## Object",
		"### object_field_frequency (n=4)",
		"- files: 4",
		"- present: 2", // title/status appear in 2 files
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\n---\n%s", want, md)
		}
	}
}

func TestRenderMarkdown_includesDescription(t *testing.T) {
	md := inspect.RenderMarkdown(renderInput(t), 0)
	// Each inspector's results carry a one-line description from the registry.
	want := "_" + inspect.Summary("object_field_frequency") + "_"
	if !strings.Contains(md, want) {
		t.Errorf("markdown missing description %q\n---\n%s", want, md)
	}
}

func TestRenderMarkdown_truncatesPerInspector(t *testing.T) {
	// A file with many distinct sections makes markdown_sections exceed the cap.
	var body strings.Builder
	body.WriteString("---\ntitle: A\n---\n# A\n")
	for i := 0; i < 30; i++ {
		body.WriteString("## Section ")
		body.WriteByte(byte('a' + i))
		body.WriteString("\n")
	}
	c := inspect.Corpus{Scope: "x", Files: []inspect.File{file(t, "a.md", body.String())}}
	ev := inspect.MarkdownSections{}.Inspect(c)

	truncated := inspect.RenderMarkdown([]inspect.Evidence{ev}, 5)
	if !strings.Contains(truncated, "more line(s) truncated") {
		t.Errorf("expected truncation notice\n%s", truncated)
	}
	// Body lines after the heading/description should be capped near the limit.
	if got := strings.Count(truncated, "- Section "); got > 5 {
		t.Errorf("rendered %d section lines, want <= 5", got)
	}

	full := inspect.RenderMarkdown([]inspect.Evidence{ev}, 0)
	if strings.Contains(full, "truncated") {
		t.Errorf("maxLines=0 should not truncate\n%s", full)
	}
	if got := strings.Count(full, "- Section "); got != 30 {
		t.Errorf("full render has %d section lines, want 30", got)
	}
}

func TestRenderJSON_roundTrips(t *testing.T) {
	out, err := inspect.RenderJSON(renderInput(t))
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
	if first["inspector"] != "walk_parse" {
		t.Errorf("inspector = %v, want walk_parse", first["inspector"])
	}
	if first["description"] != inspect.Summary("walk_parse") {
		t.Errorf("description = %v, want %q", first["description"], inspect.Summary("walk_parse"))
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
