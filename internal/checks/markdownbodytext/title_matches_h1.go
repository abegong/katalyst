package markdownbodytext

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"gopkg.in/yaml.v3"
)

// titleMatchesArgs is markdown_title_matches_h1's own config shape. It rejects
// schema and defaults field to "title".
type titleMatchesArgs struct {
	Field  string `yaml:"field"`
	Schema string `yaml:"schema"`
}

// MarkdownTitleMatchesH1 checks that a frontmatter field matches the first H1.
type MarkdownTitleMatchesH1 struct {
	Field string
}

func (m MarkdownTitleMatchesH1) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + m.Field
	raw, ok := ctx.Meta[m.Field]
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing frontmatter field %q", m.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	title, ok := raw.(string)
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("frontmatter field %q must be a string", m.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}

	h1, h1Line, found := firstH1(ctx.Doc.Body, ctx.Doc.BodyLine)
	if !found {
		return []checks.Violation{{
			Path:    "/",
			Message: "missing H1 heading in markdown body",
			Line:    0,
		}}
	}
	if strings.TrimSpace(title) == h1 {
		return nil
	}
	return []checks.Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("%q does not match first H1 %q", title, h1),
		Line:    h1Line,
	}}
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckMarkdownTitleMatchesH1,
		Family:    "markdownBodyText",
		Slug:      "title-matches-h1",
		Title:     "Title matches H1",
		Summary:   "Require a frontmatter field to match the first H1 heading in the body.",
		Fields: []checks.Field{
			{Name: "field", Required: false, Default: "title", Desc: "Frontmatter key compared to the first H1."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_title_matches_h1
        field: title`,
	}, func(n *yaml.Node) (any, error) {
		var a titleMatchesArgs
		if n != nil {
			if err := n.Decode(&a); err != nil {
				return nil, err
			}
		}
		if a.Schema != "" {
			return nil, errors.New(`markdown_title_matches_h1 does not support "schema"`)
		}
		if a.Field == "" {
			a.Field = "title"
		}
		return a, nil
	}, func(a any) checks.Check {
		return MarkdownTitleMatchesH1{Field: a.(titleMatchesArgs).Field}
	}, nil)
}
