package markdownbodytext

import (
	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project/config"
)

// MarkdownSingleH1 checks that only one H1 is present.
type MarkdownSingleH1 struct{}

func (m MarkdownSingleH1) Run(ctx checks.Context) []checks.Violation {
	h1Lines := make([]int, 0)
	for _, line := range checks.MarkdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		if level, _, ok := heading(line.Text); ok && level == 1 {
			h1Lines = append(h1Lines, line.Line)
		}
	}
	if len(h1Lines) <= 1 {
		return nil
	}
	return []checks.Violation{{
		Path:    "/",
		Message: "markdown body must contain only one H1 heading",
		Line:    h1Lines[1],
	}}
}

func init() {
	register(checks.Descriptor{
		CheckType: config.CheckMarkdownSingleH1,
		Family:    "markdownBodyText",
		Slug:      "single-h1",
		Title:     "Single H1",
		Summary:   "Require that the markdown body contains at most one H1 heading.",
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_single_h1`,
	}, func(ch config.CheckInstance) checks.Check {
		return MarkdownSingleH1{}
	}, nil)
}
