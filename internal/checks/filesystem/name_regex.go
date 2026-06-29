package filesystem

import (
	"fmt"
	"regexp"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// NameRegex checks that the target matches an anchored pattern.
type NameRegex struct {
	Re      *regexp.Regexp
	Pattern string
	Target  string
}

func (c NameRegex) Run(ctx checks.Context) []checks.Violation {
	if c.Re == nil {
		return nil
	}
	var out []checks.Violation
	noun := targetNoun(c.Target)
	for _, v := range resolveTarget(ctx, c.Target) {
		if !c.Re.MatchString(v) {
			out = append(out, checks.Violation{
				Path:    "/",
				Message: fmt.Sprintf("%s %q must match pattern %q", noun, v, c.Pattern),
			})
		}
	}
	return out
}

type nameRegexArgs struct {
	Pattern string `yaml:"pattern"`
	Target  string `yaml:"target"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckFilesystemNameRegex,
		Family:    "fileSystem",
		Targets:   []string{checks.TargetCollection, checks.TargetFilesystem},
		Slug:      "name-regex",
		Title:     "Name regex",
		Summary:   "Require a name to match a regular expression (anchored).",
		Fields: []checks.Field{
			{Name: "pattern", Required: true, Desc: "Regular expression; matched anchored (`^pattern$`)."},
			{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, `parent-dir`, or `path-segments`."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_regex
        pattern: '[0-9]{4}-[a-z-]+'`,
	}, checks.ParseInto(func(a nameRegexArgs) error {
		if err := argcheck.RequireString("filesystem_name_regex", "pattern", a.Pattern); err != nil {
			return err
		}
		if _, err := regexp.Compile(checks.AnchoredPattern(a.Pattern)); err != nil {
			return fmt.Errorf("filesystem_name_regex: invalid pattern %q: %w", a.Pattern, err)
		}
		return validateTarget("filesystem_name_regex", a.Target)
	}), func(a any) checks.Check {
		x := a.(nameRegexArgs)
		return NameRegex{
			Re:      regexp.MustCompile(checks.AnchoredPattern(x.Pattern)),
			Pattern: x.Pattern,
			Target:  x.Target,
		}
	}, nil)
}
