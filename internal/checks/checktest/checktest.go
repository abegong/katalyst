// Package checktest holds tiny shared helpers for the check-type family test
// suites (parsing a document, taking a float pointer), so each family package
// can test its checks without redefining them.
package checktest

import (
	"testing"

	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
)

// Ptr returns a pointer to v, for the *float64 bounds checks take.
func Ptr(v float64) *float64 { return &v }

// MustParseDoc parses src into a frontmatter document or fails the test.
func MustParseDoc(t *testing.T, src string) *markdownbodytext.Document {
	t.Helper()
	doc, err := markdownbodytext.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}
