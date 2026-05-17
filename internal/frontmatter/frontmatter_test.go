package frontmatter_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/katabase-ai/katabridge/internal/frontmatter"
)

func TestParse_extractsYAMLFrontmatter(t *testing.T) {
	src := strings.Join([]string{
		"---",
		"title: Dune",
		"year: 1965",
		"tags:",
		"  - sci-fi",
		"  - classic",
		"---",
		"",
		"# Dune",
		"",
		"A story about spice.",
	}, "\n")

	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}

	if !doc.HasFrontmatter {
		t.Fatalf("expected HasFrontmatter=true")
	}

	want := map[string]any{
		"title": "Dune",
		"year":  1965,
		"tags":  []any{"sci-fi", "classic"},
	}
	if !reflect.DeepEqual(doc.Meta, want) {
		t.Errorf("Meta mismatch:\n got: %#v\nwant: %#v", doc.Meta, want)
	}

	wantBody := "\n# Dune\n\nA story about spice."
	if string(doc.Body) != wantBody {
		t.Errorf("Body mismatch:\n got: %q\nwant: %q", string(doc.Body), wantBody)
	}
}

func TestParse_noFrontmatter(t *testing.T) {
	src := "# Just a heading\n\nNo frontmatter here.\n"

	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if doc.HasFrontmatter {
		t.Errorf("expected HasFrontmatter=false")
	}
	if doc.Meta != nil {
		t.Errorf("expected nil Meta, got %#v", doc.Meta)
	}
	if string(doc.Body) != src {
		t.Errorf("expected Body to equal the whole input when no frontmatter present")
	}
}

func TestParse_emptyFrontmatter(t *testing.T) {
	src := "---\n---\nbody\n"

	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !doc.HasFrontmatter {
		t.Errorf("expected HasFrontmatter=true even when block is empty")
	}
	if len(doc.Meta) != 0 {
		t.Errorf("expected empty Meta, got %#v", doc.Meta)
	}
	if string(doc.Body) != "body\n" {
		t.Errorf("Body mismatch: got %q", string(doc.Body))
	}
}

func TestParse_unterminatedFrontmatter(t *testing.T) {
	src := "---\ntitle: Dune\n\n# Body\n"

	_, err := frontmatter.Parse([]byte(src))
	if err == nil {
		t.Fatalf("expected error for unterminated frontmatter")
	}
	if !errors.Is(err, frontmatter.ErrUnterminated) {
		t.Errorf("expected ErrUnterminated, got %v", err)
	}
}

func TestParse_malformedYAML(t *testing.T) {
	src := "---\ntitle: : :\n---\nbody\n"

	_, err := frontmatter.Parse([]byte(src))
	if err == nil {
		t.Fatalf("expected error for malformed YAML")
	}
	if !errors.Is(err, frontmatter.ErrInvalidYAML) {
		t.Errorf("expected ErrInvalidYAML, got %v", err)
	}
}

func TestParse_crlfLineEndings(t *testing.T) {
	src := "---\r\ntitle: Dune\r\n---\r\nbody\r\n"

	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !doc.HasFrontmatter {
		t.Fatalf("expected HasFrontmatter=true")
	}
	if doc.Meta["title"] != "Dune" {
		t.Errorf("expected title=Dune, got %#v", doc.Meta["title"])
	}
}

// A leading BOM is common when files are authored on Windows.
func TestParse_leadingBOM(t *testing.T) {
	src := "\ufeff---\ntitle: Dune\n---\nbody\n"

	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !doc.HasFrontmatter {
		t.Errorf("expected HasFrontmatter=true after BOM")
	}
}

// The opening "---" fence is only meaningful at the very top of the file.
// A "---" later in the body is a thematic break, not frontmatter.
func TestParse_fenceMustBeAtTop(t *testing.T) {
	src := "\n---\ntitle: Dune\n---\nbody\n"

	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if doc.HasFrontmatter {
		t.Errorf("expected HasFrontmatter=false when leading line is blank")
	}
}
