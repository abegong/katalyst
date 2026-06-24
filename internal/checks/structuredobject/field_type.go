package structuredobject

import (
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// ObjectFieldType checks that a field has a specific type.
type ObjectFieldType struct {
	Field string
	Type  string
}

// fieldTypeArgs is object_field_type's own config shape.
type fieldTypeArgs struct {
	Field string `yaml:"field"`
	Type  string `yaml:"type"`
}

func (o ObjectFieldType) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing field %q", o.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	expected := strings.ToLower(strings.TrimSpace(o.Type))
	if typeMatches(v, expected) {
		return nil
	}
	return []checks.Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("field %q must be type %q", o.Field, expected),
		Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
	}}
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckObjectFieldType,
		Family:    "structuredObject",
		Slug:      "field-type",
		Title:     "Field type",
		Summary:   "Require that a frontmatter field has a specific type.",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key to check."},
			{Name: "type", Required: true, Desc: "One of `string`, `boolean`, `array`, `object`, `number`, `integer`."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_field_type
        field: year
        type: integer`,
	}, checks.ParseInto(func(a fieldTypeArgs) error {
		if err := argcheck.RequireString("object_field_type", "field", a.Field); err != nil {
			return err
		}
		return argcheck.RequireString("object_field_type", "type", a.Type)
	}), func(a any) checks.Check {
		t := a.(fieldTypeArgs)
		return ObjectFieldType{Field: t.Field, Type: t.Type}
	}, nil)
}
