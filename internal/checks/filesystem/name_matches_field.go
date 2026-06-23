package filesystem

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// NameMatchesField checks that the target equals a frontmatter field,
// optionally after a transform.
type NameMatchesField struct {
	Field     string
	Transform string
	Target    string
}

func (c NameMatchesField) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + c.Field
	raw, ok := ctx.Meta[c.Field]
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing frontmatter field %q", c.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	want, ok := raw.(string)
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("frontmatter field %q must be a string", c.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if c.Transform == "slugify" {
		want = slugify(want)
	}
	got := resolveTarget(ctx, c.Target)[0]
	if got == want {
		return nil
	}
	return []checks.Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("%s %q must match field %q (%q)", targetNoun(c.Target), got, c.Field, want),
		Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
	}}
}

var nonSlugRun = regexp.MustCompile(`[^a-z0-9]+`)

// slugify lowercases and kebab-cases a string: runs of non-alphanumerics
// collapse to a single hyphen, and leading/trailing hyphens are trimmed.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonSlugRun.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckFilesystemNameMatchesField,
		Family:    "fileSystem",
		Slug:      "name-matches-field",
		Title:     "Name matches field",
		Summary:   "Require a name to equal a frontmatter field, optionally slugified.",
		Fields: []checks.Field{
			{Name: "field", Required: false, Default: "slug", Desc: "Frontmatter key compared to the name."},
			{Name: "transform", Required: false, Default: "none", Desc: "`none` or `slugify` (applied to the field value before comparison)."},
			{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, or `parent-dir`."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_matches_field
        field: slug`,
	}, func(ch config.CheckInstance) checks.Check {
		return NameMatchesField{Field: ch.Field, Transform: ch.Transform, Target: ch.Target}
	}, nil)
}
