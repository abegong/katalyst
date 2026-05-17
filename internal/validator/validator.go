// Package validator wraps github.com/santhosh-tekuri/jsonschema/v6 with a
// small, stable API tailored to katabridge's needs.
//
// The wrapper exists so the rest of the codebase doesn't depend directly on
// the underlying library. That keeps three things flexible:
//
//  1. We can swap implementations without touching command code.
//  2. We can normalize input from YAML (which produces native Go ints)
//     into the JSON-compatible shape the validator expects.
//  3. We can flatten the library's nested ValidationError tree into a
//     simple list that's easy to print and assert against in tests.
package validator

import (
	"errors"
	"fmt"
	"io"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Schema is a compiled JSON Schema.
type Schema struct {
	name     string
	compiled *jsonschema.Schema
}

// Name returns the identifier the schema was loaded under (typically a
// file path or URL). It's stable and safe to use in error messages.
func (s *Schema) Name() string { return s.name }

// Error is a single validation failure, flattened from the library's
// nested error tree.
//
// Path is an RFC 6901 JSON pointer into the instance ("/year",
// "/tags/0", or "" for the root).
type Error struct {
	Path    string
	Message string
}

// Result is the outcome of validating a single instance.
type Result struct {
	Valid  bool
	Errors []Error
}

// Load compiles a JSON Schema from r. The name is used both as the
// schema's identifier (for $ref resolution and error reporting) and as
// the key in the compiler's resource map.
func Load(name string, r io.Reader) (*Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(r)
	if err != nil {
		return nil, fmt.Errorf("parse schema %q: %w", name, err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource(name, doc); err != nil {
		return nil, fmt.Errorf("register schema %q: %w", name, err)
	}

	compiled, err := c.Compile(name)
	if err != nil {
		return nil, fmt.Errorf("compile schema %q: %w", name, err)
	}

	return &Schema{name: name, compiled: compiled}, nil
}

// Validate checks instance against the schema. The instance may use
// either JSON-native types (float64 for numbers, etc.) or YAML-native
// types (int, int64); Validate normalizes them before validation.
func (s *Schema) Validate(instance any) Result {
	normalized := normalize(instance)

	err := s.compiled.Validate(normalized)
	if err == nil {
		return Result{Valid: true}
	}

	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return Result{
			Valid:  false,
			Errors: []Error{{Message: err.Error()}},
		}
	}

	return Result{
		Valid:  false,
		Errors: flatten(ve),
	}
}

// flatten walks the library's basic-output tree and returns a flat list
// of leaf-level errors. Each leaf corresponds to a concrete violation
// (wrong type, missing required field, etc.) rather than a structural
// node ("oneOf failed").
func flatten(ve *jsonschema.ValidationError) []Error {
	basic := ve.BasicOutput()
	var out []Error
	visit(basic, &out)
	if len(out) == 0 {
		out = append(out, Error{
			Path:    basic.InstanceLocation,
			Message: ve.Error(),
		})
	}
	return out
}

func visit(u *jsonschema.OutputUnit, out *[]Error) {
	if u == nil {
		return
	}
	if u.Error != nil {
		*out = append(*out, Error{
			Path:    u.InstanceLocation,
			Message: u.Error.String(),
		})
	}
	for i := range u.Errors {
		visit(&u.Errors[i], out)
	}
}

// normalize converts YAML-native Go values into the JSON-shaped values
// the validator expects:
//
//   - map[string]any stays as-is, recursively normalized
//   - map[any]any (a yaml.v3 quirk for non-string keys) is converted,
//     with non-string keys stringified
//   - []any is recursively normalized
//   - integer types are converted to float64
//
// All other values pass through unchanged.
func normalize(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = normalize(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[fmt.Sprint(k)] = normalize(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = normalize(item)
		}
		return out
	case int:
		return float64(x)
	case int8:
		return float64(x)
	case int16:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint8:
		return float64(x)
	case uint16:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	default:
		return v
	}
}
