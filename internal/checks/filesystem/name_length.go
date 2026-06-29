package filesystem

import (
	"fmt"
	"unicode/utf8"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
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

type nameLengthArgs struct {
	Min    *float64 `yaml:"min"`
	Max    *float64 `yaml:"max"`
	Target string   `yaml:"target"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckFilesystemNameLength,
		Family:    "fileSystem",
		Targets:   []string{checks.TargetCollection, checks.TargetFilesystem},
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
	}, checks.ParseInto(func(a nameLengthArgs) error {
		if err := argcheck.RequireOneOfFields("filesystem_name_length", a.Min != nil || a.Max != nil, "min", "max"); err != nil {
			return err
		}
		return validateTarget("filesystem_name_length", a.Target)
	}), func(a any) checks.Check {
		x := a.(nameLengthArgs)
		return NameLength{Min: intPtr(x.Min), Max: intPtr(x.Max), Target: x.Target}
	}, nil)
}
