package inspect_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
	"github.com/abegong/katalyst/internal/storage"
)

func TestFileTree_opensNothingAndProfilesDirs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes/dune.md", "---\ntitle: Dune\n---\n# Dune\n\n## Review\n")
	writeFile(t, dir, "notes/messiah.md", "---\ntitle: Messiah\n---\n# Messiah\n\n## Review\n")
	writeFile(t, dir, "assets/logo.png", "binary")

	view, err := inspect.NewSourceView(dir)
	if err != nil {
		t.Fatalf("NewSourceView: %v", err)
	}

	ft := inspect.FileTree{}
	if !ft.AppliesTo(storage.Filesystem) {
		t.Error("file_tree should apply to filesystem")
	}
	if ft.AppliesTo(storage.StorageType("sqlite")) {
		t.Error("file_tree should not apply to a non-filesystem type")
	}

	p, _ := inspect.ParseParams("exact", -1, 0)
	ev := ft.Inspect(view, p)
	if view.ParseCount() != 0 {
		t.Errorf("file_tree opened %d files, want 0", view.ParseCount())
	}
	if ev.Inspector != "file_tree" || ev.Scope != dir {
		t.Errorf("file_tree evidence = %+v", ev)
	}
	// notes (.md, kebab) and assets (.png) are distinct directory profiles.
	if got := classTotal(t, ev); got != 2 {
		t.Errorf("distinct directory classes = %d, want 2", got)
	}
}

func TestFileTreeContent_parsesMarkdown(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes/dune.md", "---\ntitle: Dune\n---\n# Dune\n")

	view, err := inspect.NewSourceView(dir)
	if err != nil {
		t.Fatalf("NewSourceView: %v", err)
	}
	_ = inspect.FileTreeContent{}.Inspect(view, inspect.Params{})
	if view.ParseCount() == 0 {
		t.Error("file_tree_content should parse markdown (ParseCount > 0)")
	}
}

func TestDocumentShape_clustersOnCompositeFingerprint(t *testing.T) {
	dir := t.TempDir()
	// Identical across all dimensions → one class.
	writeFile(t, dir, "books/dune.md", "---\ntitle: Dune\nrating: 5\n---\n# Dune\n\n## Review\n")
	writeFile(t, dir, "books/messiah.md", "---\ntitle: Messiah\nrating: 4\n---\n# Messiah\n\n## Review\n")
	// Same frontmatter keys, different body skeleton (Summary, not Review) →
	// a different class, proving clustering is not on frontmatter alone.
	writeFile(t, dir, "books/notes.md", "---\ntitle: Notes\nrating: 3\n---\n# Notes\n\n## Summary\n")

	view, err := inspect.NewSourceView(dir)
	if err != nil {
		t.Fatalf("NewSourceView: %v", err)
	}
	p, _ := inspect.ParseParams("exact", -1, 0)
	ev := inspect.DocumentShape{}.Inspect(view, p)

	classes := ev.Data["classes"].([]any)
	if len(classes) != 1 {
		t.Fatalf("classes = %d, want 1 (dune+messiah)", len(classes))
	}
	if classes[0].(map[string]any)["size"].(int) != 2 {
		t.Errorf("class size = %v, want 2", classes[0].(map[string]any)["size"])
	}
	if outliers := ev.Data["outliers"].([]any); len(outliers) != 1 {
		t.Errorf("outliers = %d, want 1 (notes, distinct body)", len(outliers))
	}
}

// classTotal counts distinct classes (non-singleton classes plus singleton
// outliers) in a summarized evidence payload.
func classTotal(t *testing.T, ev inspect.Evidence) int {
	t.Helper()
	return len(ev.Data["classes"].([]any)) + len(ev.Data["outliers"].([]any))
}
