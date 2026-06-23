package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/abegong/katalyst/internal/config"
	"github.com/abegong/katalyst/internal/storage"
)

// scaffoldNotes writes a small notes/ directory and returns a Collection
// pointing at it. Collection fields are all exported, so a test can build one
// directly without going through config.Load.
//
//	notes/ (pattern *.md): dune.md, messiah.md, stray.txt (unmatched)
func scaffoldNotes(t *testing.T) config.Collection {
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
	return config.Collection{
		Name:    "notes",
		Path:    "notes",
		Dir:     filepath.Join(dir, "notes"),
		Pattern: "*.md",
	}
}

func TestFilesystem_Items_sortedStems(t *testing.T) {
	c := scaffoldNotes(t)
	def := storage.NewFilesystem(filepath.Dir(c.Dir), []config.Collection{c})
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
	c := config.Collection{Name: "ghost", Dir: filepath.Join(t.TempDir(), "nope"), Pattern: "*.md"}
	def := storage.NewFilesystem("", []config.Collection{c})
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
	def := storage.NewFilesystem("", []config.Collection{c})
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
	def := storage.NewFilesystem("", []config.Collection{c})
	ref, err := def.Reference(c, "dune")
	if err != nil {
		t.Fatalf("Reference: %v", err)
	}
	if want := filepath.Join(c.Dir, "dune.md"); string(ref) != want {
		t.Fatalf("Reference = %q, want %q", ref, want)
	}
}

func TestFilesystem_Granularity_fileIsItem(t *testing.T) {
	if g := storage.NewFilesystem("", nil).Granularity(); g != storage.FileIsItem {
		t.Fatalf("Granularity = %v, want FileIsItem", g)
	}
}

func TestFilesystem_Collections_returnsBound(t *testing.T) {
	c := scaffoldNotes(t)
	cols := storage.NewFilesystem("", []config.Collection{c}).Collections()
	if len(cols) != 1 || cols[0].Name != "notes" {
		t.Fatalf("expected [notes], got %+v", cols)
	}
}

func TestKnown_onlyFilesystem(t *testing.T) {
	if !storage.Known(storage.Filesystem) {
		t.Errorf("filesystem should be a known storage type")
	}
	if storage.Known(storage.StorageType("sqlite")) {
		t.Errorf("sqlite is not implemented yet and should not be known")
	}
}
