package jsonschema_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/jsonschema"
	"github.com/abegong/katalyst/internal/storage/collection/document"
)

//go:embed testdata/schemas/book.json
var bookSchema string

func mustCompile(t *testing.T, src string) checks.Schema {
	t.Helper()
	s, err := jsonschema.Compile("book.json", []byte(src))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return s
}

func check(t *testing.T, s checks.Schema, meta map[string]any) []checks.Violation {
	t.Helper()
	return s.Check(checks.Context{Meta: meta})
}

func TestLibrary_identity(t *testing.T) {
	lib := jsonschema.Library{}
	if lib.Name() != "json-schema" {
		t.Errorf("Name = %q, want json-schema", lib.Name())
	}
	if err := lib.Available(); err != nil {
		t.Errorf("Available = %v, want nil", err)
	}
}

func TestCompile_invalidSchema(t *testing.T) {
	if _, err := jsonschema.Compile("bad.json", []byte(`{ "type": 123 }`)); err == nil {
		t.Fatalf("expected error compiling invalid schema")
	}
}

// A YAML-authored schema compiles through the same Compile path as JSON and
// validates the same way.
func TestCompile_yamlCompilesAndValidates(t *testing.T) {
	const bookSchemaYAML = `type: object
additionalProperties: false
required:
  - title
  - year
properties:
  title:
    type: string
    minLength: 1
  year:
    type: integer
  tags:
    type: array
    items:
      type: string
`
	s, err := jsonschema.Compile("book.yaml", []byte(bookSchemaYAML))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if v := check(t, s, map[string]any{"title": "Dune", "year": 1965}); len(v) != 0 {
		t.Fatalf("expected valid, got %+v", v)
	}
	v := check(t, s, map[string]any{"title": "Dune"})
	if len(v) == 0 {
		t.Fatalf("expected invalid (missing year)")
	}
	if !mentions(v, "year") {
		t.Errorf("expected a violation mentioning 'year', got: %+v", v)
	}
}

func TestCheck_valid(t *testing.T) {
	s := mustCompile(t, bookSchema)
	v := check(t, s, map[string]any{"title": "Dune", "year": 1965, "tags": []any{"sci-fi"}})
	if len(v) != 0 {
		t.Errorf("expected no violations, got %+v", v)
	}
}

func TestCheck_missingRequired(t *testing.T) {
	s := mustCompile(t, bookSchema)
	v := check(t, s, map[string]any{"title": "Dune"})
	if !mentions(v, "year") {
		t.Errorf("expected a violation mentioning 'year', got: %+v", v)
	}
}

func TestCheck_wrongType(t *testing.T) {
	s := mustCompile(t, bookSchema)
	v := check(t, s, map[string]any{"title": "Dune", "year": "not a number"})
	if !hasPath(v, "/year") {
		t.Errorf("expected a violation at /year, got: %+v", v)
	}
}

func TestCheck_additionalProperty(t *testing.T) {
	s := mustCompile(t, bookSchema)
	v := check(t, s, map[string]any{"title": "Dune", "year": 1965, "unknown": "field"})
	if len(v) == 0 {
		t.Fatalf("expected invalid due to additionalProperties: false")
	}
}

// Check resolves a violation's source line through the document's line map,
// the behavior the object check type relies on.
func TestCheck_reportsLineForSchemaViolation(t *testing.T) {
	doc, err := document.Parse([]byte("---\ntitle: Dune\nyear: nope\n---\n# Dune\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := mustCompile(t, `{"type":"object","properties":{"year":{"type":"integer"}},"required":["year"]}`)
	v := s.Check(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune", "year": "nope"},
	})
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %+v", len(v), v)
	}
	if v[0].Path != "/year" {
		t.Fatalf("expected /year path, got %q", v[0].Path)
	}
	if v[0].Line != 3 {
		t.Fatalf("expected line 3, got %d", v[0].Line)
	}
}

// TestObject_delegatesToSchema covers the object check type wrapper.
func TestObject_delegatesToSchema(t *testing.T) {
	s := mustCompile(t, `{"type":"object","required":["year"]}`)
	v := jsonschema.Object{Schema: s}.Run(checks.Context{Meta: map[string]any{"title": "Dune"}})
	if !mentions(v, "year") {
		t.Errorf("expected object check to report missing year, got: %+v", v)
	}
}

func mentions(vs []checks.Violation, needle string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, needle) || strings.Contains(v.Path, needle) {
			return true
		}
	}
	return false
}

func hasPath(vs []checks.Violation, path string) bool {
	for _, v := range vs {
		if v.Path == path {
			return true
		}
	}
	return false
}
