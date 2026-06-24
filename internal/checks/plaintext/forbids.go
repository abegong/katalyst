package plaintext

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// TextForbids asserts that an unanchored regex appears in no selected span.
type TextForbids struct {
	Re      *regexp.Regexp
	Pattern string
	Target  string
	Select  *regexp.Regexp
	// Fix is an optional replacement template applied to the matched text by
	// the fix command; empty means report-only.
	Fix string
}

func (t TextForbids) Run(ctx checks.Context) []checks.Violation {
	var out []checks.Violation
	for _, s := range textSpans(ctx, t.Target, t.Select) {
		if loc := t.Re.FindStringIndex(s.Text); loc != nil {
			out = append(out, checks.Violation{
				Path:    "/",
				Message: fmt.Sprintf("forbidden text /%s/ found", t.Pattern),
				Line:    lineOf(s, loc[0]),
			})
		}
	}
	return out
}

// ApplyFix returns body with the forbidden pattern replaced by the rule's fix
// template (regexp capture syntax) across its selected spans. It replaces only
// matched substrings, never whole spans; a rule with no fix returns body
// unchanged.
func (t TextForbids) ApplyFix(body []byte) []byte {
	if t.Fix == "" {
		return body
	}
	if t.Target == "" || t.Target == "body" {
		return t.Re.ReplaceAll(body, []byte(t.Fix))
	}
	lines := strings.Split(string(body), "\n")
	switch t.Target {
	case "first-line":
		for i := range lines {
			if strings.TrimSpace(lines[i]) != "" {
				lines[i] = t.Re.ReplaceAllString(lines[i], t.Fix)
				break
			}
		}
	case "line":
		for i := range lines {
			lines[i] = t.Re.ReplaceAllString(lines[i], t.Fix)
		}
	case "matched-lines":
		for i := range lines {
			if t.Select != nil && t.Select.MatchString(lines[i]) {
				lines[i] = t.Re.ReplaceAllString(lines[i], t.Fix)
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

type forbidsArgs struct {
	Pattern string `yaml:"pattern"`
	Target  string `yaml:"target"`
	Select  string `yaml:"select"`
	Fix     string `yaml:"fix"`
	Match   string `yaml:"match"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckTextForbids,
		Family:    "plainText",
		Slug:      "forbids",
		Title:     "Forbids",
		Summary:   "Forbid a regular expression from appearing in the body text.",
		Fields: []checks.Field{
			{Name: "pattern", Required: true, Desc: "Go regular expression, matched unanchored."},
			{Name: "target", Required: false, Default: "body", Desc: "Span selector: body, line, first-line, or matched-lines."},
			{Name: "select", Required: false, Desc: "Line-filter regex; required for and only valid with target matched-lines."},
			{Name: "fix", Required: false, Desc: "Optional replacement template (regexp capture syntax) applied to the matched text by the fix command."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: text_forbids
        target: line
        pattern: '\bTODO\b'`,
	}, checks.ParseInto(func(a forbidsArgs) error {
		if err := argcheck.RequireString("text_forbids", "pattern", a.Pattern); err != nil {
			return err
		}
		if _, err := regexp.Compile(a.Pattern); err != nil {
			return fmt.Errorf("text_forbids: invalid pattern %q: %w", a.Pattern, err)
		}
		if a.Match != "" {
			return errors.New(`text_forbids does not support "match"`)
		}
		if err := validateTextTarget("text_forbids", a.Target); err != nil {
			return err
		}
		return validateSelect("text_forbids", a.Target, a.Select)
	}), func(a any) checks.Check {
		x := a.(forbidsArgs)
		return TextForbids{
			Re:      regexp.MustCompile(x.Pattern),
			Pattern: x.Pattern,
			Target:  x.Target,
			Select:  CompileSelect(x.Select),
			Fix:     x.Fix,
		}
	}, nil)
}
