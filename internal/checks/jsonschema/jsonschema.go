// Package jsonschema is the JSON Schema check library: the first schema-backed
// CheckLibrary. It wraps github.com/santhosh-tekuri/jsonschema/v6 behind the
// checks.Schema interface and provides the `object` check type.
//
// The wrapper exists so the rest of the codebase doesn't depend on the
// underlying library directly. That keeps three things flexible:
//
//  1. We can swap implementations without touching command code.
//  2. We can normalize input from YAML (which produces native Go ints) into
//     the JSON-compatible shape the validator expects.
//  3. We can flatten the library's nested ValidationError tree into a simple
//     list that maps cleanly onto checks.Violation.
//
// `--schema` and the inline `schema:` frontmatter directive are this library's
// sugar for selecting an object schema; see Resolve and cmd/engine.
package jsonschema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
	schemalib "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// Library implements checks.SchemaLibrary for JSON Schema. It is stateless and
// always available: the engine runs in-process.
type Library struct{}

var (
	_ checks.SchemaLibrary = Library{}
	_ checks.Schema        = (*schema)(nil)
	_ checks.Check         = Object{}
)

// Name is the stable library id, used as Descriptor.Library on the object check.
func (Library) Name() string { return "json-schema" }

// Available reports the library can always run.
func (Library) Available() error { return nil }

// CompileSchema compiles one named JSON Schema from src.
func (Library) CompileSchema(name string, src []byte) (checks.Schema, error) {
	return Compile(name, src)
}

// Compile compiles a JSON Schema authored as JSON or YAML. The bytes are
// decoded as YAML (a superset of JSON) and re-encoded as JSON, then compiled,
// so a .json schema and a .yaml schema take the exact same path and there is
// no extension to sniff. The name is the schema's identifier (for $ref
// resolution and error reporting) and the compiler's resource key.
func Compile(name string, src []byte) (checks.Schema, error) {
	var raw any
	if err := yaml.Unmarshal(src, &raw); err != nil {
		return nil, fmt.Errorf("parse schema %q: %w", name, err)
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("convert schema %q to JSON: %w", name, err)
	}
	doc, err := schemalib.UnmarshalJSON(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("parse schema %q: %w", name, err)
	}
	c := schemalib.NewCompiler()
	if err := c.AddResource(name, doc); err != nil {
		return nil, fmt.Errorf("register schema %q: %w", name, err)
	}
	compiled, err := c.Compile(name)
	if err != nil {
		return nil, fmt.Errorf("compile schema %q: %w", name, err)
	}
	return &schema{name: name, compiled: compiled}, nil
}

// schema is a compiled JSON Schema implementing checks.Schema.
type schema struct {
	name     string
	compiled *schemalib.Schema
}

// Check validates the item's metadata against the schema and maps each failure
// to a checks.Violation, resolving the source line via the document's line map.
func (s *schema) Check(ctx checks.Context) []checks.Violation {
	errs := s.validate(ctx.Meta)
	if len(errs) == 0 {
		return nil
	}
	var lines map[string]int
	if ctx.Doc != nil {
		lines = ctx.Doc.Lines
	}
	out := make([]checks.Violation, 0, len(errs))
	for _, e := range errs {
		out = append(out, checks.Violation{
			Path:    e.path,
			Message: e.message,
			Line:    checks.LookupLine(lines, e.path),
		})
	}
	return out
}

// schemaError is one validation failure, flattened from the library's nested
// error tree. path is an RFC 6901 JSON pointer into the instance ("/year",
// "/tags/0", or "" for the root).
type schemaError struct {
	path    string
	message string
}

// validate runs the compiled schema over an instance, normalizing YAML-native
// values first, and returns a flat list of failures (empty when valid). The
// instance may use either JSON-native types (float64) or YAML-native types
// (int, int64).
func (s *schema) validate(instance any) []schemaError {
	err := s.compiled.Validate(normalize(instance))
	if err == nil {
		return nil
	}
	var ve *schemalib.ValidationError
	if !errors.As(err, &ve) {
		return []schemaError{{message: err.Error()}}
	}
	return flatten(ve)
}

// flatten walks the library's basic-output tree and returns a flat list of
// leaf-level errors. Each leaf is a concrete violation (wrong type, missing
// required field) rather than a structural node ("oneOf failed").
func flatten(ve *schemalib.ValidationError) []schemaError {
	basic := ve.BasicOutput()
	var out []schemaError
	visit(basic, &out)
	if len(out) == 0 {
		out = append(out, schemaError{path: basic.InstanceLocation, message: ve.Error()})
	}
	return out
}

func visit(u *schemalib.OutputUnit, out *[]schemaError) {
	if u == nil {
		return
	}
	if u.Error != nil {
		*out = append(*out, schemaError{path: u.InstanceLocation, message: u.Error.String()})
	}
	for i := range u.Errors {
		visit(&u.Errors[i], out)
	}
}

// normalize converts YAML-native Go values into the JSON-shaped values the
// validator expects:
//
//   - map[string]any stays as-is, recursively normalized
//   - map[any]any (a yaml.v3 quirk for non-string keys) is converted, with
//     non-string keys stringified
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
