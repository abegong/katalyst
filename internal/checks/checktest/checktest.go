// Package checktest holds tiny shared helpers for the check-type family test
// suites, so each family package can test its checks without redefining common
// document and context setup.
package checktest

import (
	"testing"

	"github.com/abegong/katalyst/internal/checks"
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

// Context returns a checks context with the given frontmatter metadata.
func Context(meta map[string]any) checks.Context {
	return checks.Context{Meta: meta}
}

// ContextWithDoc returns a checks context with a parsed markdown document and
// the given frontmatter metadata.
func ContextWithDoc(t *testing.T, path, src string, meta map[string]any) checks.Context {
	t.Helper()
	return checks.Context{
		FilePath: path,
		Doc:      MustParseDoc(t, src),
		Meta:     meta,
	}
}
