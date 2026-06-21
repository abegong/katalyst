package inspect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/katabase-ai/katalyst/internal/inspect"
)

// writeFile scaffolds a file (creating parent dirs) under root for Load tests.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// typedCorpus exercises value-level inspectors: a numeric field with a range,
// a small-cardinality status (enum candidate), a string field, and a key whose
// type is inconsistent across files.
func typedCorpus(t *testing.T) inspect.Corpus {
	t.Helper()
	return inspect.Corpus{
		Scope: "books",
		Files: []inspect.File{
			file(t, "a.md", "---\ntitle: Dune\nrating: 5\nstatus: read\nyear: 1965\n---\n# Dune\n## Review\n"),
			file(t, "b.md", "---\ntitle: It\nrating: 3\nstatus: reading\nyear: \"1986\"\n---\n# It\n## Review\n"),
			file(t, "c.md", "---\ntitle: Sula\nrating: 4\nstatus: read\nyear: 1973\n---\n# Sula\n"),
		},
	}
}

func TestObjectFieldTypes_reportsMixedAsMixed(t *testing.T) {
	ev := inspect.ObjectFieldTypes{}.Inspect(typedCorpus(t))
	year, ok := ev.Data["year"].(map[string]any)
	if !ok {
		t.Fatalf("year missing: %v", ev.Data["year"])
	}
	types := year["types"].(map[string]any)
	if got, _ := types["integer"].(int); got != 2 {
		t.Errorf("year integer count = %v, want 2", types["integer"])
	}
	if got, _ := types["string"].(int); got != 1 {
		t.Errorf("year string count = %v, want 1", types["string"])
	}
}

func TestObjectFieldValues_enumCandidateAndCardinality(t *testing.T) {
	ev := inspect.ObjectFieldValues{}.Inspect(typedCorpus(t))
	status := ev.Data["status"].(map[string]any)
	if got, _ := status["cardinality"].(int); got != 2 {
		t.Errorf("status cardinality = %v, want 2", status["cardinality"])
	}
	values, ok := status["values"].(map[string]any)
	if !ok {
		t.Fatalf("status values missing for small set: %v", status)
	}
	if got, _ := values["read"].(int); got != 2 {
		t.Errorf("status read count = %v, want 2", values["read"])
	}
}

func TestObjectFieldValues_omitsValuesForNonScalar(t *testing.T) {
	c := inspect.Corpus{Scope: "x", Files: []inspect.File{
		file(t, "a.md", "---\ntags:\n  - go\n  - md\n---\nbody\n"),
	}}
	ev := inspect.ObjectFieldValues{}.Inspect(c)
	tags := ev.Data["tags"].(map[string]any)
	if _, present := tags["values"]; present {
		t.Errorf("array field should not report a value set: %v", tags)
	}
}

func TestObjectFieldNumericRange_minMax(t *testing.T) {
	ev := inspect.ObjectFieldNumericRange{}.Inspect(typedCorpus(t))
	rating := ev.Data["rating"].(map[string]any)
	if rating["min"].(float64) != 3 || rating["max"].(float64) != 5 {
		t.Errorf("rating range = [%v,%v], want [3,5]", rating["min"], rating["max"])
	}
	if got, _ := rating["count"].(int); got != 3 {
		t.Errorf("rating count = %v, want 3", rating["count"])
	}
	// year is numeric in 2 of 3 files (one is a string).
	year := ev.Data["year"].(map[string]any)
	if got, _ := year["count"].(int); got != 2 {
		t.Errorf("year numeric count = %v, want 2", year["count"])
	}
}

func TestObjectFieldStringLength_minMax(t *testing.T) {
	ev := inspect.ObjectFieldStringLength{}.Inspect(typedCorpus(t))
	title := ev.Data["title"].(map[string]any)
	if title["min_length"].(int) != 2 { // "It"
		t.Errorf("title min_length = %v, want 2", title["min_length"])
	}
	if title["max_length"].(int) != 4 { // "Dune"/"Sula"
		t.Errorf("title max_length = %v, want 4", title["max_length"])
	}
}

func TestMarkdownHeadingShape_rates(t *testing.T) {
	ev := inspect.MarkdownHeadingShape{}.Inspect(typedCorpus(t))
	if got, _ := ev.Data["single_h1"].(int); got != 3 {
		t.Errorf("single_h1 = %v, want 3", ev.Data["single_h1"])
	}
	if got, _ := ev.Data["h1_matches_title"].(int); got != 3 {
		t.Errorf("h1_matches_title = %v, want 3", ev.Data["h1_matches_title"])
	}
}

func TestMarkdownSections_recurring(t *testing.T) {
	ev := inspect.MarkdownSections{}.Inspect(typedCorpus(t))
	if got, _ := ev.Data["Review"].(int); got != 2 {
		t.Errorf("Review section count = %v, want 2", ev.Data["Review"])
	}
}

func TestMarkdownCodeFences_languageRate(t *testing.T) {
	c := inspect.Corpus{Scope: "x", Files: []inspect.File{
		file(t, "a.md", "---\ntitle: A\n---\n# A\n```go\nx := 1\n```\n```\nplain\n```\n"),
	}}
	ev := inspect.MarkdownCodeFences{}.Inspect(c)
	if got, _ := ev.Data["opening_fences"].(int); got != 2 {
		t.Errorf("opening_fences = %v, want 2", ev.Data["opening_fences"])
	}
	if got, _ := ev.Data["with_language"].(int); got != 1 {
		t.Errorf("with_language = %v, want 1", ev.Data["with_language"])
	}
}

func TestFilesystemNaming_casingAndSpaces(t *testing.T) {
	c := inspect.Corpus{Scope: "x", Files: []inspect.File{
		file(t, "dune-messiah.md", "body"),
		file(t, "Children Of Dune.md", "body"),
		file(t, "god_emperor.md", "body"),
	}}
	ev := inspect.FilesystemNaming{}.Inspect(c)
	casing := ev.Data["casing"].(map[string]any)
	if casing["kebab"].(int) != 1 || casing["snake"].(int) != 1 || casing["other"].(int) != 1 {
		t.Errorf("casing = %v, want kebab/snake/other = 1/1/1", casing)
	}
	if got, _ := ev.Data["with_spaces"].(int); got != 1 {
		t.Errorf("with_spaces = %v, want 1", ev.Data["with_spaces"])
	}
}

func TestFrontmatterShape_groupsIdenticalKeysets(t *testing.T) {
	c := inspect.Corpus{Scope: "x", Files: []inspect.File{
		file(t, "a.md", "---\ntitle: A\nstatus: read\n---\nbody"),
		file(t, "b.md", "---\nstatus: reading\ntitle: B\n---\nbody"), // same keys, different order
		file(t, "c.md", "---\ntitle: C\n---\nbody"),                  // different keyset
	}}
	ev := inspect.FrontmatterShape{}.Inspect(c)
	groups := ev.Data["groups"].([]any)
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2: %v", len(groups), groups)
	}
	top := groups[0].(map[string]any)
	if top["fingerprint"].(string) != "status,title" {
		t.Errorf("top fingerprint = %q, want status,title", top["fingerprint"])
	}
	if got, _ := top["count"].(int); got != 2 {
		t.Errorf("top group count = %v, want 2", top["count"])
	}
}

// TestAll_runOverLoadedCorpus is a smoke test: every registered inspector runs
// over a real on-disk corpus and returns its own name with N set.
func TestAll_runOverLoadedCorpus(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "books/dune.md", "---\ntitle: Dune\nrating: 5\n---\n# Dune\n## Review\n")
	writeFile(t, dir, "books/it.md", "---\ntitle: It\nrating: 3\n---\n# It\n")

	c, err := inspect.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Files) != 2 {
		t.Fatalf("loaded %d files, want 2", len(c.Files))
	}
	for _, ins := range inspect.All() {
		ev := ins.Inspect(c)
		if ev.Inspector != ins.Name() {
			t.Errorf("%s: evidence inspector = %q", ins.Name(), ev.Inspector)
		}
		if ev.N != 2 {
			t.Errorf("%s: N = %d, want 2", ins.Name(), ev.N)
		}
	}
}
