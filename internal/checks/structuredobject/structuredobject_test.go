package structuredobject_test

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/checktest"
	"github.com/abegong/katalyst/internal/checks/structuredobject"
)

func TestObjectRequiredFieldRun_detectsMissing(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\n---\n# Dune\n")
	violations := structuredobject.ObjectRequiredField{Field: "year"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "missing required") {
		t.Fatalf("expected required-field violation, got %v", violations)
	}
}

func TestObjectFieldTypeRun_detectsWrongType(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nyear: nope\n---\n# Dune\n")
	violations := structuredobject.ObjectFieldType{Field: "year", Type: "integer"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"year": "nope"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "must be type") {
		t.Fatalf("expected type violation, got %v", violations)
	}
}

func TestObjectFieldEnumRun_detectsOutOfSet(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nstatus: draft\n---\n# Dune\n")
	violations := structuredobject.ObjectFieldEnum{Field: "status", Values: []string{"published", "archived"}}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"status": "draft"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "allowed set") {
		t.Fatalf("expected enum violation, got %v", violations)
	}
}

func TestObjectNumberRangeRun_detectsOutOfRange(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nrating: 2\n---\n# Dune\n")
	violations := structuredobject.ObjectNumberRange{Field: "rating", Min: checktest.Ptr(3), Max: checktest.Ptr(5)}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"rating": 2},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, ">=") {
		t.Fatalf("expected number-range violation, got %v", violations)
	}
}

func TestObjectStringLengthRun_detectsTooShort(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: D\n---\n# D\n")
	violations := structuredobject.ObjectStringLength{Field: "title", MinLength: 3}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "D"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "length") {
		t.Fatalf("expected string-length violation, got %v", violations)
	}
}

func TestUniqueField_flagsDuplicateValues(t *testing.T) {
	ctx := checks.CollectionContext{Items: []checks.ItemContext{
		{FilePath: "notes/x.md", Meta: map[string]any{"slug": "dune"}},
		{FilePath: "notes/y.md", Meta: map[string]any{"slug": "dune"}},
		{FilePath: "notes/z.md", Meta: map[string]any{"slug": "other"}},
	}}
	violations := structuredobject.UniqueField{Field: "slug"}.RunCollection(ctx)
	if len(violations) != 1 || !strings.Contains(violations[0].Message, `"dune"`) {
		t.Fatalf("expected one duplicate-slug violation, got %v", violations)
	}
}

func TestObjectSentenceCaseRun_flagsTitleCase(t *testing.T) {
	allow := map[string]bool{"Katalyst": true}
	cases := []struct {
		title string
		want  bool // true if a violation is expected
	}{
		{"Progressive Operations", true},
		{"Progressive operations", false},
		{"Getting Started", true},
		{"Code of Conduct", true},
		{"Why Katalyst?", false},    // allowlisted proper noun
		{"Validate in CI", false},   // acronym
		{"Requires H1", false},      // letter+digit acronym
		{"File tree (deep)", false}, // parenthesized lowercase
		{"lowercase opener", true},  // first word must be capitalized
		{"Add a schema", false},
	}
	for _, tc := range cases {
		doc := checktest.MustParseDoc(t, "---\ntitle: x\n---\n# x\n")
		got := structuredobject.ObjectSentenceCase{Field: "title", Allow: allow}.Run(checks.Context{
			FilePath: "docs/x.md",
			Doc:      doc,
			Meta:     map[string]any{"title": tc.title},
		})
		if (len(got) > 0) != tc.want {
			t.Errorf("title %q: expected violation=%v, got %v", tc.title, tc.want, got)
		}
	}
}

func TestObjectSentenceCaseRun_skipsMissingOrNonString(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: x\n---\n# x\n")
	check := structuredobject.ObjectSentenceCase{Field: "title"}
	if got := check.Run(checks.Context{Doc: doc, Meta: map[string]any{}}); got != nil {
		t.Errorf("missing field should not violate, got %v", got)
	}
	if got := check.Run(checks.Context{Doc: doc, Meta: map[string]any{"title": 42}}); got != nil {
		t.Errorf("non-string field should not violate, got %v", got)
	}
}
