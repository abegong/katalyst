package markdownbodytext

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// MarkdownNoHeadingLevelJumps checks that heading levels increase at most by one.
type MarkdownNoHeadingLevelJumps struct{}

func (m MarkdownNoHeadingLevelJumps) Run(ctx checks.Context) []checks.Violation {
	prevLevel := 0
	for _, line := range checks.MarkdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		level, _, ok := heading(line.Text)
		if !ok {
			continue
		}
		if prevLevel > 0 && level > prevLevel+1 {
			return []checks.Violation{{
				Path:    "/",
				Message: fmt.Sprintf("heading level jump from H%d to H%d is not allowed", prevLevel, level),
				Line:    line.Line,
			}}
		}
		prevLevel = level
	}
	return nil
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckMarkdownNoHeadingLevelJumps,
		Family:    "markdownBodyText",
		Slug:      "no-heading-level-jumps",
		Title:     "No Heading Level Jumps",
		Summary:   "Disallow jumps larger than one heading level (for example `H1 -> H3`).",
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_no_heading_level_jumps`,
	}, func(ch config.CheckInstance) checks.Check {
		return MarkdownNoHeadingLevelJumps{}
	}, nil)
}
