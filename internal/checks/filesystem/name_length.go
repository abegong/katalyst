package filesystem

import (
	"fmt"
	"unicode/utf8"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project/config"
)

// NameLength bounds the character length of the target. At least one of
// Min/Max is set (enforced at config load).
type NameLength struct {
	Min    *int
	Max    *int
	Target string
}

func (c NameLength) Run(ctx checks.Context) []checks.Violation {
	var out []checks.Violation
	noun := targetNoun(c.Target)
	for _, v := range resolveTarget(ctx, c.Target) {
		n := utf8.RuneCountInString(v)
		if c.Min != nil && n < *c.Min {
			out = append(out, checks.Violation{
				Path:    "/",
				Message: fmt.Sprintf("%s %q must be at least %d characters", noun, v, *c.Min),
			})
		}
		if c.Max != nil && n > *c.Max {
			out = append(out, checks.Violation{
				Path:    "/",
				Message: fmt.Sprintf("%s %q must be at most %d characters", noun, v, *c.Max),
			})
		}
	}
	return out
}

func init() {
	register(checks.Descriptor{
		CheckType: config.CheckFilesystemNameLength,
		Family:    "fileSystem",
		Slug:      "name-length",
		Title:     "Name length",
		Summary:   "Bound the character length of a name.",
		Fields: []checks.Field{
			{Name: "min", Required: false, Desc: "Minimum length (at least one of min/max)."},
			{Name: "max", Required: false, Desc: "Maximum length (at least one of min/max)."},
			{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, `parent-dir`, or `path-segments`."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_length
        max: 80`,
	}, func(ch config.CheckInstance) checks.Check {
		return NameLength{Min: ch.MinInt, Max: ch.MaxInt, Target: ch.Target}
	}, nil)
}
