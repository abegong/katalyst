package plaintext

import (
	"fmt"
	"regexp"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// TextRequires asserts that an unanchored regex appears in the body. With
// All (match: all) every selected span must match; otherwise (match: any) at
// least one must.
type TextRequires struct {
	Re      *regexp.Regexp
	Pattern string
	Target  string
	Select  *regexp.Regexp
	All     bool
}

func (t TextRequires) Run(ctx checks.Context) []checks.Violation {
	spans := textSpans(ctx, t.Target, t.Select)
	if t.All {
		var out []checks.Violation
		for _, s := range spans {
			if !t.Re.MatchString(s.Text) {
				out = append(out, checks.Violation{
					Path:    "/",
					Message: fmt.Sprintf("required text /%s/ not found", t.Pattern),
					Line:    s.Line,
				})
			}
		}
		return out
	}
	for _, s := range spans {
		if t.Re.MatchString(s.Text) {
			return nil
		}
	}
	return []checks.Violation{{
		Path:    "/",
		Message: fmt.Sprintf("required text /%s/ not found", t.Pattern),
		Line:    ctx.Doc.BodyLine,
	}}
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckTextRequires,
		Family:    "plainText",
		Slug:      "requires",
		Title:     "Requires",
		Summary:   "Require a regular expression to appear in the body text.",
		Fields: []checks.Field{
			{Name: "pattern", Required: true, Desc: "Go regular expression, matched unanchored (appears somewhere in the span)."},
			{Name: "target", Required: false, Default: "body", Desc: "Span selector: body, line, first-line, or matched-lines."},
			{Name: "select", Required: false, Desc: "Line-filter regex; required for and only valid with target matched-lines."},
			{Name: "match", Required: false, Default: "any", Desc: "For multi-span targets: any (at least one span matches) or all (every span matches)."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: text_requires
        pattern: Sources`,
	}, func(ch config.CheckInstance) checks.Check {
		return TextRequires{
			Re:      regexp.MustCompile(ch.Pattern),
			Pattern: ch.Pattern,
			Target:  ch.Target,
			Select:  CompileSelect(ch.Select),
			All:     ch.Match == "all",
		}
	}, nil)
}
