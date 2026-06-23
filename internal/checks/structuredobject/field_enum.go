package structuredobject

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// ObjectFieldEnum checks that a string field is in the allowed set.
type ObjectFieldEnum struct {
	Field  string
	Values []string
}

func (o ObjectFieldEnum) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing field %q", o.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	s, ok := v.(string)
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be a string for enum check", o.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	for _, allowed := range o.Values {
		if s == allowed {
			return nil
		}
	}
	return []checks.Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("field %q value %q is not in allowed set", o.Field, s),
		Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
	}}
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckObjectFieldEnum,
		Family:    "structuredObject",
		Slug:      "field-enum",
		Title:     "Field Enum",
		Summary:   "Require that a field is one of a fixed set of values.",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key to check."},
			{Name: "values", Required: true, Desc: "Allowed values."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_field_enum
        field: status
        values: [draft, published, archived]`,
	}, func(ch config.CheckInstance) checks.Check {
		return ObjectFieldEnum{Field: ch.Field, Values: ch.Values}
	}, nil)
}
