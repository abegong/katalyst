// Package checktest holds tiny shared helpers for the check-type family test
// suites (parsing a document, loading a schema, taking a float pointer), so
// each family package can test its checks without redefining them.
package checktest

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/frontmatter"
	"github.com/abegong/katalyst/internal/validator"
)

// Ptr returns a pointer to v, for the *float64 bounds checks take.
func Ptr(v float64) *float64 { return &v }

// MustParseDoc parses src into a frontmatter document or fails the test.
func MustParseDoc(t *testing.T, src string) *frontmatter.Document {
	t.Helper()
	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

// MustLoadSchema loads a JSON Schema from src or fails the test.
func MustLoadSchema(t *testing.T, src string) *validator.Schema {
	t.Helper()
	s, err := validator.Load("test-schema", strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load schema: %v", err)
	}
	return s
}
