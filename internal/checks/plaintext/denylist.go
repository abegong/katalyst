package plaintext

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// TextDenylist forbids any of a list of literal substrings. Matching is
// literal (regex metacharacters are inert) via strings.Index.
type TextDenylist struct {
	Values []string
	Target string
	Select *regexp.Regexp
}

func (t TextDenylist) Run(ctx checks.Context) []checks.Violation {
	var out []checks.Violation
	for _, s := range textSpans(ctx, t.Target, t.Select) {
		for _, v := range t.Values {
			if idx := strings.Index(s.Text, v); idx >= 0 {
				out = append(out, checks.Violation{
					Path:    "/",
					Message: fmt.Sprintf("forbidden term %q found", v),
					Line:    lineOf(s, idx),
				})
			}
		}
	}
	return out
}

func init() {
	register(checks.Descriptor{
		CheckType: config.CheckTextDenylist,
		Family:    "plainText",
		Slug:      "denylist",
		Title:     "Denylist",
		Summary:   "Forbid any of a list of literal substrings in the body text.",
		Fields: []checks.Field{
			{Name: "values", Required: true, Desc: "Literal substrings to forbid; regex metacharacters are inert."},
			{Name: "target", Required: false, Default: "body", Desc: "Span selector: body, line, first-line, or matched-lines."},
			{Name: "select", Required: false, Desc: "Line-filter regex; required for and only valid with target matched-lines."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: text_denylist
        values: [TODO, FIXME, XXX]`,
	}, func(ch config.CheckInstance) checks.Check {
		return TextDenylist{
			Values: ch.Values,
			Target: ch.Target,
			Select: CompileSelect(ch.Select),
		}
	}, nil)
}
