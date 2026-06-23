package inspect

import "testing"

func TestFileMetadata_casingExtensionsDepth(t *testing.T) {
	refs := []string{
		"notes/dune.md",     // kebab
		"notes/the_book.md", // snake
		"notes/My Book.md",  // other + space
		"a/b/c/deep.md",     // kebab, depth 3
		"image.png",         // kebab, .png
	}
	data := fileMetadata(refs)

	casing := data["casing"].(map[string]any)
	if casing["kebab"].(int) != 3 {
		t.Errorf("kebab = %v, want 3", casing["kebab"])
	}
	if casing["snake"].(int) != 1 {
		t.Errorf("snake = %v, want 1", casing["snake"])
	}
	if casing["other"].(int) != 1 {
		t.Errorf("other = %v, want 1", casing["other"])
	}

	if data["with_spaces"].(int) != 1 {
		t.Errorf("with_spaces = %v, want 1", data["with_spaces"])
	}

	exts := data["extensions"].(map[string]any)
	if exts[".md"].(int) != 4 {
		t.Errorf(".md = %v, want 4", exts[".md"])
	}
	if exts[".png"].(int) != 1 {
		t.Errorf(".png = %v, want 1", exts[".png"])
	}

	if data["max_depth"].(int) != 3 {
		t.Errorf("max_depth = %v, want 3", data["max_depth"])
	}
}
