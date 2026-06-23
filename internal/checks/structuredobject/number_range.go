package structuredobject

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// ObjectNumberRange checks numeric bounds for a field.
type ObjectNumberRange struct {
	Field string
	Min   *float64
	Max   *float64
}

func (o ObjectNumberRange) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing field %q", o.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	num, ok := toFloat(v)
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be numeric", o.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if o.Min != nil && num < *o.Min {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be >= %v", o.Field, *o.Min),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if o.Max != nil && num > *o.Max {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be <= %v", o.Field, *o.Max),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	return nil
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckObjectNumberRange,
		Family:    "structuredObject",
		Slug:      "number-range",
		Title:     "Number range",
		Summary:   "Constrain a numeric field to a minimum and/or maximum value.",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key to check."},
			{Name: "min", Required: false, Desc: "Inclusive lower bound. At least one of `min`/`max` is required."},
			{Name: "max", Required: false, Desc: "Inclusive upper bound. At least one of `min`/`max` is required."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_number_range
        field: year
        min: 1900
        max: 2100`,
	}, func(ch config.CheckInstance) checks.Check {
		return ObjectNumberRange{Field: ch.Field, Min: ch.Min, Max: ch.Max}
	}, nil)
}
