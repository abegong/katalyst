package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Set up an init-scaffolded repo and chdir into it.
func setupScaffoldRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, _, err := runRoot(t, "init", "--dir", dir); err != nil {
		t.Fatalf("init: %v", err)
	}
	chdir(t, dir)
	return dir
}

func TestValidate_usesConfigWhenSchemaFlagOmitted(t *testing.T) {
	dir := setupScaffoldRepo(t)

	_, stderr, err := runRoot(t, "validate", filepath.Join(dir, "notes/example.md"))
	if err != nil {
		t.Fatalf("validate via config failed: %v\nstderr: %s", err, stderr)
	}
}

func TestValidate_inlineSchemaKeyTakesPrecedence(t *testing.T) {
	dir := setupScaffoldRepo(t)

	// Add a second schema and a doc that asks for it inline. The config
	// rules (`notes/**` -> book) would otherwise apply.
	mustWrite(t, filepath.Join(dir, "schemas/strict-book.json"), strictBookSchemaFixture)
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), strictBookConfigFixture)

	docPath := filepath.Join(dir, "notes/strict.md")
	mustWrite(t, docPath, `---
schema: strict-book
title: Dune
year: 1965
---
# Body
`)

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected validation failure (missing isbn under strict-book)")
	}
	if !strings.Contains(stderr, "isbn") {
		t.Errorf("expected stderr to mention 'isbn', got: %q", stderr)
	}
}

func TestValidate_unmatchedFileIsError(t *testing.T) {
	dir := setupScaffoldRepo(t)

	// `elsewhere/` is outside the `notes/**` rule. No inline schema.
	outsider := filepath.Join(dir, "elsewhere/random.md")
	mustWrite(t, outsider, "---\ntitle: x\nyear: 1\n---\n# Body\n")

	_, stderr, err := runRoot(t, "validate", outsider)
	if err == nil {
		t.Fatalf("expected error for unmatched file")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(stderr, "no schema") && !strings.Contains(stderr, "unmatched") {
		t.Errorf("expected stderr to explain why the file was unmatched, got: %q", stderr)
	}
}

func TestValidate_schemaFlagWinsOverConfig(t *testing.T) {
	dir := setupScaffoldRepo(t)

	loose := `{"type":"object"}`
	loosePath := filepath.Join(dir, "schemas/loose.json")
	mustWrite(t, loosePath, loose)

	docPath := filepath.Join(dir, "notes/missing-required.md")
	mustWrite(t, docPath, "---\nslug: missing-required\ntitle: Missing Required\n---\n# Missing Required\n")

	// Config would apply object schema `book` (title+year required), but
	// --schema overrides object checks while leaving markdown/filesystem
	// checks active.
	if _, _, err := runRoot(t, "validate", "--schema", loosePath, docPath); err != nil {
		t.Fatalf("--schema should have overridden config rules: %v", err)
	}

	nonObjectFailure := filepath.Join(dir, "notes/non-object-fail.md")
	mustWrite(t, nonObjectFailure, "---\nslug: wrong-slug\ntitle: Non Object Fail\n---\n# Non Object Fail\n")
	_, stderr, err := runRoot(t, "validate", "--schema", loosePath, nonObjectFailure)
	if err == nil {
		t.Fatalf("expected non-object checks to still fail with --schema")
	}
	if !strings.Contains(stderr, "slug") {
		t.Fatalf("expected slug mismatch in stderr, got: %q", stderr)
	}
}

func TestValidate_objectCheck_reportsTypeError(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), objectCheckConfigFixture)
	mustWrite(t, filepath.Join(dir, "schemas/book.json"), bookSchemaFixture)
	chdir(t, dir)

	docPath := filepath.Join(dir, "notes/bad.md")
	mustWrite(t, docPath, "---\ntitle: Dune\nyear: not-a-number\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected object check failure")
	}
	if !strings.Contains(stderr, "/year") {
		t.Fatalf("expected /year in stderr, got: %q", stderr)
	}
}

func TestValidate_markdownCheck_reportsTitleMismatch(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), markdownCheckConfigFixture)
	chdir(t, dir)

	docPath := filepath.Join(dir, "notes/dune.md")
	mustWrite(t, docPath, "---\ntitle: Dune\n---\n# Children of Dune\n")

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected markdown check failure")
	}
	if !strings.Contains(stderr, "does not match first H1") {
		t.Fatalf("expected title/H1 mismatch message, got: %q", stderr)
	}
}

func TestValidate_filesystemCheck_reportsSlugMismatch(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "katalyst.yaml"), filesystemCheckConfigFixture)
	chdir(t, dir)

	docPath := filepath.Join(dir, "notes/dune.md")
	mustWrite(t, docPath, "---\nslug: dune-messiah\n---\n# Dune Messiah\n")

	_, stderr, err := runRoot(t, "validate", docPath)
	if err == nil {
		t.Fatalf("expected filesystem check failure")
	}
	if !strings.Contains(stderr, "must match filename") {
		t.Fatalf("expected slug/filename mismatch message, got: %q", stderr)
	}
}

func TestValidate_newRuleKinds_reportViolations(t *testing.T) {
	tests := []struct {
		name       string
		checkYAML  string
		docPath    string
		docContent string
		want       string
		setup      func(t *testing.T, dir string)
	}{
		{
			name:      "object_required_field",
			checkYAML: "- kind: object_required_field\n        field: year",
			docPath:   "notes/a.md",
			docContent: `---
title: Dune
---
# Dune
`,
			want: "missing required field",
		},
		{
			name:      "object_field_type",
			checkYAML: "- kind: object_field_type\n        field: year\n        type: integer",
			docPath:   "notes/a.md",
			docContent: `---
year: no
---
# Dune
`,
			want: "must be type",
		},
		{
			name:      "object_field_enum",
			checkYAML: "- kind: object_field_enum\n        field: status\n        values: [draft, published]",
			docPath:   "notes/a.md",
			docContent: `---
status: archived
---
# Dune
`,
			want: "allowed set",
		},
		{
			name:      "object_number_range",
			checkYAML: "- kind: object_number_range\n        field: year\n        min: 1900\n        max: 2100",
			docPath:   "notes/a.md",
			docContent: `---
year: 1800
---
# Dune
`,
			want: "must be >=",
		},
		{
			name:      "object_string_length",
			checkYAML: "- kind: object_string_length\n        field: title\n        min_length: 3",
			docPath:   "notes/a.md",
			docContent: `---
title: D
---
# D
`,
			want: "length",
		},
		{
			name:      "markdown_requires_h1",
			checkYAML: "- kind: markdown_requires_h1",
			docPath:   "notes/a.md",
			docContent: `---
title: Dune
---
No heading
`,
			want: "missing H1",
		},
		{
			name:      "markdown_single_h1",
			checkYAML: "- kind: markdown_single_h1",
			docPath:   "notes/a.md",
			docContent: `---
title: Dune
---
# One
# Two
`,
			want: "only one H1",
		},
		{
			name:      "markdown_no_heading_level_jumps",
			checkYAML: "- kind: markdown_no_heading_level_jumps",
			docPath:   "notes/a.md",
			docContent: `---
title: Dune
---
# One
### Jump
`,
			want: "jump",
		},
		{
			name:      "markdown_required_section",
			checkYAML: "- kind: markdown_required_section\n        heading: Summary",
			docPath:   "notes/a.md",
			docContent: `---
title: Dune
---
# Dune
## Notes
`,
			want: "required section",
		},
		{
			name:       "markdown_code_fence_language_required",
			checkYAML:  "- kind: markdown_code_fence_language_required",
			docPath:    "notes/a.md",
			docContent: "---\ntitle: Dune\n---\n```\ntext\n```\n",
			want:       "code fence",
		},
		{
			name:      "filesystem_extension_in",
			checkYAML: "- kind: filesystem_extension_in\n        values: [.md]",
			docPath:   "notes/a.txt",
			docContent: `---
title: Dune
---
# Dune
`,
			want: "extension",
		},
		{
			name:      "filesystem_filename_kebab_case",
			checkYAML: "- kind: filesystem_filename_kebab_case",
			docPath:   "notes/Bad Name.md",
			docContent: `---
title: Dune
---
# Dune
`,
			want: "kebab-case",
		},
		{
			name:      "filesystem_no_spaces_in_path",
			checkYAML: "- kind: filesystem_no_spaces_in_path",
			docPath:   "notes/with space.md",
			docContent: `---
title: Dune
---
# Dune
`,
			want: "spaces",
		},
		{
			name:      "filesystem_parent_dir_in",
			checkYAML: "- kind: filesystem_parent_dir_in\n        values: [books]",
			docPath:   "notes/a.md",
			docContent: `---
title: Dune
---
# Dune
`,
			want: "parent directory",
		},
		{
			name:      "filesystem_filename_prefix",
			checkYAML: "- kind: filesystem_filename_prefix\n        value: book-",
			docPath:   "notes/a.md",
			docContent: `---
title: Dune
---
# Dune
`,
			want: "prefix",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := "schemas: {}\nrules:\n  - paths: \"**/*\"\n    checks:\n      " + tc.checkYAML + "\n"
			mustWrite(t, filepath.Join(dir, "katalyst.yaml"), cfg)
			docPath := filepath.Join(dir, tc.docPath)
			mustWrite(t, docPath, tc.docContent)
			if tc.setup != nil {
				tc.setup(t, dir)
			}
			chdir(t, dir)

			_, stderr, err := runRoot(t, "validate", docPath)
			if err == nil {
				t.Fatalf("expected validation failure for %s", tc.name)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Fatalf("expected stderr to contain %q, got: %q", tc.want, stderr)
			}
		})
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
