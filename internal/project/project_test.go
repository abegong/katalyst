package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/katabase-ai/katalyst/internal/config"
	"github.com/katabase-ai/katalyst/internal/project"
)

// setup writes a two-collection repo and returns a loaded Project.
//
//	notes/   (pattern *.md): dune.md, messiah.md, stray.txt (unmatched)
//	people/  (pattern *.md): herbert.md
func setup(t *testing.T) *project.Project {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(".katalyst/collections/notes.yaml", `path: notes
checks:
  - kind: markdown_requires_h1
`)
	write(".katalyst/collections/people.yaml", `path: people
checks:
  - kind: markdown_requires_h1
`)
	write("notes/dune.md", "---\ntitle: Dune\n---\n# Dune\n")
	write("notes/messiah.md", "---\ntitle: Messiah\n---\n# Messiah\n")
	write("notes/stray.txt", "not markdown\n")
	write("people/herbert.md", "---\ntitle: Herbert\n---\n# Herbert\n")

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return project.New(cfg)
}

func TestParseSelector(t *testing.T) {
	cases := []struct {
		raw        string
		collection string
		item       string
		wantErr    bool
	}{
		{"notes", "notes", "", false},
		{"notes/dune", "notes", "dune", false},
		{"", "", "", true},
		{"notes/sub/deep", "", "", true},
		{"/dune", "", "", true},
		{"notes/", "", "", true},
	}
	for _, tc := range cases {
		sel, err := project.ParseSelector(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseSelector(%q): expected error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSelector(%q): %v", tc.raw, err)
			continue
		}
		if sel.Collection != tc.collection || sel.Item != tc.item {
			t.Errorf("ParseSelector(%q) = {%q,%q}, want {%q,%q}", tc.raw, sel.Collection, sel.Item, tc.collection, tc.item)
		}
	}
}

func TestResolve_emptySelectsAllCollections(t *testing.T) {
	p := setup(t)
	res, err := p.Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// dune, messiah (notes) + herbert (people) = 3 items.
	if len(res.Items) != 3 {
		t.Fatalf("expected 3 items, got %d: %+v", len(res.Items), res.Items)
	}
	if len(res.Scan) != 2 {
		t.Fatalf("expected 2 collections to scan, got %d", len(res.Scan))
	}
}

func TestResolve_collectionSelector(t *testing.T) {
	p := setup(t)
	res, err := p.Resolve([]string{"notes"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items in notes, got %d", len(res.Items))
	}
	for _, it := range res.Items {
		if it.Collection.Name != "notes" {
			t.Errorf("unexpected item from %q", it.Collection.Name)
		}
	}
}

func TestResolve_itemSelector(t *testing.T) {
	p := setup(t)
	res, err := p.Resolve([]string{"notes/dune"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != "dune" {
		t.Fatalf("expected single item dune, got %+v", res.Items)
	}
	// A single-item selector does not schedule an unmatched scan.
	if len(res.Scan) != 0 {
		t.Fatalf("expected no scan for item selector, got %d", len(res.Scan))
	}
}

func TestResolve_bareTokenIsCollectionNotItem(t *testing.T) {
	p := setup(t)
	// "dune" has no slash, so it is treated as a (nonexistent) collection,
	// never as the item notes/dune.
	_, err := p.Resolve([]string{"dune"})
	var ue *project.UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UsageError for bare token, got %v", err)
	}
}

func TestResolve_unknownCollection(t *testing.T) {
	p := setup(t)
	_, err := p.Resolve([]string{"nope"})
	var ue *project.UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UsageError for unknown collection, got %v", err)
	}
}

func TestResolve_unknownItem(t *testing.T) {
	p := setup(t)
	_, err := p.Resolve([]string{"notes/ghost"})
	var ue *project.UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UsageError for unknown item, got %v", err)
	}
}

func TestItems_sortedAndIdResolution(t *testing.T) {
	p := setup(t)
	notes, _ := p.Collection("notes")
	items, err := p.Items(notes)
	if err != nil {
		t.Fatalf("Items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "dune" || items[1].ID != "messiah" {
		t.Fatalf("expected sorted [dune messiah], got [%s %s]", items[0].ID, items[1].ID)
	}
	// Reverse resolution: notes/dune → notes/dune.md.
	if got := project.ItemPath(notes, "dune"); got != items[0].Path {
		t.Errorf("ItemPath = %q, want %q", got, items[0].Path)
	}
}

func TestUnmatched_reportsNonMatchingFiles(t *testing.T) {
	p := setup(t)
	notes, _ := p.Collection("notes")
	unmatched, err := p.Unmatched(notes)
	if err != nil {
		t.Fatalf("Unmatched: %v", err)
	}
	if len(unmatched) != 1 || unmatched[0] != "stray.txt" {
		t.Fatalf("expected [stray.txt], got %v", unmatched)
	}
}

func TestItemAt_unknownItemIsUsageError(t *testing.T) {
	p := setup(t)
	_, err := p.ItemAt("notes", "ghost")
	var ue *project.UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UsageError, got %v", err)
	}
}
