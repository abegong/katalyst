package inspect_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
)

func TestFileTree_opensNothingAndReportsFilesystemMap(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes/dune.md", "---\ntitle: Dune\n---\n# Dune\n\n## Review\n")
	writeFile(t, dir, "notes/messiah.md", "---\ntitle: Messiah\n---\n# Messiah\n\n## Review\n")
	writeFile(t, dir, "assets/logo.png", "binary")

	view, err := inspect.NewSourceView(dir)
	if err != nil {
		t.Fatalf("NewSourceView: %v", err)
	}

	ft := inspect.FileTree{}
	if !ft.AppliesTo("filesystem") {
		t.Error("file_tree should apply to filesystem")
	}
	if ft.AppliesTo("sqlite") {
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
	if got := ev.Data["file_count"].(int); got != 3 {
		t.Errorf("file_count = %d, want 3", got)
	}
	if got := ev.Data["dir_count"].(int); got != 3 {
		t.Errorf("dir_count = %d, want 3", got)
	}
	if got := ev.Data["max_depth"].(int); got != 2 {
		t.Errorf("max_depth = %d, want 2", got)
	}
	extensions := ev.Data["extensions"].(map[string]any)
	if extensions[".md"].(int) != 2 || extensions[".png"].(int) != 1 {
		t.Errorf("extensions = %v, want .md=2 .png=1", extensions)
	}
	regions := ev.Data["top_level_regions"].([]any)
	if len(regions) != 2 {
		t.Fatalf("regions = %d, want 2", len(regions))
	}
	first := regions[0].(map[string]any)
	if first["path"] != "notes/" || first["file_count"].(int) != 2 {
		t.Errorf("first region = %v, want notes/ with 2 files", first)
	}
	if len(ev.Data["tree_entries"].([]any)) == 0 {
		t.Errorf("small file tree should include tree_entries")
	}
}

func TestFileContentShape_profilesSelectedMarkdown(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "notes/dune.md", "---\ntitle: Dune\n---\n# Dune\n")
	writeFile(t, dir, "data/books.csv", "title,rating\nDune,5\n")

	view, err := inspect.NewSourceView(dir)
	if err != nil {
		t.Fatalf("NewSourceView: %v", err)
	}
	ev := inspect.FileContentShape{}.Inspect(view, inspect.Params{}.WithSelection(inspect.ParseSelection(`ext = ".md"`)))
	if view.ParseCount() == 0 {
		t.Error("file_content_shape should open selected files (ParseCount > 0)")
	}
	if ev.Inspector != "file_content_shape" {
		t.Errorf("inspector = %q, want file_content_shape", ev.Inspector)
	}
	if got := ev.Data["file_count"].(int); got != 1 {
		t.Errorf("file_count = %d, want selected markdown file only", got)
	}
	md := ev.Data["markdown"].(map[string]any)
	if got := md["files"].(int); got != 1 {
		t.Errorf("markdown.files = %d, want 1", got)
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
