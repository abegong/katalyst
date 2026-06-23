package inspect

import "testing"

func TestMarkdownBody_headingShapeAndSections(t *testing.T) {
	docs := []mdInput{
		{Body: []byte("# Dune\n\n## Review\n\nText\n"), Title: "Dune"},
		{Body: []byte("# Other\n\n## Review\n\n## Notes\n"), Title: "Mismatch"},
	}
	data := markdownBody(docs)

	hs := data["heading_shape"].(map[string]any)
	if hs["bodies"].(int) != 2 {
		t.Errorf("bodies = %v, want 2", hs["bodies"])
	}
	if hs["single_h1"].(int) != 2 {
		t.Errorf("single_h1 = %v, want 2", hs["single_h1"])
	}
	if hs["h1_matches_title"].(int) != 1 {
		t.Errorf("h1_matches_title = %v, want 1", hs["h1_matches_title"])
	}

	sections := data["sections"].(map[string]any)
	if sections["Review"].(int) != 2 {
		t.Errorf("Review section count = %v, want 2", sections["Review"])
	}
	if sections["Notes"].(int) != 1 {
		t.Errorf("Notes section count = %v, want 1", sections["Notes"])
	}
}

func TestMarkdownBody_levelJump(t *testing.T) {
	docs := []mdInput{
		{Body: []byte("# A\n\n### Skipped\n")}, // H1 -> H3 is a level jump
	}
	data := markdownBody(docs)
	hs := data["heading_shape"].(map[string]any)
	if hs["has_level_jump"].(int) != 1 {
		t.Errorf("has_level_jump = %v, want 1", hs["has_level_jump"])
	}
}
