package checks

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/katabase-ai/katalyst/internal/validator"
)

// Object validates frontmatter metadata against JSON Schema.
type Object struct {
	Schema *validator.Schema
}

func (o Object) Run(ctx Context) []Violation {
	result := o.Schema.Validate(ctx.Meta)
	if result.Valid {
		return nil
	}
	out := make([]Violation, 0, len(result.Errors))
	for _, err := range result.Errors {
		out = append(out, Violation{
			Path:    err.Path,
			Message: err.Message,
			Line:    lookupLine(ctx.Doc.Lines, err.Path),
		})
	}
	return out
}

// ObjectRequiredField checks that a frontmatter field exists.
type ObjectRequiredField struct {
	Field string
}

func (o ObjectRequiredField) Run(ctx Context) []Violation {
	ptr := "/" + o.Field
	if _, ok := ctx.Meta[o.Field]; ok {
		return nil
	}
	return []Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("missing required field %q", o.Field),
		Line:    lookupLine(ctx.Doc.Lines, ptr),
	}}
}

// ObjectFieldType checks that a field has a specific type.
type ObjectFieldType struct {
	Field string
	Type  string
}

func (o ObjectFieldType) Run(ctx Context) []Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing field %q", o.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	expected := strings.ToLower(strings.TrimSpace(o.Type))
	if typeMatches(v, expected) {
		return nil
	}
	return []Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("field %q must be type %q", o.Field, expected),
		Line:    lookupLine(ctx.Doc.Lines, ptr),
	}}
}

// ObjectFieldEnum checks that a string field is in the allowed set.
type ObjectFieldEnum struct {
	Field  string
	Values []string
}

func (o ObjectFieldEnum) Run(ctx Context) []Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing field %q", o.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	s, ok := v.(string)
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be a string for enum check", o.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	for _, allowed := range o.Values {
		if s == allowed {
			return nil
		}
	}
	return []Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("field %q value %q is not in allowed set", o.Field, s),
		Line:    lookupLine(ctx.Doc.Lines, ptr),
	}}
}

// ObjectNumberRange checks numeric bounds for a field.
type ObjectNumberRange struct {
	Field string
	Min   *float64
	Max   *float64
}

func (o ObjectNumberRange) Run(ctx Context) []Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing field %q", o.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	num, ok := toFloat(v)
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be numeric", o.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if o.Min != nil && num < *o.Min {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be >= %v", o.Field, *o.Min),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if o.Max != nil && num > *o.Max {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be <= %v", o.Field, *o.Max),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	return nil
}

// ObjectStringLength checks minimum and/or maximum string length.
type ObjectStringLength struct {
	Field     string
	MinLength int
	MaxLength int
}

func (o ObjectStringLength) Run(ctx Context) []Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing field %q", o.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	s, ok := v.(string)
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be a string", o.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	l := utf8.RuneCountInString(s)
	if o.MinLength > 0 && l < o.MinLength {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q length must be >= %d", o.Field, o.MinLength),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	if o.MaxLength > 0 && l > o.MaxLength {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q length must be <= %d", o.Field, o.MaxLength),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	return nil
}

func typeMatches(v any, expected string) bool {
	switch expected {
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean", "bool":
		_, ok := v.(bool)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "number":
		_, ok := toFloat(v)
		return ok
	case "integer", "int":
		return isInteger(v)
	default:
		return false
	}
}

func isInteger(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	default:
		return 0, false
	}
}
