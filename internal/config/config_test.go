package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/config"
)

// writeProject scaffolds a .katalyst/ tree: keys are paths relative to
// the .katalyst/ directory (e.g. "schemas/book.yaml", "config.yaml"),
// values are file contents. It always creates the .katalyst/ dir so the
// project is discoverable even when files is empty.
func writeProject(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".katalyst"), 0o755); err != nil {
		t.Fatalf("mkdir .katalyst: %v", err)
	}
	for rel, content := range files {
		p := filepath.Join(dir, ".katalyst", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(p), err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// localStorage builds a .katalyst/storage/local.yaml body: a filesystem
// instance rooted at the project, declaring the given collections verbatim.
// Each value is the collection's YAML body, indented under its name.
func localStorage(collections map[string]string) string {
	var b strings.Builder
	b.WriteString("type: filesystem\nroot: .\ncollections:\n")
	// Deterministic order keeps the fixture stable.
	names := make([]string, 0, len(collections))
	for n := range collections {
		names = append(names, n)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, n := range names {
		b.WriteString("  " + n + ":\n")
		for _, line := range strings.Split(strings.TrimRight(collections[n], "\n"), "\n") {
			if line == "" {
				b.WriteString("\n")
				continue
			}
			b.WriteString("    " + line + "\n")
		}
	}
	return b.String()
}

// minimalSchema is a placeholder schema body; the config layer records a
// schema's path but never compiles it, so the contents only need to be a
// valid file.
const minimalSchema = "type: object\n"

// realPath returns dir with symlinks resolved. macOS's $TMPDIR is
// /var/folders/... which is a symlink to /private/var/folders/...;
// Load canonicalizes via EvalSymlinks, so tests must compare against
// the resolved form.
func realPath(t *testing.T, dir string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return dir
	}
	return r
}

func TestLoad_convention_discoversSchemasAndCollections(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":   minimalSchema,
		"schemas/person.yaml": minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{
			"books":  "path: notes/books\nschema: book\n",
			"people": "path: notes/people\npattern: \"*.markdown\"\nschema: person\n",
		}),
	})

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	wantRoot := realPath(t, dir)
	if cfg.Root != wantRoot {
		t.Errorf("Root = %q, want %q", cfg.Root, wantRoot)
	}
	if len(cfg.Schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(cfg.Schemas))
	}
	if got, want := cfg.SchemaPath("book"), filepath.Join(wantRoot, ".katalyst/schemas/book.yaml"); got != want {
		t.Errorf("SchemaPath(book) = %q, want %q", got, want)
	}

	// One filesystem instance named "local".
	if len(cfg.Storage) != 1 || cfg.Storage[0].Name != "local" || cfg.Storage[0].Type != "filesystem" {
		t.Fatalf("expected one filesystem instance 'local', got %+v", cfg.Storage)
	}
	if cfg.Storage[0].Root != wantRoot {
		t.Errorf("instance Root = %q, want %q", cfg.Storage[0].Root, wantRoot)
	}

	// Collections are flattened and sorted by name: books, people.
	if got := cfg.CollectionNames(); strings.Join(got, ",") != "books,people" {
		t.Fatalf("CollectionNames = %v, want [books people]", got)
	}

	books, ok := cfg.Collection("books")
	if !ok {
		t.Fatal("expected books collection")
	}
	if books.Schema != "book" {
		t.Errorf("books.Schema = %q, want book", books.Schema)
	}
	if books.Storage != "local" {
		t.Errorf("books.Storage = %q, want local", books.Storage)
	}
	if books.Pattern != "*.md" {
		t.Errorf("books.Pattern = %q, want default *.md", books.Pattern)
	}
	if books.Dir != filepath.Join(wantRoot, "notes/books") {
		t.Errorf("books.Dir = %q", books.Dir)
	}
	if len(books.Checks) != 1 || books.Checks[0].Type != config.CheckObject {
		t.Fatalf("books schema shorthand should map to one object check, got %+v", books.Checks)
	}

	people, _ := cfg.Collection("people")
	if people.Pattern != "*.markdown" {
		t.Errorf("people.Pattern = %q, want *.markdown", people.Pattern)
	}
	if people.Ext() != ".markdown" {
		t.Errorf("people.Ext() = %q, want .markdown", people.Ext())
	}
}

func TestLoad_defaultsPathToCollectionName(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.Path != "notes" {
		t.Errorf("notes.Path = %q, want default 'notes'", notes.Path)
	}
}

func TestLoad_instanceRoot_resolvesCollectionDirs(t *testing.T) {
	// A non-default instance root is the base for its collections' Dir.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"storage/vault.yaml": "type: filesystem\nroot: content\ncollections:\n" +
			"  notes:\n    path: notes\n    schema: book\n",
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if want := filepath.Join(realPath(t, dir), "content/notes"); notes.Dir != want {
		t.Errorf("notes.Dir = %q, want %q (resolved against instance root)", notes.Dir, want)
	}
}

func TestLoad_perCollectionFiles_inInstanceDir(t *testing.T) {
	// A collection may live in its own file under storage/<instance>/, the
	// escape hatch for instances that outgrow an inline block.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":         minimalSchema,
		"storage/local.yaml":        "type: filesystem\nroot: .\ncollections: {}\n",
		"storage/local/books.yaml":  "path: notes/books\nschema: book\n",
		"storage/local/people.yaml": "path: notes/people\nschema: book\n",
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.CollectionNames(); strings.Join(got, ",") != "books,people" {
		t.Fatalf("CollectionNames = %v, want [books people]", got)
	}
	books, _ := cfg.Collection("books")
	if books.Storage != "local" {
		t.Errorf("books.Storage = %q, want local", books.Storage)
	}
}

func TestLoad_perCollectionFiles_coexistWithInline(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":        minimalSchema,
		"storage/local.yaml":       localStorage(map[string]string{"books": "path: notes/books\nschema: book\n"}),
		"storage/local/notes.yaml": "path: notes\nschema: book\n",
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.CollectionNames(); strings.Join(got, ",") != "books,notes" {
		t.Fatalf("CollectionNames = %v, want [books notes]", got)
	}
}

func TestLoad_perCollectionFiles_rejectInlineCollision(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":        minimalSchema,
		"storage/local.yaml":       localStorage(map[string]string{"notes": "path: notes\nschema: book\n"}),
		"storage/local/notes.yaml": "path: other\nschema: book\n",
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "both inline and in a file") {
		t.Fatalf("expected inline/file collision error, got: %v", err)
	}
}

func TestLoad_ascendsToFindProject(t *testing.T) {
	repo := t.TempDir()
	writeProject(t, repo, nil)
	deep := filepath.Join(repo, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(deep)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantRoot := realPath(t, repo)
	if cfg.Root != wantRoot {
		t.Errorf("Root = %q, want %q", cfg.Root, wantRoot)
	}
}

func TestLoad_noStorage_isEmptyButValid(t *testing.T) {
	// A project with schemas but no storage instances loads with zero
	// collections. There is no implicit instance synthesized.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Storage) != 0 {
		t.Errorf("expected no storage instances, got %d", len(cfg.Storage))
	}
	if len(cfg.Collections) != 0 {
		t.Errorf("expected no collections, got %d", len(cfg.Collections))
	}
}

func TestLoad_noConfigFile_usesConventionDefaults(t *testing.T) {
	// A project with a .katalyst/ dir but no config.yaml loads via the
	// default convention + yaml discovery.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Collection("notes"); !ok {
		t.Errorf("expected notes collection from convention defaults")
	}
}

func TestLoad_notFound(t *testing.T) {
	dir := t.TempDir()
	_, err := config.Load(dir)
	if !errors.Is(err, config.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLoad_rejectsUnknownStorageType(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"storage/db.yaml": "type: sqlite\ncollections:\n  notes:\n    path: notes\n    checks:\n      - kind: markdown_requires_h1\n",
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "unknown type") {
		t.Fatalf("expected unknown-type error, got: %v", err)
	}
}

func TestLoad_rejectsDuplicateCollectionAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"storage/a.yaml": "type: filesystem\ncollections:\n  notes:\n    path: a\n    checks:\n      - kind: markdown_requires_h1\n",
		"storage/b.yaml": "type: filesystem\ncollections:\n  notes:\n    path: b\n    checks:\n      - kind: markdown_requires_h1\n",
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected duplicate-collection-name error, got: %v", err)
	}
}

func TestLoad_rejectsUnknownSchemaInCollection(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": "path: notes\nschema: nonexistent\n"}),
	})
	_, err := config.Load(dir)
	if err == nil {
		t.Fatalf("expected error for collection referencing unknown schema")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the bad name: %v", err)
	}
}

func TestLoad_rejectsCollectionWithNoChecks(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"storage/local.yaml": localStorage(map[string]string{"notes": "path: notes\n"}),
	})
	_, err := config.Load(dir)
	if err == nil {
		t.Fatalf("expected error for collection with no checks")
	}
	if !strings.Contains(err.Error(), "no checks") {
		t.Errorf("expected 'no checks' message, got: %v", err)
	}
}

func TestLoad_variantsParsed(t *testing.T) {
	dir := t.TempDir()
	body := "path: pages\nschema: page\nuseExhaustiveVariants: true\n" +
		"variants:\n" +
		"  - when:\n" +
		"      where: [\"kind=section\"]\n" +
		"    schema: section\n" +
		"  - when: [\"kind!=section\", \"weight>=1\"]\n" +
		"    schema: content\n" +
		"    checks:\n" +
		"      - kind: markdown_requires_h1\n"
	writeProject(t, dir, map[string]string{
		"schemas/page.yaml":    minimalSchema,
		"schemas/section.yaml": minimalSchema,
		"schemas/content.yaml": minimalSchema,
		"storage/local.yaml":   localStorage(map[string]string{"pages": body}),
	})

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	pages, ok := cfg.Collection("pages")
	if !ok {
		t.Fatal("expected pages collection")
	}
	if !pages.UseExhaustiveVariants {
		t.Error("UseExhaustiveVariants = false, want true")
	}
	if len(pages.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(pages.Variants))
	}

	// Variant 0: one predicate, schema folded into a single leading object check.
	v0 := pages.Variants[0]
	if len(v0.Where) != 1 {
		t.Errorf("variant 0 Where = %d predicates, want 1", len(v0.Where))
	}
	if len(v0.Checks) != 1 || v0.Checks[0].Type != config.CheckObject || v0.Checks[0].Schema != "section" {
		t.Errorf("variant 0 Checks = %+v, want one object check on 'section'", v0.Checks)
	}

	// Variant 1: two ANDed predicates, schema folded plus its own check.
	v1 := pages.Variants[1]
	if len(v1.Where) != 2 {
		t.Errorf("variant 1 Where = %d predicates, want 2", len(v1.Where))
	}
	if len(v1.Checks) != 2 || v1.Checks[0].Type != config.CheckObject || v1.Checks[0].Schema != "content" {
		t.Fatalf("variant 1 Checks = %+v, want object check then requires_h1", v1.Checks)
	}
	if v1.Checks[1].Type != config.CheckMarkdownRequiresH1 {
		t.Errorf("variant 1 second check = %q, want markdown_requires_h1", v1.Checks[1].Type)
	}
}

func TestLoad_whenShorthandDesugars(t *testing.T) {
	dir := t.TempDir()
	// A single bare-string when and a list when both desugar to where-lists.
	body := "path: pages\nschema: page\n" +
		"variants:\n" +
		"  - when: \"kind=section\"\n" +
		"    checks:\n" +
		"      - kind: markdown_requires_h1\n" +
		"  - when: [\"a=1\", \"b=2\"]\n" +
		"    checks:\n" +
		"      - kind: markdown_requires_h1\n"
	writeProject(t, dir, map[string]string{
		"schemas/page.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"pages": body}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	pages, _ := cfg.Collection("pages")
	if len(pages.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(pages.Variants))
	}
	if len(pages.Variants[0].Where) != 1 {
		t.Errorf("string shorthand: Where = %d, want 1", len(pages.Variants[0].Where))
	}
	if len(pages.Variants[1].Where) != 2 {
		t.Errorf("list shorthand: Where = %d, want 2", len(pages.Variants[1].Where))
	}
}

func TestLoad_variantOnlyCollectionIsValid(t *testing.T) {
	// A collection with no base checks but at least one variant is allowed.
	dir := t.TempDir()
	body := "path: pages\n" +
		"variants:\n" +
		"  - when: \"kind=section\"\n" +
		"    checks:\n" +
		"      - kind: markdown_requires_h1\n"
	writeProject(t, dir, map[string]string{
		"storage/local.yaml": localStorage(map[string]string{"pages": body}),
	})
	if _, err := config.Load(dir); err != nil {
		t.Fatalf("variant-only collection should load: %v", err)
	}
}

func TestLoad_rejectsInvalidVariantPredicate(t *testing.T) {
	dir := t.TempDir()
	body := "path: pages\nschema: page\n" +
		"variants:\n" +
		"  - when: \"=nofield\"\n"
	writeProject(t, dir, map[string]string{
		"schemas/page.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"pages": body}),
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "variants[0]") {
		t.Fatalf("expected variants[0] predicate error, got: %v", err)
	}
}

func TestLoad_rejectsUnknownVariantSchema(t *testing.T) {
	dir := t.TempDir()
	body := "path: pages\nschema: page\n" +
		"variants:\n" +
		"  - when: \"kind=section\"\n" +
		"    schema: nonexistent\n"
	writeProject(t, dir, map[string]string{
		"schemas/page.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"pages": body}),
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "nonexistent") {
		t.Fatalf("expected unknown variant schema error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "variants[0]") {
		t.Errorf("error should locate the variant: %v", err)
	}
}

func TestLoad_rejectsEmptyWhen(t *testing.T) {
	dir := t.TempDir()
	body := "path: pages\nschema: page\n" +
		"variants:\n" +
		"  - when: []\n" +
		"    checks:\n" +
		"      - kind: markdown_requires_h1\n"
	writeProject(t, dir, map[string]string{
		"schemas/page.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"pages": body}),
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "at least one predicate") {
		t.Fatalf("expected empty-when error, got: %v", err)
	}
}

func TestLoad_useExhaustiveVariantsDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": "path: notes\nschema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.UseExhaustiveVariants {
		t.Error("UseExhaustiveVariants = true, want default false")
	}
	if len(notes.Variants) != 0 {
		t.Errorf("Variants = %d, want 0 by default", len(notes.Variants))
	}
}

func TestLoad_parsesChecks(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": `path: notes
checks:
  - kind: object
    schema: book
  - kind: object_required_field
    field: year
  - kind: object_field_type
    field: year
    type: integer
  - kind: object_field_enum
    field: status
    values: [draft, published]
  - kind: object_number_range
    field: year
    min: 1900
    max: 2100
  - kind: object_string_length
    field: title
    min_length: 1
    max_length: 100
  - kind: markdown_title_matches_h1
  - kind: markdown_requires_h1
  - kind: markdown_single_h1
  - kind: markdown_no_heading_level_jumps
  - kind: markdown_required_section
    heading: Summary
  - kind: markdown_code_fence_language_required
  - kind: filesystem_name_matches_field
  - kind: filesystem_extension_in
    values: [.md]
  - kind: filesystem_name_case
    style: kebab
  - kind: filesystem_path_charset
    deny: [" "]
  - kind: filesystem_parent_dir_in
    values: [books, notes]
  - kind: filesystem_name_affix
    prefix: book-
`}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, ok := cfg.Collection("notes")
	if !ok {
		t.Fatal("expected notes collection")
	}
	got := notes.Checks
	if len(got) != 18 {
		t.Fatalf("expected 18 checks, got %d", len(got))
	}
	if got[0].Type != config.CheckObject || got[0].Schema != "book" {
		t.Fatalf("check[0] = %+v, want object schema=book", got[0])
	}
	if got[6].Type != config.CheckMarkdownTitleMatchesH1 || got[6].Field != "title" {
		t.Fatalf("check[6] = %+v, want markdown default field title", got[6])
	}
	if got[12].Type != config.CheckFilesystemNameMatchesField || got[12].Field != "slug" || got[12].Transform != "none" {
		t.Fatalf("check[12] = %+v, want name_matches_field default field slug, transform none", got[12])
	}
	if got[14].Type != config.CheckFilesystemNameCase || got[14].Style != "kebab" {
		t.Fatalf("check[14] = %+v, want name_case style kebab", got[14])
	}
}

func TestLoad_rejectsUnknownCheckType(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": `path: notes
checks:
  - kind: not-real
`}),
	})
	_, err := config.Load(dir)
	if err == nil {
		t.Fatalf("expected error for unknown check type")
	}
	if !strings.Contains(err.Error(), "unknown check type") {
		t.Fatalf("expected unknown check type message, got: %v", err)
	}
}

func TestLoad_rejectsMalformedCheckPayload(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": `path: notes
checks:
  - kind: object
`}),
	})
	_, err := config.Load(dir)
	if err == nil {
		t.Fatalf("expected error for missing object schema")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Fatalf("expected malformed payload error, got: %v", err)
	}
}

func TestLoad_rejectsInvalidFilesystemCheckConfig(t *testing.T) {
	cases := map[string]struct{ checks, want string }{
		"name_case unknown style": {
			"  - kind: filesystem_name_case\n    style: nope\n", "unknown style",
		},
		"name_case unknown target": {
			"  - kind: filesystem_name_case\n    style: kebab\n    target: nope\n", "unknown target",
		},
		"name_affix needs prefix or suffix": {
			"  - kind: filesystem_name_affix\n", `requires "prefix" or "suffix"`,
		},
		"path_charset both allow and deny": {
			"  - kind: filesystem_path_charset\n    allow: [a]\n    deny: [b]\n", "not both",
		},
		"path_charset neither": {
			"  - kind: filesystem_path_charset\n", `requires "allow" or "deny"`,
		},
		"name_matches_field bad transform": {
			"  - kind: filesystem_name_matches_field\n    transform: shout\n", "must be none or slugify",
		},
		"name_regex bad pattern": {
			"  - kind: filesystem_name_regex\n    pattern: '['\n", "invalid pattern",
		},
		"name_length needs a bound": {
			"  - kind: filesystem_name_length\n", `requires "min" or "max"`,
		},
		"path_depth needs a bound": {
			"  - kind: filesystem_path_depth\n", `requires "min" or "max"`,
		},
		"referenced_files needs fields": {
			"  - kind: filesystem_referenced_files_exist\n", `requires "fields"`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeProject(t, dir, map[string]string{
				"storage/local.yaml": localStorage(map[string]string{"notes": "path: notes\nchecks:\n" + tc.checks}),
			})
			_, err := config.Load(dir)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestLoad_parsesTextChecks(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"storage/local.yaml": localStorage(map[string]string{"notes": `path: notes
checks:
  - kind: text_requires
    pattern: Sources
  - kind: text_requires
    target: line
    pattern: x
    match: all
  - kind: text_forbids
    target: matched-lines
    select: '^-'
    pattern: '\bTODO\b'
    fix: ''
  - kind: text_denylist
    values: [TODO, FIXME]
`}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	got := notes.Checks
	if len(got) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(got))
	}
	if got[0].Type != config.CheckTextRequires || got[0].Match != "any" {
		t.Fatalf("check[0] = %+v, want text_requires default match any", got[0])
	}
	if got[1].Match != "all" || got[1].Target != "line" {
		t.Fatalf("check[1] = %+v, want match all target line", got[1])
	}
	if got[2].Type != config.CheckTextForbids || got[2].Select != "^-" {
		t.Fatalf("check[2] = %+v, want text_forbids select ^-", got[2])
	}
	if got[3].Type != config.CheckTextDenylist || len(got[3].Values) != 2 {
		t.Fatalf("check[3] = %+v, want text_denylist with 2 values", got[3])
	}
}

func TestLoad_rejectsInvalidTextCheckConfig(t *testing.T) {
	cases := map[string]struct{ checks, want string }{
		"requires needs pattern": {
			"  - kind: text_requires\n", `text_requires requires "pattern"`,
		},
		"requires bad pattern": {
			"  - kind: text_requires\n    pattern: '['\n", "invalid pattern",
		},
		"requires bad match": {
			"  - kind: text_requires\n    pattern: x\n    match: some\n", `"match" must be any or all`,
		},
		"requires rejects fix": {
			"  - kind: text_requires\n    pattern: x\n    fix: y\n", `does not support "fix"`,
		},
		"forbids needs pattern": {
			"  - kind: text_forbids\n", `text_forbids requires "pattern"`,
		},
		"forbids rejects match": {
			"  - kind: text_forbids\n    pattern: x\n    match: any\n", `does not support "match"`,
		},
		"denylist needs values": {
			"  - kind: text_denylist\n", `text_denylist requires "values"`,
		},
		"denylist rejects fix": {
			"  - kind: text_denylist\n    values: [x]\n    fix: y\n", `does not support "fix"`,
		},
		"unknown target": {
			"  - kind: text_forbids\n    pattern: x\n    target: nope\n", "unknown target",
		},
		"select without matched-lines": {
			"  - kind: text_forbids\n    pattern: x\n    select: '^-'\n", `only valid with target "matched-lines"`,
		},
		"matched-lines without select": {
			"  - kind: text_forbids\n    pattern: x\n    target: matched-lines\n", `requires "select"`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeProject(t, dir, map[string]string{
				"storage/local.yaml": localStorage(map[string]string{"notes": "path: notes\nchecks:\n" + tc.checks}),
			})
			_, err := config.Load(dir)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestLoad_explicitDiscovery_readsDefs(t *testing.T) {
	// In explicit mode, the storage directory scan is ignored and the inline
	// defs map in config.yaml is authoritative.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"my-schemas/book.yaml": minimalSchema,
		"config.yaml": `schemas:
  discovery: explicit
  defs:
    book: ./.katalyst/my-schemas/book.yaml
storage:
  discovery: explicit
  defs:
    local:
      type: filesystem
      root: .
      collections:
        notes:
          path: notes
          schema: book
`,
		// Stray files in the convention dirs must be ignored.
		"schemas/ignored.yaml":      minimalSchema,
		"storage/ignored-inst.yaml": "type: filesystem\ncollections: {}\n",
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Schemas["ignored"]; ok {
		t.Errorf("explicit discovery must ignore the schemas/ dir scan")
	}
	for _, inst := range cfg.Storage {
		if inst.Name != "local" {
			t.Errorf("explicit discovery must ignore the storage/ dir scan, saw instance %q", inst.Name)
		}
	}
	wantRoot := realPath(t, dir)
	if got, want := cfg.SchemaPath("book"), filepath.Join(wantRoot, ".katalyst/my-schemas/book.yaml"); got != want {
		t.Errorf("SchemaPath(book) = %q, want %q", got, want)
	}
	if _, ok := cfg.Collection("notes"); !ok {
		t.Errorf("expected notes collection from explicit defs")
	}
}

func TestLoad_explicitDiscovery_requiresDefs(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": "storage:\n  discovery: explicit\n",
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "defs") {
		t.Fatalf("expected explicit-requires-defs error, got: %v", err)
	}
}

func TestLoad_formatJSON_scansJSONFiles(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.json":  `{"type":"object"}`,
		"config.yaml":        "schemas:\n  format: json\n",
		"storage/local.yaml": localStorage(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.SchemaPath("book"), filepath.Join(realPath(t, dir), ".katalyst/schemas/book.json"); got != want {
		t.Errorf("SchemaPath(book) = %q, want %q", got, want)
	}
}

func TestLoad_formatBoth_rejectsNameCollision(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"schemas/book.json": `{"type":"object"}`,
		"config.yaml":       "schemas:\n  format: both\n",
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "two files") {
		t.Fatalf("expected name-collision error, got: %v", err)
	}
}

func TestLoad_perKindIndependence(t *testing.T) {
	// Schemas explicit + json; storage convention + yaml.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.json": `{"type":"object"}`,
		"config.yaml": `schemas:
  discovery: explicit
  format: json
  defs:
    book: ./.katalyst/schemas/book.json
`,
		"storage/local.yaml": localStorage(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SchemaPath("book") == "" {
		t.Errorf("expected book schema from explicit json defs")
	}
	if _, ok := cfg.Collection("notes"); !ok {
		t.Errorf("expected notes collection from convention yaml scan")
	}
}

func TestLoad_rejectsBadDiscovery(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": "schemas:\n  discovery: bogus\n",
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "discovery") {
		t.Fatalf("expected discovery validation error, got: %v", err)
	}
}

func TestLoad_queryDefaults_whenUnset(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.Query.FilterTypeMismatch != "skip" {
		t.Errorf("FilterTypeMismatch = %q, want default skip", notes.Query.FilterTypeMismatch)
	}
	if notes.Query.SortMissing != "last" {
		t.Errorf("SortMissing = %q, want default last", notes.Query.SortMissing)
	}
}

func TestLoad_query_projectDefaultApplies(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"config.yaml": `query:
  filterTypeMismatch: error
  sortMissing: lowest
`,
		"storage/local.yaml": localStorage(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.Query.FilterTypeMismatch != "error" {
		t.Errorf("FilterTypeMismatch = %q, want error from project default", notes.Query.FilterTypeMismatch)
	}
	if notes.Query.SortMissing != "lowest" {
		t.Errorf("SortMissing = %q, want lowest from project default", notes.Query.SortMissing)
	}
}

func TestLoad_query_collectionOverridesPerKey(t *testing.T) {
	// The collection sets only filterTypeMismatch; sortMissing must fall
	// through to the project default, not back to the built-in.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"config.yaml": `query:
  sortMissing: lowest
`,
		"storage/local.yaml": localStorage(map[string]string{"notes": `schema: book
query:
  filterTypeMismatch: error
`}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.Query.FilterTypeMismatch != "error" {
		t.Errorf("FilterTypeMismatch = %q, want error from collection", notes.Query.FilterTypeMismatch)
	}
	if notes.Query.SortMissing != "lowest" {
		t.Errorf("SortMissing = %q, want lowest from project (fall-through)", notes.Query.SortMissing)
	}
}

func TestLoad_query_rejectsUnknownValue(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": `schema: book
query:
  filterTypeMismatch: bogus
`}),
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "filterTypeMismatch") {
		t.Fatalf("expected filterTypeMismatch validation error, got: %v", err)
	}
}

func TestCollection_unknownReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":  minimalSchema,
		"storage/local.yaml": localStorage(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Collection("missing"); ok {
		t.Errorf("expected no collection named 'missing'")
	}
}

func TestSchemaNames_returnsSortedNames(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/zebra.yaml":  minimalSchema,
		"schemas/apple.yaml":  minimalSchema,
		"schemas/middle.yaml": minimalSchema,
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.SchemaNames()
	want := []string{"apple", "middle", "zebra"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("SchemaNames = %v, want %v", got, want)
	}
}

func TestLoad_parsesWritingTells(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"storage/local.yaml": localStorage(map[string]string{"notes": "path: notes\nchecks:\n  - kind: markdown_writing_tells\n"}),
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if len(notes.Checks) != 1 || notes.Checks[0].Type != config.CheckMarkdownWritingTells {
		t.Fatalf("expected one markdown_writing_tells check, got %+v", notes.Checks)
	}
}
