package plaintext

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"regexp"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
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

type requiresArgs struct {
	Pattern string `yaml:"pattern"`
	Target  string `yaml:"target"`
	Select  string `yaml:"select"`
	Match   string `yaml:"match"`
	Fix     string `yaml:"fix"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckTextRequires,
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
	}, func(n *yaml.Node) (any, error) {
		var a requiresArgs
		if n != nil {
			if err := n.Decode(&a); err != nil {
				return nil, err
			}
		}
		if err := argcheck.RequireString("text_requires", "pattern", a.Pattern); err != nil {
			return nil, err
		}
		if _, err := regexp.Compile(a.Pattern); err != nil {
			return nil, fmt.Errorf("text_requires: invalid pattern %q: %w", a.Pattern, err)
		}
		if a.Match == "" {
			a.Match = "any"
		}
		if a.Match != "any" && a.Match != "all" {
			return nil, fmt.Errorf(`text_requires: "match" must be any or all (got %q)`, a.Match)
		}
		if a.Fix != "" {
			return nil, errors.New(`text_requires does not support "fix"`)
		}
		if err := validateTextTarget("text_requires", a.Target); err != nil {
			return nil, err
		}
		if err := validateSelect("text_requires", a.Target, a.Select); err != nil {
			return nil, err
		}
		return a, nil
	}, func(a any) checks.Check {
		x := a.(requiresArgs)
		return TextRequires{
			Re:      regexp.MustCompile(x.Pattern),
			Pattern: x.Pattern,
			Target:  x.Target,
			Select:  CompileSelect(x.Select),
			All:     x.Match == "all",
		}
	}, nil)
}
