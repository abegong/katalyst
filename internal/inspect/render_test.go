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
	md := inspect.RenderMarkdown(renderInput(t))

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
	for _, key := range []string{"inspector", "scope", "n", "evidence"} {
		if _, ok := first[key]; !ok {
			t.Errorf("record missing %q: %v", key, first)
		}
	}
	if first["inspector"] != "walk_parse" {
		t.Errorf("inspector = %v, want walk_parse", first["inspector"])
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
