package validator_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/katabase-ai/katalyst/internal/validator"
)

//go:embed testdata/schemas/book.json
var bookSchema string

func mustLoad(t *testing.T, src string) *validator.Schema {
	t.Helper()
	s, err := validator.Load("book.json", strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return s
}

func TestLoad_invalidSchema(t *testing.T) {
	_, err := validator.Load("bad.json", strings.NewReader(`{ "type": 123 }`))
	if err == nil {
		t.Fatalf("expected error loading invalid schema")
	}
}

// A YAML-authored schema compiles through LoadYAML and validates the same
// way the equivalent JSON schema would.
func TestLoadYAML_compilesAndValidates(t *testing.T) {
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
	s, err := validator.LoadYAML("book.yaml", strings.NewReader(bookSchemaYAML))
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	if r := s.Validate(map[string]any{"title": "Dune", "year": 1965}); !r.Valid {
		t.Fatalf("expected valid, got errors: %+v", r.Errors)
	}
	r := s.Validate(map[string]any{"title": "Dune"})
	if r.Valid {
		t.Fatalf("expected invalid (missing year)")
	}
	if !hasErrorMentioning(r.Errors, "year") {
		t.Errorf("expected an error mentioning 'year', got: %+v", r.Errors)
	}
}

func TestValidate_valid(t *testing.T) {
	s := mustLoad(t, bookSchema)

	doc := map[string]any{
		"title": "Dune",
		"year":  1965,
		"tags":  []any{"sci-fi"},
	}

	result := s.Validate(doc)
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %+v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}
}

func TestValidate_missingRequired(t *testing.T) {
	s := mustLoad(t, bookSchema)

	doc := map[string]any{
		"title": "Dune",
	}

	result := s.Validate(doc)
	if result.Valid {
		t.Fatalf("expected invalid")
	}
	if !hasErrorMentioning(result.Errors, "year") {
		t.Errorf("expected an error mentioning 'year', got: %+v", result.Errors)
	}
}

func TestValidate_wrongType(t *testing.T) {
	s := mustLoad(t, bookSchema)

	doc := map[string]any{
		"title": "Dune",
		"year":  "not a number",
	}

	result := s.Validate(doc)
	if result.Valid {
		t.Fatalf("expected invalid")
	}
	if !hasErrorWithPath(result.Errors, "/year") {
		t.Errorf("expected an error at /year, got: %+v", result.Errors)
	}
}

func TestValidate_additionalProperty(t *testing.T) {
	s := mustLoad(t, bookSchema)

	doc := map[string]any{
		"title":   "Dune",
		"year":    1965,
		"unknown": "field",
	}

	result := s.Validate(doc)
	if result.Valid {
		t.Fatalf("expected invalid due to additionalProperties: false")
	}
}

func hasErrorMentioning(errs []validator.Error, needle string) bool {
	for _, e := range errs {
		if strings.Contains(e.Message, needle) || strings.Contains(e.Path, needle) {
			return true
		}
	}
	return false
}

func hasErrorWithPath(errs []validator.Error, path string) bool {
	for _, e := range errs {
		if e.Path == path {
			return true
		}
	}
	return false
}
