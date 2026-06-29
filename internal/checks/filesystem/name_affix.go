package filesystem

import (
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// NameAffix checks that the target starts with Prefix and/or ends with Suffix.
type NameAffix struct {
	Prefix string
	Suffix string
	Target string
}

func (c NameAffix) Run(ctx checks.Context) []checks.Violation {
	v := resolveTarget(ctx, c.Target)[0]
	noun := targetNoun(c.Target)
	var out []checks.Violation
	if c.Prefix != "" && !strings.HasPrefix(v, c.Prefix) {
		out = append(out, checks.Violation{
			Path:    "/",
			Message: fmt.Sprintf("%s %q must start with prefix %q", noun, v, c.Prefix),
		})
	}
	if c.Suffix != "" && !strings.HasSuffix(v, c.Suffix) {
		out = append(out, checks.Violation{
			Path:    "/",
			Message: fmt.Sprintf("%s %q must end with suffix %q", noun, v, c.Suffix),
		})
	}
	return out
}

type nameAffixArgs struct {
	Prefix string `yaml:"prefix"`
	Suffix string `yaml:"suffix"`
	Target string `yaml:"target"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckFilesystemNameAffix,
		Family:    "fileSystem",
		Targets:   []string{checks.TargetCollection, checks.TargetFilesystem},
		Slug:      "name-affix",
		Title:     "Name affix",
		Summary:   "Require a name to start with a prefix and/or end with a suffix.",
		Fields: []checks.Field{
			{Name: "prefix", Required: false, Desc: "Required name prefix (at least one of prefix/suffix)."},
			{Name: "suffix", Required: false, Desc: "Required name suffix (at least one of prefix/suffix)."},
			{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, or `parent-dir`."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_affix
        prefix: book-`,
	}, checks.ParseInto(func(a nameAffixArgs) error {
		if err := argcheck.RequireOneOfFields("filesystem_name_affix", a.Prefix != "" || a.Suffix != "", "prefix", "suffix"); err != nil {
			return err
		}
		return validateTarget("filesystem_name_affix", a.Target)
	}), func(a any) checks.Check {
		x := a.(nameAffixArgs)
		return NameAffix{Prefix: x.Prefix, Suffix: x.Suffix, Target: x.Target}
	}, nil)
}
