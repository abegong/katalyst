package structuredobject

import (
	"fmt"
	"unicode/utf8"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// ObjectStringLength checks minimum and/or maximum string length.
type ObjectStringLength struct {
	Field     string
	MinLength int
	MaxLength int
}

// stringLengthArgs is object_string_length's own config shape.
type stringLengthArgs struct {
	Field     string `yaml:"field"`
	MinLength int    `yaml:"min_length"`
	MaxLength int    `yaml:"max_length"`
}

func (o ObjectStringLength) Run(ctx checks.Context) []checks.Violation {
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
			Message: fmt.Sprintf("field %q must be a string", o.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	l := utf8.RuneCountInString(s)
	if o.MinLength > 0 && l < o.MinLength {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q length must be >= %d", o.Field, o.MinLength),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if o.MaxLength > 0 && l > o.MaxLength {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q length must be <= %d", o.Field, o.MaxLength),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	return nil
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckObjectStringLength,
		Family:    "structuredObject",
		Slug:      "string-length",
		Title:     "String length",
		Summary:   "Constrain the minimum and/or maximum length of a string field.",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key to check."},
			{Name: "min_length", Required: false, Desc: "Minimum length. At least one of `min_length`/`max_length` is required."},
			{Name: "max_length", Required: false, Desc: "Maximum length. At least one of `min_length`/`max_length` is required."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_string_length
        field: title
        min_length: 3
        max_length: 120`,
	}, checks.ParseInto(func(a stringLengthArgs) error {
		if err := argcheck.RequireString("object_string_length", "field", a.Field); err != nil {
			return err
		}
		return argcheck.RequireOneOfFields("object_string_length", a.MinLength != 0 || a.MaxLength != 0, "min_length", "max_length")
	}), func(a any) checks.Check {
		s := a.(stringLengthArgs)
		return ObjectStringLength{Field: s.Field, MinLength: s.MinLength, MaxLength: s.MaxLength}
	}, nil)
}
