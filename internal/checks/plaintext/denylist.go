package plaintext

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
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

type denylistArgs struct {
	Values  []string `yaml:"values"`
	Target  string   `yaml:"target"`
	Select  string   `yaml:"select"`
	Pattern string   `yaml:"pattern"`
	Match   string   `yaml:"match"`
	Fix     string   `yaml:"fix"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckTextDenylist,
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
	}, checks.ParseInto(func(a denylistArgs) error {
		if err := argcheck.RequireStrings("text_denylist", "values", a.Values); err != nil {
			return err
		}
		if a.Pattern != "" {
			return errors.New(`text_denylist does not support "pattern"`)
		}
		if a.Match != "" {
			return errors.New(`text_denylist does not support "match"`)
		}
		if a.Fix != "" {
			return errors.New(`text_denylist does not support "fix"`)
		}
		if err := validateTextTarget("text_denylist", a.Target); err != nil {
			return err
		}
		return validateSelect("text_denylist", a.Target, a.Select)
	}), func(a any) checks.Check {
		x := a.(denylistArgs)
		return TextDenylist{
			Values: x.Values,
			Target: x.Target,
			Select: CompileSelect(x.Select),
		}
	}, nil)
}
