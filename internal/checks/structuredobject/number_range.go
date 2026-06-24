package structuredobject

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// ObjectNumberRange checks numeric bounds for a field.
type ObjectNumberRange struct {
	Field string
	Min   *float64
	Max   *float64
}

// numberRangeArgs is object_number_range's own config shape.
type numberRangeArgs struct {
	Field string   `yaml:"field"`
	Min   *float64 `yaml:"min"`
	Max   *float64 `yaml:"max"`
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
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckObjectNumberRange,
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
	}, checks.ParseInto(func(a numberRangeArgs) error {
		if err := argcheck.RequireString("object_number_range", "field", a.Field); err != nil {
			return err
		}
		return argcheck.RequireOneOfFields("object_number_range", a.Min != nil || a.Max != nil, "min", "max")
	}), func(a any) checks.Check {
		r := a.(numberRangeArgs)
		return ObjectNumberRange{Field: r.Field, Min: r.Min, Max: r.Max}
	}, nil)
}
