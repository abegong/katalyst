package markdownbodytext

import (
	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project/config"
)

// MarkdownRequiresH1 checks that the body has at least one H1.
type MarkdownRequiresH1 struct{}

func (m MarkdownRequiresH1) Run(ctx checks.Context) []checks.Violation {
	_, _, ok := firstH1(ctx.Doc.Body, ctx.Doc.BodyLine)
	if ok {
		return nil
	}
	return []checks.Violation{{
		Path:    "/",
		Message: "missing H1 heading in markdown body",
	}}
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: config.CheckMarkdownRequiresH1,
		Family:    "markdownBodyText",
		Slug:      "requires-h1",
		Title:     "Requires H1",
		Summary:   "Require at least one H1 heading in the markdown body.",
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1`,
	}, checks.NoArgs, func(any) checks.Check {
		return MarkdownRequiresH1{}
	}, nil)
}
