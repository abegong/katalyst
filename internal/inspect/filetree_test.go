package inspect

import "testing"

func TestNamingBucket(t *testing.T) {
	tests := map[string]string{
		"old-notes": "kebab-case",
		"old_notes": "snake_case",
		"oldNotes":  "camelCase",
		"OldNotes":  "PascalCase",
		"Old Notes": "title/spaces",
		"lower":     "lowercase",
		"UPPER":     "uppercase",
		"123":       "numeric",
		"old.Notes": "mixed/other",
	}
	for stem, want := range tests {
		if got := namingBucket(stem); got != want {
			t.Errorf("namingBucket(%q) = %q, want %q", stem, got, want)
		}
	}
}

func TestBuildFileTreeSummary_regionsNamingAndRepresentatives(t *testing.T) {
	view := SourceView{files: []sourceFile{
		{rel: "README", dir: ".", ext: ""},
		{rel: "books/dune-book.md", dir: "books", ext: ".md"},
		{rel: "books/it-review.md", dir: "books", ext: ".md"},
		{rel: "books/messiah-notes.md", dir: "books", ext: ".md"},
		{rel: "books/Old Notes.md", dir: "books", ext: ".md"},
		{rel: "notes/reading-list.md", dir: "notes", ext: ".md"},
		{rel: "static/logo.png", dir: "static", ext: ".png"},
		{rel: "static/site.css", dir: "static", ext: ".css"},
	}}

	data := buildFileTreeSummary(view)
	if data["file_count"].(int) != 8 {
		t.Fatalf("file_count = %v, want 8", data["file_count"])
	}
	if data["dir_count"].(int) != 4 {
		t.Errorf("dir_count = %v, want 4", data["dir_count"])
	}
	if data["max_depth"].(int) != 2 {
		t.Errorf("max_depth = %v, want 2", data["max_depth"])
	}

	regions := data["top_level_regions"].([]any)
	first := regions[0].(map[string]any)
	if first["path"] != "books/" || first["file_count"].(int) != 4 {
		t.Errorf("top region = %v, want books/ with 4 files", first)
	}

	naming := data["naming"].(map[string]any)
	if naming["dominant_extension_scope"] != ".md" {
		t.Fatalf("dominant_extension_scope = %v, want .md", naming["dominant_extension_scope"])
	}
	byExt := naming["by_extension"].(map[string]any)
	mdNaming := byExt[".md"].(map[string]any)
	if mdNaming["dominant_bucket"] != "kebab-case" || mdNaming["dominant_count"].(int) != 4 {
		t.Errorf("markdown naming = %v, want kebab-case count 4", mdNaming)
	}
	exceptions := mdNaming["exceptions"].([]any)
	if len(exceptions) != 1 || exceptions[0].(map[string]any)["path"] != "books/Old Notes.md" {
		t.Errorf("exceptions = %v, want Old Notes.md", exceptions)
	}

	reps := data["representative_paths"].([]any)
	want := []string{"README", "books/Old Notes.md", "notes/reading-list.md", "static/logo.png"}
	for i, rel := range want {
		if reps[i] != rel {
			t.Errorf("representative_paths[%d] = %v, want %s", i, reps[i], rel)
		}
	}
}
