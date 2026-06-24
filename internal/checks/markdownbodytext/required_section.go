package markdownbodytext

import (
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// requiredSectionArgs is markdown_required_section's own config shape.
type requiredSectionArgs struct {
	Heading string `yaml:"heading"`
}

// MarkdownRequiredSection checks that a specific heading exists.
type MarkdownRequiredSection struct {
	Heading string
}

func (m MarkdownRequiredSection) Run(ctx checks.Context) []checks.Violation {
	target := strings.TrimSpace(m.Heading)
	for _, line := range checks.MarkdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		_, text, ok := heading(line.Text)
		if !ok {
			continue
		}
		if text == target {
			return nil
		}
	}
	return []checks.Violation{{
		Path:    "/",
		Message: fmt.Sprintf("missing required section heading %q", target),
	}}
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckMarkdownRequiredSection,
		Family:    "markdownBodyText",
		Slug:      "required-section",
		Title:     "Required section",
		Summary:   "Require that a heading with specific text exists somewhere in the body.",
		Fields: []checks.Field{
			{Name: "heading", Required: true, Desc: "Heading text that must appear."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_required_section
        heading: Summary`,
	}, checks.ParseInto(func(a requiredSectionArgs) error {
		return argcheck.RequireString("markdown_required_section", "heading", a.Heading)
	}), func(a any) checks.Check {
		return MarkdownRequiredSection{Heading: a.(requiredSectionArgs).Heading}
	}, nil)
}
