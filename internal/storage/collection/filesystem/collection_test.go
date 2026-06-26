package filesystem_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection"
	"github.com/abegong/katalyst/internal/storage/collection/filesystem"
)

// scaffoldNotes writes a small notes/ directory and returns a Collection
// pointing at it. Collection fields are all exported, so a test can build one
// directly without going through project.Load.
//
//	notes/ (pattern *.md): dune.md, messiah.md, stray.txt (unmatched)
func scaffoldNotes(t *testing.T) collection.Collection {
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
	write("notes/dune.md", "# Dune\n")
	write("notes/messiah.md", "# Messiah\n")
	write("notes/stray.txt", "not markdown\n")
	return collection.Collection{
		Name:    "notes",
		Path:    "notes",
		Dir:     filepath.Join(dir, "notes"),
		Pattern: "*.md",
	}
}

func TestFilesystem_Items_sortedStems(t *testing.T) {
	c := scaffoldNotes(t)
	def := filesystem.New(filepath.Dir(c.Dir), []collection.Collection{c})
	items, err := def.Items(c)
	if err != nil {
		t.Fatalf("Items: %v", err)
	}
	if len(items) != 2 || items[0].ID != "dune" || items[1].ID != "messiah" {
		t.Fatalf("expected [dune messiah], got %+v", items)
	}
	if items[0].Path != filepath.Join(c.Dir, "dune.md") {
		t.Errorf("unexpected path %q", items[0].Path)
	}
}

func TestFilesystem_Items_missingDirIsEmpty(t *testing.T) {
	c := collection.Collection{Name: "ghost", Dir: filepath.Join(t.TempDir(), "nope"), Pattern: "*.md"}
	def := filesystem.New("", []collection.Collection{c})
	items, err := def.Items(c)
	if err != nil {
		t.Fatalf("Items: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items for missing dir, got %d", len(items))
	}
}

func TestFilesystem_Unmatched_reportsNonMatching(t *testing.T) {
	c := scaffoldNotes(t)
	def := filesystem.New("", []collection.Collection{c})
	un, err := def.Unmatched(c)
	if err != nil {
		t.Fatalf("Unmatched: %v", err)
	}
	if len(un) != 1 || un[0] != "stray.txt" {
		t.Fatalf("expected [stray.txt], got %v", un)
	}
}

func TestFilesystem_Reference_reverseResolution(t *testing.T) {
	c := scaffoldNotes(t)
	def := filesystem.New("", []collection.Collection{c})
	ref, err := def.Reference(c, "dune")
	if err != nil {
		t.Fatalf("Reference: %v", err)
	}
	if want := filepath.Join(c.Dir, "dune.md"); string(ref) != want {
		t.Fatalf("Reference = %q, want %q", ref, want)
	}
}

func TestFilesystem_Scope_fileIsItem(t *testing.T) {
	if g := filesystem.New("", nil).Scope(); g != storage.FileIsItem {
		t.Fatalf("Scope = %v, want FileIsItem", g)
	}
}

func TestFilesystem_Collections_returnsBound(t *testing.T) {
	c := scaffoldNotes(t)
	cols := filesystem.New("", []collection.Collection{c}).Collections()
	if len(cols) != 1 || cols[0].Name != "notes" {
		t.Fatalf("expected [notes], got %+v", cols)
	}
}
