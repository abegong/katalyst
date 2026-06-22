package checks_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/frontmatter"
	"github.com/abegong/katalyst/internal/validator"
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

func TestNameMatchesFieldRun_detectsMismatch(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: dune-messiah\n---\n# Dune Messiah\n")
	meta := map[string]any{"slug": "dune-messiah"}

	violations := checks.NameMatchesField{Field: "slug"}.Run(checks.Context{
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

func TestNameMatchesFieldRun_slugifyMatchesTitle(t *testing.T) {
	doc := mustParseDoc(t, "---\ntitle: My First Note\n---\n# My First Note\n")
	meta := map[string]any{"title": "My First Note"}

	violations := checks.NameMatchesField{Field: "title", Transform: "slugify"}.Run(checks.Context{
		FilePath: "notes/my-first-note.md",
		Doc:      doc,
		Meta:     meta,
	})
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestNameCaseRun_styles(t *testing.T) {
	cases := []struct {
		style string
		name  string
		ok    bool
	}{
		{"kebab", "dune-messiah", true},
		{"kebab", "Dune Messiah", false},
		{"snake", "dune_messiah", true},
		{"snake", "dune-messiah", false},
		{"screaming-snake", "DUNE_MESSIAH", true},
		{"screaming-snake", "dune_messiah", false},
		{"camel", "duneMessiah", true},
		{"camel", "DuneMessiah", false},
		{"pascal", "DuneMessiah", true},
		{"pascal", "duneMessiah", false},
		{"point", "dune.messiah", true},
		{"point", "dune_messiah", false},
		{"lower", "dune.messiah-1", true},
		{"lower", "Dune", false},
	}
	for _, tc := range cases {
		doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
		violations := checks.NameCase{Style: tc.style}.Run(checks.Context{
			FilePath: "notes/" + tc.name + ".md",
			Doc:      doc,
			Meta:     map[string]any{"slug": "x"},
		})
		if tc.ok && len(violations) != 0 {
			t.Errorf("style %q name %q: expected pass, got %v", tc.style, tc.name, violations)
		}
		if !tc.ok && len(violations) == 0 {
			t.Errorf("style %q name %q: expected violation", tc.style, tc.name)
		}
	}
}

func TestNameCaseRun_parentDirTarget(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := checks.NameCase{Style: "kebab", Target: checks.TargetParentDir}.Run(checks.Context{
		FilePath: "notes/My Folder/ok-name.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "parent directory") {
		t.Fatalf("expected parent-dir kebab violation, got %v", violations)
	}
}

func TestNameCaseRun_pathSegmentsInclusive(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	// A bad mid-path segment AND a bad basename should both flag.
	violations := checks.NameCase{Style: "kebab", Target: checks.TargetPathSegments}.Run(checks.Context{
		FilePath:       "/proj/notes/Bad Dir/Bad File.md",
		CollectionRoot: "/proj/notes",
		Doc:            doc,
		Meta:           map[string]any{"slug": "x"},
	})
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations (dir + basename), got %d: %v", len(violations), violations)
	}
}

func TestNameAffixRun_prefixAndSuffix(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := checks.NameAffix{Prefix: "book-", Suffix: "-draft"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "x"},
	})
	if len(violations) != 2 {
		t.Fatalf("expected 2 affix violations, got %d: %v", len(violations), violations)
	}
}

func TestPathCharsetRun_denyReproducesNoSpaces(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := checks.PathCharset{Deny: []string{" "}}.Run(checks.Context{
		FilePath: "notes/my dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "must not contain") {
		t.Fatalf("expected deny-space violation, got %v", violations)
	}
}

func TestPathCharsetRun_allowWhitelist(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := checks.PathCharset{Allow: []string{"abcdefghijklmnopqrstuvwxyz-."}}.Run(checks.Context{
		FilePath:       "/proj/notes/My_File.md",
		CollectionRoot: "/proj/notes",
		Doc:            doc,
		Meta:           map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "disallowed character") {
		t.Fatalf("expected allow-whitelist violation, got %v", violations)
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

func TestNameRegexRun_anchored(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	re := regexp.MustCompile(checks.AnchoredPattern(`[0-9]{4}-[a-z-]+`))
	// "2024-dune" matches fully; "x-2024-dune" must NOT (anchored).
	pass := checks.NameRegex{Re: re, Pattern: "p"}.Run(checks.Context{
		FilePath: "notes/2024-dune.md", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(pass) != 0 {
		t.Fatalf("expected pass for anchored match, got %v", pass)
	}
	fail := checks.NameRegex{Re: re, Pattern: "p"}.Run(checks.Context{
		FilePath: "notes/x-2024-dune.md", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(fail) != 1 {
		t.Fatalf("expected anchored mismatch to fail, got %v", fail)
	}
}

func TestNameLengthRun_bounds(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	max := 5
	violations := checks.NameLength{Max: &max}.Run(checks.Context{
		FilePath: "notes/toolongname.md", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "at most") {
		t.Fatalf("expected max-length violation, got %v", violations)
	}
}

func TestPathDepthRun_flat(t *testing.T) {
	doc := mustParseDoc(t, "---\nslug: x\n---\n# x\n")
	max := 0
	// A nested file violates max depth 0; a root file does not.
	nested := checks.PathDepth{Max: &max}.Run(checks.Context{
		FilePath: "/proj/notes/sub/dune.md", CollectionRoot: "/proj/notes", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(nested) != 1 || !strings.Contains(nested[0].Message, "depth") {
		t.Fatalf("expected depth violation for nested file, got %v", nested)
	}
	flat := checks.PathDepth{Max: &max}.Run(checks.Context{
		FilePath: "/proj/notes/dune.md", CollectionRoot: "/proj/notes", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(flat) != 0 {
		t.Fatalf("expected no violation for flat file, got %v", flat)
	}
}

func TestParentDirMatchesFieldRun_detectsMismatch(t *testing.T) {
	doc := mustParseDoc(t, "---\ncategory: recipes\n---\n# x\n")
	violations := checks.ParentDirMatchesField{Field: "category"}.Run(checks.Context{
		FilePath: "notes/books/dune.md", Doc: doc, Meta: map[string]any{"category": "recipes"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "must match field") {
		t.Fatalf("expected parent-dir-matches-field violation, got %v", violations)
	}
}

func TestReferencedFilesExistRun_missingAndList(t *testing.T) {
	dir := t.TempDir()
	itemPath := filepath.Join(dir, "dune.md")
	if err := os.WriteFile(filepath.Join(dir, "cover.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := mustParseDoc(t, "---\ncover: cover.png\nextras:\n  - a.png\n  - b.png\n---\n# x\n")
	violations := checks.ReferencedFilesExist{Fields: []string{"cover", "extras"}}.Run(checks.Context{
		FilePath: itemPath,
		Doc:      doc,
		Meta:     map[string]any{"cover": "cover.png", "extras": []any{"a.png", "b.png"}},
	})
	// cover.png exists; a.png and b.png do not → 2 violations.
	if len(violations) != 2 {
		t.Fatalf("expected 2 missing-file violations, got %d: %v", len(violations), violations)
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
