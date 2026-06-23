package markdownbodytext_test

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/checktest"
	"github.com/abegong/katalyst/internal/checks/markdownbodytext"
)

func TestMarkdownTitleMatchesH1Run_detectsMismatch(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\n# Children of Dune\n")
	violations := markdownbodytext.MarkdownTitleMatchesH1{Field: "title"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Line != 4 {
		t.Fatalf("expected mismatch to report H1 line 4, got %d", violations[0].Line)
	}
}

func TestMarkdownTitleMatchesH1Run_acceptsMatch(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\n# Dune\n")
	violations := markdownbodytext.MarkdownTitleMatchesH1{Field: "title"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestMarkdownRequiresH1Run_detectsMissing(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\nNo heading\n")
	violations := markdownbodytext.MarkdownRequiresH1{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "missing H1") {
		t.Fatalf("expected requires-h1 violation, got %v", violations)
	}
}

func TestMarkdownSingleH1Run_detectsMultiple(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\n# One\n# Two\n")
	violations := markdownbodytext.MarkdownSingleH1{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "only one H1") {
		t.Fatalf("expected single-h1 violation, got %v", violations)
	}
}

func TestMarkdownNoHeadingLevelJumpsRun_detectsJump(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\n# One\n### Jump\n")
	violations := markdownbodytext.MarkdownNoHeadingLevelJumps{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "jump") {
		t.Fatalf("expected heading-jump violation, got %v", violations)
	}
}

func TestMarkdownRequiredSectionRun_detectsMissing(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\n# Dune\n## Notes\n")
	violations := markdownbodytext.MarkdownRequiredSection{Heading: "Summary"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "required section") {
		t.Fatalf("expected required-section violation, got %v", violations)
	}
}

func TestMarkdownCodeFenceHasLanguageRun_detectsMissing(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\n```\ncode\n```\n")
	violations := markdownbodytext.MarkdownCodeFenceHasLanguage{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "language") {
		t.Fatalf("expected code-fence-language violation, got %v", violations)
	}
}
