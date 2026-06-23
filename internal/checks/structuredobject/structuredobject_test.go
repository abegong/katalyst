package structuredobject_test

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/checktest"
	"github.com/abegong/katalyst/internal/checks/structuredobject"
)

func TestObjectRun_reportsLineForSchemaViolation(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: Dune\nyear: nope\n---\n# Dune\n")
	schema := checktest.MustLoadSchema(t, `{"type":"object","properties":{"year":{"type":"integer"}},"required":["year"]}`)
	meta := map[string]any{
		"title": "Dune",
		"year":  "nope",
	}

	violations := structuredobject.Object{Schema: schema}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Path != "/year" {
		t.Fatalf("expected /year path, got %q", violations[0].Path)
	}
	if violations[0].Line != 3 {
		t.Fatalf("expected line 3, got %d", violations[0].Line)
	}
}

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
