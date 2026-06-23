package markdownbodytext_test

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/checktest"
	"github.com/abegong/katalyst/internal/checks/markdownbodytext"
)

func TestMarkdownWritingTellsRun_warnsOnTellsInBodyAndFrontmatter(t *testing.T) {
	// Em dash in the body, an overused word in the body, and a curly quote
	// in the frontmatter title.
	doc := checktest.MustParseDoc(t, "---\ntitle: Don’t Panic\n---\n# OK\n\nWe delve in — carefully.\n")
	violations := markdownbodytext.MarkdownWritingTells{}.Run(checks.Context{
		FilePath: "notes/x.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Don’t Panic"},
	})
	if len(violations) == 0 {
		t.Fatal("expected writing-tell warnings, got none")
	}
	for _, v := range violations {
		if v.Severity != checks.SeverityWarning {
			t.Errorf("tells must be warnings, got %v for %q", v.Severity, v.Message)
		}
	}
	joined := ""
	for _, v := range violations {
		joined += v.Message + "\n"
	}
	for _, want := range []string{"em dash", "delve", "curly single quote"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected a %q tell, got:\n%s", want, joined)
		}
	}
}

func TestMarkdownWritingTellsRun_cleanProseHasNoTells(t *testing.T) {
	// Plain ASCII prose with legitimate technical typography that must NOT
	// be flagged.
	doc := checktest.MustParseDoc(t, "---\ntitle: Clean\n---\n# Clean\n\nMap name -> path when year >= 1900. See A -> B and x >= y.\n")
	violations := markdownbodytext.MarkdownWritingTells{}.Run(checks.Context{
		FilePath: "notes/clean.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Clean"},
	})
	if len(violations) != 0 {
		t.Fatalf("expected no tells for clean prose, got %v", violations)
	}
}
