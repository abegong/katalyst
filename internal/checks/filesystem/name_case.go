package filesystem

import (
	"fmt"
	"regexp"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// caseStyle pairs a style's anchored pattern with its human label.
type caseStyle struct {
	pattern *regexp.Regexp
	label   string
}

var caseStyles = map[string]caseStyle{
	"kebab":           {regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`), "kebab-case"},
	"snake":           {regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`), "snake_case"},
	"screaming-snake": {regexp.MustCompile(`^[A-Z0-9]+(?:_[A-Z0-9]+)*$`), "SCREAMING_SNAKE_CASE"},
	"camel":           {regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`), "camelCase"},
	"pascal":          {regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`), "PascalCase"},
	"point":           {regexp.MustCompile(`^[a-z0-9]+(?:\.[a-z0-9]+)*$`), "point.case"},
	"lower":           {regexp.MustCompile(`^[^A-Z]*$`), "lowercase"},
}

// CaseStyles returns the set of supported style keys.
func CaseStyles() []string {
	return []string{"kebab", "snake", "screaming-snake", "camel", "pascal", "point", "lower"}
}

// NameCase checks that the target conforms to a case style.
type NameCase struct {
	Style  string
	Target string
}

func (c NameCase) Run(ctx checks.Context) []checks.Violation {
	style, ok := caseStyles[c.Style]
	if !ok {
		return nil // unknown style is rejected at config load
	}
	var out []checks.Violation
	noun := targetNoun(c.Target)
	for _, v := range resolveTarget(ctx, c.Target) {
		if !style.pattern.MatchString(v) {
			out = append(out, checks.Violation{
				Path:    "/",
				Message: fmt.Sprintf("%s %q must be %s", noun, v, style.label),
			})
		}
	}
	return out
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckFilesystemNameCase,
		Family:    "fileSystem",
		Slug:      "name-case",
		Title:     "Name case",
		Summary:   "Require a name (or path segments) to follow a case style.",
		Fields: []checks.Field{
			{Name: "style", Required: true, Desc: "One of `kebab`, `snake`, `screaming-snake`, `camel`, `pascal`, `point`, `lower`."},
			{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, `parent-dir`, or `path-segments`."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_case
        style: kebab`,
	}, func(ch config.CheckInstance) checks.Check {
		return NameCase{Style: ch.Style, Target: ch.Target}
	}, nil)
}
