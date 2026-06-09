package checks_test

import (
	"strings"
	"testing"

	"github.com/katabase-ai/katalyst/internal/checks"
	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/katabase-ai/katalyst/internal/validator"
)

func ptr(v float64) *float64 { return &v }

func mustParseDoc(t *testing.T, src string) *frontmatter.Document {
	t.Helper()
	doc, err := frontmatter.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func mustLoadSchema(t *testing.T, src string) *validator.Schema {
	t.Helper()
	s, err := validator.Load("test-schema", strings.NewReader(src))
	if err != nil {
		t.Fatalf("Load schema: %v", err)
	}
	return s
}

func TestObjectRun_reportsLineForSchemaViolation(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\nyear: nope\n---\n# Dune\n")
	schema := mustLoadSchema(t, `{"type":"object","properties":{"year":{"type":"integer"}},"required":["year"]}`)
	meta := map[string]any{
		"title": "Dune",
		"year":  "nope",
	}

	violations := checks.Object{Schema: schema}.Run(checks.Context{
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

func TestMarkdownTitleMatchesH1Run_detectsMismatch(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# Children of Dune\n")
	meta := map[string]any{"title": "Dune"}

	violations := checks.MarkdownTitleMatchesH1{Field: "title"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Line != 4 {
		t.Fatalf("expected mismatch to report H1 line 4, got %d", violations[0].Line)
	}
}

func TestMarkdownTitleMatchesH1Run_acceptsMatch(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# Dune\n")
	meta := map[string]any{"title": "Dune"}

	violations := checks.MarkdownTitleMatchesH1{Field: "title"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestFilenameMatchesSlugRun_detectsMismatch(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune-messiah\n---\n# Dune Messiah\n")
	meta := map[string]any{"slug": "dune-messiah"}

	violations := checks.FilenameMatchesSlug{Field: "slug"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Line != 2 {
		t.Fatalf("expected line 2 for slug key, got %d", violations[0].Line)
	}
}

func TestObjectRequiredFieldRun_detectsMissing(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# Dune\n")
	violations := checks.ObjectRequiredField{Field: "year"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "missing required") {
		t.Fatalf("expected required-field violation, got %v", violations)
	}
}

func TestObjectFieldTypeRun_detectsWrongType(t *testing.T) {
	doc := mustParseDoc(t, "---\nyear: nope\n---\n# Dune\n")
	violations := checks.ObjectFieldType{Field: "year", Type: "integer"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"year": "nope"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "must be type") {
		t.Fatalf("expected type violation, got %v", violations)
	}
}

func TestObjectFieldEnumRun_detectsOutOfSet(t *testing.T) {
	doc := mustParseDoc(t, "---\nstatus: draft\n---\n# Dune\n")
	violations := checks.ObjectFieldEnum{Field: "status", Values: []string{"published", "archived"}}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"status": "draft"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "allowed set") {
		t.Fatalf("expected enum violation, got %v", violations)
	}
}

func TestObjectNumberRangeRun_detectsOutOfRange(t *testing.T) {
	doc := mustParseDoc(t, "---\nrating: 2\n---\n# Dune\n")
	violations := checks.ObjectNumberRange{Field: "rating", Min: ptr(3), Max: ptr(5)}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"rating": 2},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, ">=") {
		t.Fatalf("expected number-range violation, got %v", violations)
	}
}

func TestObjectStringLengthRun_detectsTooShort(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: D\n---\n# D\n")
	violations := checks.ObjectStringLength{Field: "title", MinLength: 3}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "D"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "length") {
		t.Fatalf("expected string-length violation, got %v", violations)
	}
}

func TestMarkdownRequiresH1Run_detectsMissing(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\nNo heading\n")
	violations := checks.MarkdownRequiresH1{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "missing H1") {
		t.Fatalf("expected requires-h1 violation, got %v", violations)
	}
}

func TestMarkdownSingleH1Run_detectsMultiple(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# One\n# Two\n")
	violations := checks.MarkdownSingleH1{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "only one H1") {
		t.Fatalf("expected single-h1 violation, got %v", violations)
	}
}

func TestMarkdownNoHeadingLevelJumpsRun_detectsJump(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# One\n### Jump\n")
	violations := checks.MarkdownNoHeadingLevelJumps{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "jump") {
		t.Fatalf("expected heading-jump violation, got %v", violations)
	}
}

func TestMarkdownRequiredSectionRun_detectsMissing(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n# Dune\n## Notes\n")
	violations := checks.MarkdownRequiredSection{Heading: "Summary"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "required section") {
		t.Fatalf("expected required-section violation, got %v", violations)
	}
}

func TestMarkdownCodeFenceHasLanguageRun_detectsMissing(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: Dune\n---\n```\ncode\n```\n")
	violations := checks.MarkdownCodeFenceHasLanguage{}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "Dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "language") {
		t.Fatalf("expected code-fence-language violation, got %v", violations)
	}
}

func TestFilesystemExtensionInRun_detectsDisallowed(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune\n---\n# Dune\n")
	violations := checks.FilesystemExtensionIn{Values: []string{".md"}}.Run(checks.Context{
		FilePath: "notes/dune.txt",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "extension") {
		t.Fatalf("expected extension violation, got %v", violations)
	}
}

func TestFilesystemFilenameKebabCaseRun_detectsInvalid(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune\n---\n# Dune\n")
	violations := checks.FilesystemFilenameKebabCase{}.Run(checks.Context{
		FilePath: "notes/Dune Story.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "kebab-case") {
		t.Fatalf("expected kebab-case violation, got %v", violations)
	}
}

func TestFilesystemNoSpacesInPathRun_detectsSpaces(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune\n---\n# Dune\n")
	violations := checks.FilesystemNoSpacesInPath{}.Run(checks.Context{
		FilePath: "notes/my dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "spaces") {
		t.Fatalf("expected no-spaces violation, got %v", violations)
	}
}

func TestFilesystemParentDirInRun_detectsDisallowedParent(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune\n---\n# Dune\n")
	violations := checks.FilesystemParentDirIn{Values: []string{"books"}}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "parent directory") {
		t.Fatalf("expected parent-dir violation, got %v", violations)
	}
}

func TestFilesystemFilenamePrefixRun_detectsMissingPrefix(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune\n---\n# Dune\n")
	violations := checks.FilesystemFilenamePrefix{Value: "book-"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "prefix") {
		t.Fatalf("expected prefix violation, got %v", violations)
	}
}
