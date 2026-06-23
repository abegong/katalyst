package structuredobject

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// ObjectRequiredField checks that a frontmatter field exists.
type ObjectRequiredField struct {
	Field string
}

func (o ObjectRequiredField) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + o.Field
	if _, ok := ctx.Meta[o.Field]; ok {
		return nil
	}
	return []checks.Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("missing required field %q", o.Field),
		Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
	}}
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckObjectRequiredField,
		Family:    "structuredObject",
		Slug:      "required-field",
		Title:     "Required Field",
		Summary:   "Require that a frontmatter field exists.",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key that must be present."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_required_field
        field: year`,
	}, func(ch config.CheckInstance) checks.Check {
		return ObjectRequiredField{Field: ch.Field}
	}, nil)
}
