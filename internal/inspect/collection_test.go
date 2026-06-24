package inspect_test

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/project/config"
)

func TestCollectionView_objectFieldsAndMarkdownBody(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".katalyst/storage/local.yaml", `type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
`)
	writeFile(t, dir, "notes/dune.md", "---\ntitle: Dune\nrating: 5\n---\n# Dune\n\n## Review\n")
	writeFile(t, dir, "notes/messiah.md", "---\ntitle: Messiah\nrating: 4\n---\n# Messiah\n\n## Review\n")
	writeFile(t, dir, "notes/draft.md", "---\ntitle: Draft\n---\n# Draft\n")

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	proj := project.New(cfg)
	c, ok := proj.Collection("notes")
	if !ok {
		t.Fatal("collection notes not found")
	}
	view, err := inspect.NewCollectionView(proj, c)
	if err != nil {
		t.Fatalf("NewCollectionView: %v", err)
	}

	// Items are addressed by id (the filename stem), never by raw path.
	ids := view.IDs()
	if len(ids) != 3 {
		t.Fatalf("ids = %v, want 3", ids)
	}
	for _, id := range ids {
		if strings.ContainsAny(id, "/\\") {
			t.Errorf("id %q looks like a path, want a bare item id", id)
		}
	}

	of := inspect.ObjectFields{}.Inspect(view, inspect.Params{})
	if of.Inspector != "object_fields" || of.Scope != "notes" || of.N != 3 {
		t.Errorf("object_fields evidence = %+v", of)
	}
	rating := of.Data["rating"].(map[string]any)
	if rating["present"].(int) != 2 {
		t.Errorf("rating present = %v, want 2 (draft has no rating)", rating["present"])
	}

	mb := inspect.MarkdownBody{}.Inspect(view, inspect.Params{})
	hs := mb.Data["heading_shape"].(map[string]any)
	if hs["bodies"].(int) != 3 {
		t.Errorf("bodies = %v, want 3", hs["bodies"])
	}
	sections := mb.Data["sections"].(map[string]any)
	if sections["Review"].(int) != 2 {
		t.Errorf("Review section count = %v, want 2", sections["Review"])
	}
}
