package inspect_test

import (
	"testing"

	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/katabase-ai/katalyst/internal/inspect"
)

// file parses src into a Corpus File, recording any parse error just as
// Load would.
func file(t *testing.T, rel, src string) inspect.File {
	t.Helper()
	doc, err := frontmatter.Parse([]byte(src))
	return inspect.File{Rel: rel, Doc: doc, ParseErr: err}
}

// sampleCorpus has two well-formed files, one without frontmatter, and one
// that fails to parse — enough to exercise the counting inspectors.
func sampleCorpus(t *testing.T) inspect.Corpus {
	t.Helper()
	return inspect.Corpus{
		Scope: "books",
		Files: []inspect.File{
			file(t, "dune.md", "---\ntitle: Dune\nauthor: Herbert\nstatus: read\n---\n# Dune\n"),
			file(t, "messiah.md", "---\ntitle: Dune Messiah\nstatus: reading\n---\n# Dune Messiah\n"),
			file(t, "loose.md", "no frontmatter here\n"),
			file(t, "broken.md", "---\n- a\n- b\n---\nbody\n"), // seq root → decode error
		},
	}
}

func TestWalkParse_countsFilesAndParseOutcomes(t *testing.T) {
	ev := inspect.WalkParse{}.Inspect(sampleCorpus(t))

	if ev.Inspector != "walk_parse" {
		t.Errorf("inspector = %q, want walk_parse", ev.Inspector)
	}
	if ev.N != 4 {
		t.Errorf("N = %d, want 4", ev.N)
	}
	wantInts := map[string]int{"files": 4, "parsed": 3, "failed": 1, "with_frontmatter": 2}
	for k, want := range wantInts {
		if got, _ := ev.Data[k].(int); got != want {
			t.Errorf("Data[%q] = %v, want %d", k, ev.Data[k], want)
		}
	}
}

func TestWalkParse_reportsFailuresNotSkipped(t *testing.T) {
	ev := inspect.WalkParse{}.Inspect(sampleCorpus(t))
	failures, ok := ev.Data["failures"].([]string)
	if !ok || len(failures) != 1 || failures[0] != "broken.md" {
		t.Fatalf("failures = %v, want [broken.md]", ev.Data["failures"])
	}
}

func TestObjectFieldFrequency_presentCountsMatchCorpus(t *testing.T) {
	ev := inspect.ObjectFieldFrequency{}.Inspect(sampleCorpus(t))

	if ev.N != 4 {
		t.Errorf("N = %d, want 4", ev.N)
	}
	want := map[string]int{"title": 2, "author": 1, "status": 2}
	for field, n := range want {
		entry, ok := ev.Data[field].(map[string]any)
		if !ok {
			t.Errorf("Data[%q] missing or wrong shape: %v", field, ev.Data[field])
			continue
		}
		if got, _ := entry["present"].(int); got != n {
			t.Errorf("%q present = %v, want %d", field, entry["present"], n)
		}
	}
	if _, exists := ev.Data["nonexistent"]; exists {
		t.Errorf("unexpected field reported")
	}
}
