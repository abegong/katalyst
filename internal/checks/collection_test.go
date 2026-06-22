package checks_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
)

func TestUniqueFilename_flagsCollision(t *testing.T) {
	ctx := checks.CollectionContext{Items: []checks.ItemContext{
		{FilePath: "notes/a/dune.md"},
		{FilePath: "notes/b/dune.md"},
		{FilePath: "notes/c/other.md"},
	}}
	violations := checks.UniqueFilename{}.RunCollection(ctx)
	if len(violations) != 1 {
		t.Fatalf("expected 1 collision violation, got %d: %v", len(violations), violations)
	}
	// Names both colliding paths.
	if !strings.Contains(violations[0].Message, "notes/a/dune.md") ||
		!strings.Contains(violations[0].Message, "notes/b/dune.md") {
		t.Fatalf("expected both paths named, got %q", violations[0].Message)
	}
}

func TestUniqueField_flagsDuplicateValues(t *testing.T) {
	ctx := checks.CollectionContext{Items: []checks.ItemContext{
		{FilePath: "notes/x.md", Meta: map[string]any{"slug": "dune"}},
		{FilePath: "notes/y.md", Meta: map[string]any{"slug": "dune"}},
		{FilePath: "notes/z.md", Meta: map[string]any{"slug": "other"}},
	}}
	violations := checks.UniqueField{Field: "slug"}.RunCollection(ctx)
	if len(violations) != 1 || !strings.Contains(violations[0].Message, `"dune"`) {
		t.Fatalf("expected one duplicate-slug violation, got %v", violations)
	}
}

func TestIndexFileRequired_flagsMissing(t *testing.T) {
	root := t.TempDir()
	withIndex := filepath.Join(root, "has")
	without := filepath.Join(root, "missing")
	if err := os.MkdirAll(withIndex, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(without, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withIndex, "_index.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := checks.CollectionContext{Items: []checks.ItemContext{
		{FilePath: filepath.Join(withIndex, "a.md")},
		{FilePath: filepath.Join(without, "b.md")},
	}}
	violations := checks.IndexFileRequired{}.RunCollection(ctx)
	if len(violations) != 1 || violations[0].File != without {
		t.Fatalf("expected one missing-index violation for %q, got %v", without, violations)
	}
}
