package checks_test

import (
	"strings"
	"testing"

	"github.com/katabase-ai/katalyst/internal/checks"
	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/katabase-ai/katalyst/internal/validator"
)

func mustParseDoc(t *testing.T, src string) *frontmatter.Document {
	t.Helper()
	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func mustLoadSchema(t *testing.T, src string) *validator.Schema {
	t.Helper()
	s, err := validator.Load("test-schema", strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load schema: %v", err)
	}
	return s
}

func TestObjectRun_reportsLineForSchemaViolation(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\nyear: nope\n---\n# Dune\n")
	schema := mustLoadSchema(t, `{"type":"object","properties":{"year":{"type":"integer"}},"required":["year"]}`)
	meta := map[string]any{
		"title": "Dune",
		"year":  "nope",
	}

	violations := checks.Object{Schema: schema}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Path != "/year" {
		t.Fatalf("expected /year path, got %q", violations[0].Path)
	}
	if violations[0].Line != 3 {
		t.Fatalf("expected line 3, got %d", violations[0].Line)
	}
}

func TestMarkdownTitleMatchesH1Run_detectsMismatch(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# Children of Dune\n")
	meta := map[string]any{"title": "Dune"}

	violations := checks.MarkdownTitleMatchesH1{Field: "title"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Line != 4 {
		t.Fatalf("expected mismatch to report H1 line 4, got %d", violations[0].Line)
	}
}

func TestMarkdownTitleMatchesH1Run_acceptsMatch(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# Dune\n")
	meta := map[string]any{"title": "Dune"}

	violations := checks.MarkdownTitleMatchesH1{Field: "title"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestFilenameMatchesSlugRun_detectsMismatch(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune-messiah\n---\n# Dune Messiah\n")
	meta := map[string]any{"slug": "dune-messiah"}

	violations := checks.FilenameMatchesSlug{Field: "slug"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Line != 2 {
		t.Fatalf("expected line 2 for slug key, got %d", violations[0].Line)
	}
}
