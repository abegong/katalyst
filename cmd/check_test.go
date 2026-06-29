package cmd_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// setupNotesRepo writes a project with the book schema and a single
// `notes` collection defined by the given collection body, then chdirs in.
// The book schema requires title+year.
func setupNotesRepo(t *testing.T, notesCollection string) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml":       schemaFormatJSON,
		"schemas/book.json": bookSchemaFixture,
		"bases/local.yaml":  baseLocal(map[string]string{"notes": notesCollection}),
	})
	chdir(t, dir)
	return dir
}

const objectNotesConfig = `path: notes
schema: book
`

func TestCheck_validItem_OK(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	stdout, _, err := runRoot(t, "check", "notes/dune")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("expected OK, got: %q", stdout)
	}
}

func TestCheck_frontmatterlessFile_lintsBodyText(t *testing.T) {
	dir := setupNotesRepo(t, "path: notes\npattern: \"*.txt\"\nchecks:\n  - kind: text_forbids\n    target: line\n    pattern: '\\bTODO\\b'\n")
	mustWrite(t, filepath.Join(dir, "notes/a.txt"), "plain line\nhas TODO here\n")

	_, stderr, err := runRoot(t, "check", "notes/a")
	if err == nil {
		t.Fatalf("expected a violation on the frontmatter-less file")
	}
	if !strings.Contains(stderr, "forbidden text") || !strings.Contains(stderr, "a.txt:2:") {
		t.Errorf("expected a TODO violation on line 2, got: %q", stderr)
	}
}

func TestCheck_frontmatterlessFile_passesWhenClean(t *testing.T) {
	dir := setupNotesRepo(t, "path: notes\npattern: \"*.txt\"\nchecks:\n  - kind: text_forbids\n    target: line\n    pattern: '\\bTODO\\b'\n")
	mustWrite(t, filepath.Join(dir, "notes/clean.txt"), "all good here\n")

	stdout, _, err := runRoot(t, "check", "notes/clean")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("expected OK on a clean frontmatter-less file, got: %q", stdout)
	}
}

func TestCheck_invalidItem_exit1WithPointer(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	// year is a string → type error at /year on line 3.
	mustWrite(t, filepath.Join(dir, "notes/bad.md"), "---\ntitle: Dune\nyear: \"not a number\"\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/bad")
	if err == nil {
		t.Fatalf("expected validation failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	// The diagnostic voice (path:line: /pointer: message) is pinned as a
	// snapshot; normTmp stabilizes the absolute item path.
	snapshot(t, "check/invalid-pointer.txt", stderr, normTmp(dir))
}

func TestCheck_missingRequired_fallsBackToAncestorLine(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/missing.md"), "---\ntitle: Dune\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/missing")
	if err == nil {
		t.Fatalf("expected failure for missing required field")
	}
	if !strings.Contains(stderr, "year") {
		t.Errorf("expected mention of missing 'year', got: %q", stderr)
	}
	// The error is attributed to a known ancestor line (the document has
	// a frontmatter root), so a line number is present.
	missing := filepath.Join(dir, "notes/missing.md")
	if !strings.Contains(stderr, missing+":") {
		t.Errorf("expected a path:line prefix, got: %q", stderr)
	}
}

func TestCheck_wholeProjectWhenNoSelector(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/a.md"), "---\ntitle: A\nyear: 1\n---\n# A\n")
	mustWrite(t, filepath.Join(dir, "notes/b.md"), "---\ntitle: B\nyear: 2\n---\n# B\n")

	stdout, _, err := runRoot(t, "check")
	if err != nil {
		t.Fatalf("check (whole project): %v", err)
	}
	if strings.Count(stdout, "OK") != 2 {
		t.Errorf("expected 2 OK lines, got: %q", stdout)
	}
}

func TestCheck_filesystemChecks_runWithoutCollections(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": `type: filesystem
root: .
filesystemChecks:
  - name: docs
    path: docs
    include: ["**/*.md"]
    checks:
      - kind: filesystem_name_case
        style: kebab
collections: {}
`,
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "docs/BadName.md"), "---\ntitle: Bad\n---\n# Bad\n")

	_, stderr, err := runRoot(t, "check")
	if err == nil {
		t.Fatalf("expected filesystem check failure")
	}
	if !strings.Contains(stderr, "filesystem docs: BadName.md") || !strings.Contains(stderr, "must be kebab-case") {
		t.Errorf("expected filesystem name-case diagnostic, got: %q", stderr)
	}
}

func TestCheck_selectorDoesNotRunFilesystemChecks(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": `type: filesystem
root: .
filesystemChecks:
  - name: docs
    path: docs
    include: ["**/*.md"]
    checks:
      - kind: filesystem_name_case
        style: kebab
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
`,
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "docs/BadName.md"), "---\ntitle: Bad\n---\n# Bad\n")
	mustWrite(t, filepath.Join(dir, "notes/good.md"), "---\ntitle: Good\n---\n# Good\n")

	stdout, stderr, err := runRoot(t, "check", "notes")
	if err != nil {
		t.Fatalf("selector check should ignore filesystem scopes: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "good.md: OK") {
		t.Errorf("expected collection item OK, got: %q", stdout)
	}
	if strings.Contains(stderr, "BadName") {
		t.Errorf("selector run should not report filesystem scope diagnostics, got: %q", stderr)
	}
}

func TestCheck_filesystemParseFailuresDefaultToError(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": `type: filesystem
root: .
filesystemChecks:
  - name: docs
    path: docs
    include: ["**/*.md"]
    checks:
      - kind: filesystem_name_matches_field
        field: title
collections: {}
`,
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "docs/bad.md"), "---\n: bad\n---\n# Bad\n")

	_, stderr, err := runRoot(t, "check")
	if err == nil {
		t.Fatalf("expected parse failure to fail by default")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(stderr, "filesystem docs: bad.md") || !strings.Contains(stderr, "parse document") {
		t.Errorf("expected parse diagnostic, got: %q", stderr)
	}
}

func TestCheck_filesystemParseFailuresCanWarn(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": `type: filesystem
root: .
filesystemChecks:
  - name: docs
    path: docs
    include: ["**/*.md"]
    parseFailures: warning
    checks:
      - kind: filesystem_name_matches_field
        field: title
collections: {}
`,
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "docs/bad.md"), "---\n: bad\n---\n# Bad\n")

	_, stderr, err := runRoot(t, "check")
	if err != nil {
		t.Fatalf("warning parse failure should not fail the run: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stderr, "warning: /: parse document") {
		t.Errorf("expected warning parse diagnostic, got: %q", stderr)
	}
}

func TestCheck_filesystemUnmatchedFiles(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": `type: filesystem
root: .
filesystemChecks:
  - name: docs
    path: docs
    include: ["**/*.md"]
    exclude: ["ignored/**"]
    checks:
      - kind: filesystem_unmatched_files
collections: {}
`,
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "docs/page.md"), "---\ntitle: Page\n---\n# Page\n")
	mustWrite(t, filepath.Join(dir, "docs/raw.txt"), "raw\n")
	mustWrite(t, filepath.Join(dir, "docs/ignored/raw.txt"), "ignored\n")

	_, stderr, err := runRoot(t, "check")
	if err == nil {
		t.Fatalf("expected unmatched filesystem file failure")
	}
	if !strings.Contains(stderr, "filesystem docs: raw.txt") || !strings.Contains(stderr, "unmatched file") {
		t.Errorf("expected unmatched-file diagnostic, got: %q", stderr)
	}
	if strings.Contains(stderr, "ignored/raw.txt") {
		t.Errorf("excluded files should not be reported, got: %q", stderr)
	}
}

func TestCheck_unmatchedFileInCollectionDir_isError(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	mustWrite(t, filepath.Join(dir, "notes/ok.md"), "---\ntitle: Ok\nyear: 1\n---\n# Ok\n")
	// A non-matching file sitting in the collection directory.
	mustWrite(t, filepath.Join(dir, "notes/stray.txt"), "not markdown\n")

	_, stderr, err := runRoot(t, "check", "notes")
	if err == nil {
		t.Fatalf("expected unmatched file to cause failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	snapshot(t, "check/unmatched.txt", stderr, normTmp(dir))
}

func TestCheck_unknownSelector_exit2(t *testing.T) {
	setupNotesRepo(t, objectNotesConfig)
	_, _, err := runRoot(t, "check", "ghosts")
	if err == nil {
		t.Fatalf("expected usage error for unknown collection")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestCheck_noConfig_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "check")
	if err == nil {
		t.Fatalf("expected usage error when no config found")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

// setupVariantRepo writes a project with three YAML schemas (page requires
// title; content additionally requires weight; section requires nothing) and a
// single `pages` collection defined by the given body, then chdirs in.
func setupVariantRepo(t *testing.T, pagesBody string) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/page.yaml":    "type: object\nrequired: [title]\nproperties:\n  title: {type: string}\n",
		"schemas/section.yaml": "type: object\n",
		"schemas/content.yaml": "type: object\nrequired: [weight]\nproperties:\n  weight: {type: integer}\n",
		"bases/local.yaml":     baseLocal(map[string]string{"pages": pagesBody}),
	})
	chdir(t, dir)
	return dir
}

const variantPagesConfig = `path: pages
pattern: "**/*.md"
schema: page
variants:
  - when: "kind=section"
    schema: section
  - when: "kind!=section"
    schema: content
    checks:
      - kind: markdown_requires_h1
`

func TestCheck_variant_routesByMetadata(t *testing.T) {
	dir := setupVariantRepo(t, variantPagesConfig)
	// A section page: base title + the section variant. weight and an H1 are
	// NOT required (they live in the content variant), this is the exemption.
	mustWrite(t, filepath.Join(dir, "pages/intro.md"), "---\nkind: section\ntitle: Intro\n---\n")
	// A content page satisfies base title + content weight + requires_h1.
	mustWrite(t, filepath.Join(dir, "pages/guide.md"), "---\nkind: page\ntitle: Guide\nweight: 1\n---\n# Guide\n")

	if _, _, err := runRoot(t, "check", "pages/intro"); err != nil {
		t.Errorf("section page should pass (weight/H1 exempt): %v", err)
	}
	if _, _, err := runRoot(t, "check", "pages/guide"); err != nil {
		t.Errorf("content page should pass: %v", err)
	}

	// A content page missing weight fails the content variant's schema.
	mustWrite(t, filepath.Join(dir, "pages/bad.md"), "---\nkind: page\ntitle: Bad\n---\n# Bad\n")
	_, stderr, err := runRoot(t, "check", "pages/bad")
	if err == nil {
		t.Fatalf("content page missing weight should fail")
	}
	if !strings.Contains(stderr, "weight") {
		t.Errorf("expected a weight violation, got: %q", stderr)
	}
}

func TestCheck_variant_additiveBaseSchema(t *testing.T) {
	dir := setupVariantRepo(t, variantPagesConfig)
	// Missing the base-required title (kind=page routes to content): the base
	// page schema and the content variant schema both apply.
	mustWrite(t, filepath.Join(dir, "pages/notitle.md"), "---\nkind: page\nweight: 1\n---\n# X\n")
	_, stderr, err := runRoot(t, "check", "pages/notitle")
	if err == nil {
		t.Fatalf("missing base-required title should fail even on a routed item")
	}
	if !strings.Contains(stderr, "title") {
		t.Errorf("expected a title violation from the base schema, got: %q", stderr)
	}
}

func TestCheck_variant_firstMatchWins(t *testing.T) {
	// Two overlapping variants: a section page matches both, but only the
	// first (requires_h1, no weight) applies, the second's weight is not
	// enforced.
	body := `path: pages
pattern: "**/*.md"
schema: page
variants:
  - when: "kind=section"
    checks:
      - kind: markdown_requires_h1
  - when: "title"
    schema: content
`
	dir := setupVariantRepo(t, body)
	// Section page with an H1 but no weight: passes, proving variant 2
	// (content/weight) did not also apply.
	mustWrite(t, filepath.Join(dir, "pages/s.md"), "---\nkind: section\ntitle: S\n---\n# S\n")
	if _, _, err := runRoot(t, "check", "pages/s"); err != nil {
		t.Errorf("first variant should win (weight not enforced): %v", err)
	}
	// Same page without an H1 fails the first variant's requires_h1, proving
	// variant 1 did apply.
	mustWrite(t, filepath.Join(dir, "pages/noh1.md"), "---\nkind: section\ntitle: NoH1\n---\nbody\n")
	if _, _, err := runRoot(t, "check", "pages/noh1"); err == nil {
		t.Errorf("first variant's requires_h1 should fail a section page without an H1")
	}
}

func TestCheck_variant_unrouted_lenientVsExhaustive(t *testing.T) {
	lenient := `path: pages
pattern: "**/*.md"
schema: page
variants:
  - when: "kind=section"
    schema: section
`
	dir := setupVariantRepo(t, lenient)
	// kind=other matches no variant; lenient default → base only (passes).
	mustWrite(t, filepath.Join(dir, "pages/loose.md"), "---\nkind: other\ntitle: Loose\n---\n")
	if _, _, err := runRoot(t, "check", "pages/loose"); err != nil {
		t.Errorf("unrouted item should pass under lenient default: %v", err)
	}

	// Same config + useExhaustiveVariants: the unrouted item now fails.
	exhaustive := lenient + "useExhaustiveVariants: true\n"
	dir = setupVariantRepo(t, exhaustive)
	mustWrite(t, filepath.Join(dir, "pages/loose.md"), "---\nkind: other\ntitle: Loose\n---\n")
	_, stderr, err := runRoot(t, "check", "pages/loose")
	if err == nil {
		t.Fatalf("unrouted item should fail under useExhaustiveVariants")
	}
	if !strings.Contains(stderr, "matches no variant") {
		t.Errorf("expected 'matches no variant', got: %q", stderr)
	}
}

func TestCheck_variant_schemaFlagOverridesBaseAndVariant(t *testing.T) {
	dir := setupVariantRepo(t, variantPagesConfig)
	// A content page missing both title and weight would fail base+variant
	// object schemas, but a loose --schema overrides the whole object tier.
	// The variant's markdown_requires_h1 still runs, so include an H1.
	loose := filepath.Join(dir, "schemas/loose.json")
	mustWrite(t, loose, `{"type":"object"}`)
	mustWrite(t, filepath.Join(dir, "pages/x.md"), "---\nkind: page\n---\n# X\n")
	if _, _, err := runRoot(t, "check", "--schema", loose, "pages/x"); err != nil {
		t.Errorf("--schema should override base and variant object checks: %v", err)
	}
}

func TestItemList_variant_unroutedStatusUnderExhaustive(t *testing.T) {
	body := `path: pages
pattern: "**/*.md"
schema: page
useExhaustiveVariants: true
variants:
  - when: "kind=section"
    schema: section
`
	dir := setupVariantRepo(t, body)
	mustWrite(t, filepath.Join(dir, "pages/loose.md"), "---\nkind: other\ntitle: Loose\n---\n")

	stdout, _, err := runRoot(t, "item", "list", "pages")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	if !strings.Contains(stdout, "loose") || !strings.Contains(stdout, "1 error") {
		t.Errorf("expected loose to list as '1 error', got: %q", stdout)
	}
}

func TestCheck_schemaFlagOverridesForAllItems(t *testing.T) {
	dir := setupNotesRepo(t, objectNotesConfig)
	// Config would require title+year (book). A loose --schema overrides
	// object validation for every selected item.
	loose := filepath.Join(dir, "schemas/loose.json")
	mustWrite(t, loose, `{"type":"object"}`)
	mustWrite(t, filepath.Join(dir, "notes/missing.md"), "---\ntitle: Missing\n---\n# Missing\n")

	if _, _, err := runRoot(t, "check", "--schema", loose, "notes/missing"); err != nil {
		t.Fatalf("--schema should have overridden config object checks: %v", err)
	}
}

func TestCheck_inlineSchemaKeyTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml":              schemaFormatJSON,
		"schemas/book.json":        bookSchemaFixture,
		"schemas/strict-book.json": strictBookSchemaFixture,
		"bases/local.yaml":         baseLocal(map[string]string{"notes": "path: notes\nschema: book\n"}),
	})
	chdir(t, dir)

	// Collection maps notes → book, but the doc opts into strict-book,
	// which additionally requires isbn.
	mustWrite(t, filepath.Join(dir, "notes/strict.md"),
		"---\nschema: strict-book\ntitle: Dune\nyear: 1965\n---\n# Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/strict")
	if err == nil {
		t.Fatalf("expected inline strict-book to fail (missing isbn)")
	}
	if !strings.Contains(stderr, "isbn") {
		t.Errorf("expected isbn in stderr, got: %q", stderr)
	}
}

func TestCheck_markdownAndFilesystemChecks(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": "path: notes\nchecks:\n  - kind: markdown_title_matches_h1\n    field: title\n"}),
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: Dune\n---\n# Children of Dune\n")

	_, stderr, err := runRoot(t, "check", "notes/dune")
	if err == nil {
		t.Fatalf("expected markdown title/H1 mismatch")
	}
	if !strings.Contains(stderr, "does not match first H1") {
		t.Errorf("expected title/H1 message, got: %q", stderr)
	}
}

// A single-item selector must still re-scan the whole collection for a
// collection-scoped check: selecting just one of two colliding items should
// still report the duplicate (and name both files).
func TestCheck_collectionScoped_rescanFullCollectionForSingleItemSelector(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": "path: notes\nchecks:\n  - kind: filesystem_unique_field\n    field: slug\n"}),
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "notes/a.md"), "---\nslug: dune\n---\n# A\n")
	mustWrite(t, filepath.Join(dir, "notes/b.md"), "---\nslug: dune\n---\n# B\n")

	// Select only a.md; the duplicate verdict must still consider b.md.
	_, stderr, err := runRoot(t, "check", "notes/a")
	if err == nil {
		t.Fatalf("expected duplicate-slug failure")
	}
	if !strings.Contains(stderr, "a.md") || !strings.Contains(stderr, "b.md") {
		t.Errorf("expected both colliding files named, got: %q", stderr)
	}
}

func TestCheck_writingTells_warnButPass(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": "path: notes\nchecks:\n  - kind: markdown_writing_tells\n"}),
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(dir, "notes/x.md"),
		"---\ntitle: X\n---\n# X\n\nWe delve in — carefully.\n")

	stdout, stderr, err := runRoot(t, "check", "notes/x")
	if err != nil {
		t.Fatalf("warnings must not fail the run, got: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "x.md: OK") {
		t.Errorf("expected OK on stdout (warnings are advisory), got: %q", stdout)
	}
	// The advisory warning voice (warning: ... em dash) is pinned as a snapshot.
	snapshot(t, "check/writing-tell.txt", stderr, normTmp(dir))
}
