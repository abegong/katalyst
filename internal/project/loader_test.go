package project_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/project/projecttest"
)

func assertConfiguredCheckBuilds(t *testing.T, c checks.ConfiguredCheck) {
	t.Helper()
	if c.Kind == checks.CheckObject {
		return
	}
	if _, ok := checks.Build(c.Kind, c.Args); !ok {
		t.Fatalf("%s did not build from parsed config", c.Kind)
	}
}

func TestLoad_convention_discoversSchemasAndCollections(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml":   projecttest.MinimalSchema,
		"schemas/person.yaml": projecttest.MinimalSchema,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{
			"books":  "path: notes/books\nschema: book\n",
			"people": "path: notes/people\npattern: \"*.markdown\"\nschema: person\n",
		}),
	})

	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	wantRoot := projecttest.RealPath(t, dir)
	if cfg.Root != wantRoot {
		t.Errorf("Root = %q, want %q", cfg.Root, wantRoot)
	}
	if len(cfg.Schemas) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(cfg.Schemas))
	}
	if got, want := cfg.SchemaPath("book"), filepath.Join(wantRoot, ".katalyst/schemas/book.yaml"); got != want {
		t.Errorf("SchemaPath(book) = %q, want %q", got, want)
	}

	// One filesystem base named "local".
	if len(cfg.Bases) != 1 || cfg.Bases[0].Name != "local" || cfg.Bases[0].Type != "filesystem" {
		t.Fatalf("expected one filesystem base 'local', got %+v", cfg.Bases)
	}
	if cfg.Bases[0].Root != wantRoot {
		t.Errorf("base Root = %q, want %q", cfg.Bases[0].Root, wantRoot)
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
	if books.Base != "local" {
		t.Errorf("books.Base = %q, want local", books.Base)
	}
	if books.Pattern != "*.md" {
		t.Errorf("books.Pattern = %q, want default *.md", books.Pattern)
	}
	if books.Dir != filepath.Join(wantRoot, "notes/books") {
		t.Errorf("books.Dir = %q", books.Dir)
	}
	if len(books.Checks) != 1 || books.Checks[0].Kind != checks.CheckObject {
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
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.Path != "notes" {
		t.Errorf("notes.Path = %q, want default 'notes'", notes.Path)
	}
}

func TestLoad_instanceRoot_resolvesCollectionDirs(t *testing.T) {
	// A non-default base root is the base for its collections' Dir.
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/vault.yaml": "type: filesystem\nroot: content\ncollections:\n" +
			"  notes:\n    path: notes\n    schema: book\n",
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if want := filepath.Join(projecttest.RealPath(t, dir), "content/notes"); notes.Dir != want {
		t.Errorf("notes.Dir = %q, want %q (resolved against base root)", notes.Dir, want)
	}
}

func TestLoad_filesystemChecks_loadsScopes(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml": `type: filesystem
root: content
filesystemChecks:
  - name: docs
    include: ["**/*.md"]
    exclude: ["drafts/**"]
    parseFailures: warning
    checks:
      - kind: filesystem_name_case
        style: kebab
collections: {}
`,
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	scopes := cfg.FilesystemCheckScopes()
	if len(scopes) != 1 {
		t.Fatalf("expected one filesystem check scope, got %d", len(scopes))
	}
	scope := scopes[0]
	if scope.Name != "docs" {
		t.Errorf("scope.Name = %q, want docs", scope.Name)
	}
	if scope.Path != "." {
		t.Errorf("scope.Path = %q, want default .", scope.Path)
	}
	if want := filepath.Join(projecttest.RealPath(t, dir), "content"); scope.Root != want {
		t.Errorf("scope.Root = %q, want %q", scope.Root, want)
	}
	if scope.ParseFailures != "warning" {
		t.Errorf("ParseFailures = %q, want warning", scope.ParseFailures)
	}
	if len(scope.Checks) != 1 || scope.Checks[0].Kind != checks.CheckFilesystemNameCase {
		t.Fatalf("unexpected checks: %+v", scope.Checks)
	}
	assertConfiguredCheckBuilds(t, scope.Checks[0])
}

func TestLoad_filesystemChecks_defaultsNameAndParseFailures(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml": `type: filesystem
root: .
filesystemChecks:
  - path: docs/content
    include: ["**/*.md"]
    checks:
      - kind: filesystem_name_case
        style: kebab
collections: {}
`,
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	scope := cfg.FilesystemCheckScopes()[0]
	if scope.Name != "docs/content" {
		t.Errorf("scope.Name = %q, want docs/content", scope.Name)
	}
	if scope.ParseFailures != "error" {
		t.Errorf("ParseFailures = %q, want error", scope.ParseFailures)
	}
}

func TestLoad_filesystemChecks_rejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{
			name: "missing include",
			base: `type: filesystem
filesystemChecks:
  - checks:
      - kind: filesystem_name_case
        style: kebab
collections: {}
`,
			want: "include is required",
		},
		{
			name: "bad parseFailures",
			base: `type: filesystem
filesystemChecks:
  - include: ["**/*.md"]
    parseFailures: notice
    checks:
      - kind: filesystem_name_case
        style: kebab
collections: {}
`,
			want: "unknown parseFailures",
		},
		{
			name: "collection-only check",
			base: `type: filesystem
filesystemChecks:
  - include: ["**/*.md"]
    checks:
      - kind: markdown_requires_h1
collections: {}
`,
			want: "does not support filesystem checks",
		},
		{
			name: "sqlite base",
			base: `type: sqlite
root: data.db
filesystemChecks:
  - include: ["**/*.md"]
    checks:
      - kind: filesystem_name_case
        style: kebab
collections: {}
`,
			want: "filesystemChecks requires type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			projecttest.WriteProject(t, dir, map[string]string{"bases/local.yaml": tt.base})
			_, err := project.Load(dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got: %v", tt.want, err)
			}
		})
	}
}

func TestLoad_perCollectionFiles_inInstanceDir(t *testing.T) {
	// A collection may live in its own file under bases/<base>/, the escape
	// hatch for bases that outgrow an inline block.
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml":       projecttest.MinimalSchema,
		"bases/local.yaml":        "type: filesystem\nroot: .\ncollections: {}\n",
		"bases/local/books.yaml":  "path: notes/books\nschema: book\n",
		"bases/local/people.yaml": "path: notes/people\nschema: book\n",
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.CollectionNames(); strings.Join(got, ",") != "books,people" {
		t.Fatalf("CollectionNames = %v, want [books people]", got)
	}
	books, _ := cfg.Collection("books")
	if books.Base != "local" {
		t.Errorf("books.Base = %q, want local", books.Base)
	}
}

func TestLoad_perCollectionFiles_coexistWithInline(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml":      projecttest.MinimalSchema,
		"bases/local.yaml":       projecttest.LocalBase(map[string]string{"books": "path: notes/books\nschema: book\n"}),
		"bases/local/notes.yaml": "path: notes\nschema: book\n",
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.CollectionNames(); strings.Join(got, ",") != "books,notes" {
		t.Fatalf("CollectionNames = %v, want [books notes]", got)
	}
}

func TestLoad_perCollectionFiles_rejectInlineCollision(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml":      projecttest.MinimalSchema,
		"bases/local.yaml":       projecttest.LocalBase(map[string]string{"notes": "path: notes\nschema: book\n"}),
		"bases/local/notes.yaml": "path: other\nschema: book\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "both inline and in a file") {
		t.Fatalf("expected inline/file collision error, got: %v", err)
	}
}

func TestLoad_ascendsToFindProject(t *testing.T) {
	repo := t.TempDir()
	projecttest.WriteProject(t, repo, nil)
	deep := filepath.Join(repo, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := project.Load(deep)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantRoot := projecttest.RealPath(t, repo)
	if cfg.Root != wantRoot {
		t.Errorf("Root = %q, want %q", cfg.Root, wantRoot)
	}
}

func TestLoad_noBases_isEmptyButValid(t *testing.T) {
	// A project with schemas but no bases loads with zero
	// collections. There is no implicit base synthesized.
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Bases) != 0 {
		t.Errorf("expected no bases, got %d", len(cfg.Bases))
	}
	if len(cfg.Collections) != 0 {
		t.Errorf("expected no collections, got %d", len(cfg.Collections))
	}
}

func TestLoad_noConfigFile_usesConventionDefaults(t *testing.T) {
	// A project with a .katalyst/ dir but no config.yaml loads via the
	// default convention + yaml discovery.
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Collection("notes"); !ok {
		t.Errorf("expected notes collection from convention defaults")
	}
}

func TestLoad_notFound(t *testing.T) {
	dir := t.TempDir()
	_, err := project.Load(dir)
	if !errors.Is(err, project.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestLoad_rejectsUnknownBaseType(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/db.yaml": "type: postgres\ncollections:\n  notes:\n    path: notes\n    checks:\n      - kind: markdown_requires_h1\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "unknown type") {
		t.Fatalf("expected unknown-type error, got: %v", err)
	}
}

func TestLoad_rejectsDuplicateCollectionAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/a.yaml": "type: filesystem\ncollections:\n  notes:\n    path: a\n    checks:\n      - kind: markdown_requires_h1\n",
		"bases/b.yaml": "type: filesystem\ncollections:\n  notes:\n    path: b\n    checks:\n      - kind: markdown_requires_h1\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected duplicate-collection-name error, got: %v", err)
	}
}

func TestLoad_rejectsUnknownSchemaInCollection(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"notes": "path: notes\nschema: nonexistent\n"}),
	})
	_, err := project.Load(dir)
	if err == nil {
		t.Fatalf("expected error for collection referencing unknown schema")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the bad name: %v", err)
	}
}

func TestLoad_rejectsCollectionWithNoChecks(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": "path: notes\n"}),
	})
	_, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/page.yaml":    projecttest.MinimalSchema,
		"schemas/section.yaml": projecttest.MinimalSchema,
		"schemas/content.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":     projecttest.LocalBase(map[string]string{"pages": body}),
	})

	cfg, err := project.Load(dir)
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
	if len(v0.Checks) != 1 || v0.Checks[0].Kind != checks.CheckObject || v0.Checks[0].Schema != "section" {
		t.Errorf("variant 0 Checks = %+v, want one object check on 'section'", v0.Checks)
	}

	// Variant 1: two ANDed predicates, schema folded plus its own check.
	v1 := pages.Variants[1]
	if len(v1.Where) != 2 {
		t.Errorf("variant 1 Where = %d predicates, want 2", len(v1.Where))
	}
	if len(v1.Checks) != 2 || v1.Checks[0].Kind != checks.CheckObject || v1.Checks[0].Schema != "content" {
		t.Fatalf("variant 1 Checks = %+v, want object check then requires_h1", v1.Checks)
	}
	if v1.Checks[1].Kind != checks.CheckMarkdownRequiresH1 {
		t.Errorf("variant 1 second check = %q, want markdown_requires_h1", v1.Checks[1].Kind)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/page.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"pages": body}),
	})
	cfg, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"pages": body}),
	})
	if _, err := project.Load(dir); err != nil {
		t.Fatalf("variant-only collection should load: %v", err)
	}
}

func TestLoad_rejectsInvalidVariantPredicate(t *testing.T) {
	dir := t.TempDir()
	body := "path: pages\nschema: page\n" +
		"variants:\n" +
		"  - when: \"=nofield\"\n"
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/page.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"pages": body}),
	})
	_, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/page.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"pages": body}),
	})
	_, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/page.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"pages": body}),
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "at least one predicate") {
		t.Fatalf("expected empty-when error, got: %v", err)
	}
}

func TestLoad_useExhaustiveVariantsDefaultsFalse(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"notes": "path: notes\nschema: book\n"}),
	})
	cfg, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `path: notes
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
	cfg, err := project.Load(dir)
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
	if got[0].Kind != checks.CheckObject || got[0].Schema != "book" {
		t.Fatalf("check[0] = %+v, want object schema=book", got[0])
	}
	if got[6].Kind != checks.CheckMarkdownTitleMatchesH1 {
		t.Fatalf("check[6].Kind = %v, want markdown_title_matches_h1", got[6].Kind)
	}
	if got[12].Kind != checks.CheckFilesystemNameMatchesField {
		t.Fatalf("check[12].Kind = %v, want filesystem_name_matches_field", got[12].Kind)
	}
	if got[14].Kind != checks.CheckFilesystemNameCase {
		t.Fatalf("check[14].Kind = %v, want filesystem_name_case", got[14].Kind)
	}
	for _, c := range got {
		assertConfiguredCheckBuilds(t, c)
	}
}

func TestLoad_rejectsUnknownCheckType(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `path: notes
checks:
  - kind: not-real
`}),
	})
	_, err := project.Load(dir)
	if err == nil {
		t.Fatalf("expected error for unknown check type")
	}
	if !strings.Contains(err.Error(), "unknown check type") {
		t.Fatalf("expected unknown check type message, got: %v", err)
	}
}

func TestLoad_rejectsUnknownCheckKey(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `path: notes
checks:
  - kind: markdown_requires_h1
    typo: true
`}),
	})
	_, err := project.Load(dir)
	if err == nil {
		t.Fatalf("expected error for unknown check key")
	}
	if !strings.Contains(err.Error(), `unknown check key "typo"`) {
		t.Fatalf("expected unknown check key error, got: %v", err)
	}
}

func TestLoad_rejectsMalformedCheckPayload(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `path: notes
checks:
  - kind: object
`}),
	})
	_, err := project.Load(dir)
	if err == nil {
		t.Fatalf("expected error for missing object schema")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Fatalf("expected malformed payload error, got: %v", err)
	}
}

func TestLoad_rejectsObjectCheckField(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `path: notes
checks:
  - kind: object
    schema: book
    field: title
`}),
	})
	_, err := project.Load(dir)
	if err == nil {
		t.Fatalf("expected error for unsupported object field")
	}
	if !strings.Contains(err.Error(), `does not support "field"`) {
		t.Fatalf("expected unsupported field error, got: %v", err)
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
			projecttest.WriteProject(t, dir, map[string]string{
				"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": "path: notes\nchecks:\n" + tc.checks}),
			})
			_, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `path: notes
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
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	got := notes.Checks
	if len(got) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(got))
	}
	if got[0].Kind != checks.CheckTextRequires {
		t.Fatalf("check[0].Kind = %v, want text_requires", got[0].Kind)
	}
	if got[1].Kind != checks.CheckTextRequires {
		t.Fatalf("check[1].Kind = %v, want text_requires", got[1].Kind)
	}
	if got[2].Kind != checks.CheckTextForbids {
		t.Fatalf("check[2].Kind = %v, want text_forbids", got[2].Kind)
	}
	if got[3].Kind != checks.CheckTextDenylist {
		t.Fatalf("check[3].Kind = %v, want text_denylist", got[3].Kind)
	}
	for _, c := range got {
		assertConfiguredCheckBuilds(t, c)
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
			projecttest.WriteProject(t, dir, map[string]string{
				"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": "path: notes\nchecks:\n" + tc.checks}),
			})
			_, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"my-schemas/book.yaml": projecttest.MinimalSchema,
		"config.yaml": `schemas:
  discovery: explicit
  defs:
    book: ./.katalyst/my-schemas/book.yaml
bases:
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
		"schemas/ignored.yaml":    projecttest.MinimalSchema,
		"bases/ignored-inst.yaml": "type: filesystem\ncollections: {}\n",
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Schemas["ignored"]; ok {
		t.Errorf("explicit discovery must ignore the schemas/ dir scan")
	}
	for _, inst := range cfg.Bases {
		if inst.Name != "local" {
			t.Errorf("explicit discovery must ignore the bases/ dir scan, saw base %q", inst.Name)
		}
	}
	wantRoot := projecttest.RealPath(t, dir)
	if got, want := cfg.SchemaPath("book"), filepath.Join(wantRoot, ".katalyst/my-schemas/book.yaml"); got != want {
		t.Errorf("SchemaPath(book) = %q, want %q", got, want)
	}
	if _, ok := cfg.Collection("notes"); !ok {
		t.Errorf("expected notes collection from explicit defs")
	}
}

func TestLoad_legacyStorageBlock_readsDefs(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"config.yaml": `storage:
  discovery: explicit
  defs:
    local:
      type: filesystem
      root: .
      collections:
        notes:
          schema: book
`,
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Bases) != 1 || cfg.Bases[0].Name != "local" {
		t.Fatalf("expected legacy storage block to load one base, got %+v", cfg.Bases)
	}
	if _, ok := cfg.Collection("notes"); !ok {
		t.Errorf("expected notes collection from legacy storage block")
	}
}

func TestLoad_legacyStorageDir_readsConventionFiles(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml":  projecttest.MinimalSchema,
		"storage/local.yaml": projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Bases) != 1 || cfg.Bases[0].Name != "local" {
		t.Fatalf("expected legacy storage dir to load one base, got %+v", cfg.Bases)
	}
}

func TestLoad_rejectsBasesAndStorageBlocks(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"config.yaml": "bases:\n  discovery: convention\nstorage:\n  discovery: convention\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "both bases and storage") {
		t.Fatalf("expected mixed config block error, got: %v", err)
	}
}

func TestLoad_rejectsBasesAndStorageDirs(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml":   "type: filesystem\ncollections: {}\n",
		"storage/local.yaml": "type: filesystem\ncollections: {}\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), ".katalyst/bases") || !strings.Contains(err.Error(), ".katalyst/storage") {
		t.Fatalf("expected mixed config dir error, got: %v", err)
	}
}

func TestLoad_explicitDiscovery_requiresDefs(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"config.yaml": "bases:\n  discovery: explicit\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "defs") {
		t.Fatalf("expected explicit-requires-defs error, got: %v", err)
	}
}

func TestLoad_formatJSON_scansJSONFiles(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.json": `{"type":"object"}`,
		"config.yaml":       "schemas:\n  format: json\n",
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.SchemaPath("book"), filepath.Join(projecttest.RealPath(t, dir), ".katalyst/schemas/book.json"); got != want {
		t.Errorf("SchemaPath(book) = %q, want %q", got, want)
	}
}

func TestLoad_formatBoth_rejectsNameCollision(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"schemas/book.json": `{"type":"object"}`,
		"config.yaml":       "schemas:\n  format: both\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "two files") {
		t.Fatalf("expected name-collision error, got: %v", err)
	}
}

func TestLoad_perKindIndependence(t *testing.T) {
	// Schemas explicit + json; storage convention + yaml.
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.json": `{"type":"object"}`,
		"config.yaml": `schemas:
  discovery: explicit
  format: json
  defs:
    book: ./.katalyst/schemas/book.json
`,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"config.yaml": "schemas:\n  discovery: bogus\n",
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "discovery") {
		t.Fatalf("expected discovery validation error, got: %v", err)
	}
}

func TestLoad_listingDefaults_whenUnset(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.ListingDefaults.FilterTypeMismatch != "skip" {
		t.Errorf("FilterTypeMismatch = %q, want default skip", notes.ListingDefaults.FilterTypeMismatch)
	}
	if notes.ListingDefaults.SortMissing != "last" {
		t.Errorf("SortMissing = %q, want default last", notes.ListingDefaults.SortMissing)
	}
}

func TestLoad_listing_projectDefaultApplies(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"config.yaml": `listing:
  filterTypeMismatch: error
  sortMissing: lowest
`,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.ListingDefaults.FilterTypeMismatch != "error" {
		t.Errorf("FilterTypeMismatch = %q, want error from project default", notes.ListingDefaults.FilterTypeMismatch)
	}
	if notes.ListingDefaults.SortMissing != "lowest" {
		t.Errorf("SortMissing = %q, want lowest from project default", notes.ListingDefaults.SortMissing)
	}
}

func TestLoad_listing_collectionOverridesPerKey(t *testing.T) {
	// The collection sets only filterTypeMismatch; sortMissing must fall
	// through to the project default, not back to the built-in.
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"config.yaml": `listing:
  sortMissing: lowest
`,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `schema: book
listing:
  filterTypeMismatch: error
`}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if notes.ListingDefaults.FilterTypeMismatch != "error" {
		t.Errorf("FilterTypeMismatch = %q, want error from collection", notes.ListingDefaults.FilterTypeMismatch)
	}
	if notes.ListingDefaults.SortMissing != "lowest" {
		t.Errorf("SortMissing = %q, want lowest from project (fall-through)", notes.ListingDefaults.SortMissing)
	}
}

func TestLoad_listing_rejectsUnknownValue(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `schema: book
listing:
  filterTypeMismatch: bogus
`}),
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "filterTypeMismatch") {
		t.Fatalf("expected filterTypeMismatch validation error, got: %v", err)
	}
}

func TestLoad_rejectsProjectQueryConfigBlock(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"config.yaml": `query:
  sortMissing: lowest
`,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "query is no longer a config block; use listing") {
		t.Fatalf("expected query migration error, got: %v", err)
	}
}

func TestLoad_rejectsCollectionQueryConfigBlock(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": `schema: book
query:
  sortMissing: lowest
`}),
	})
	_, err := project.Load(dir)
	if err == nil || !strings.Contains(err.Error(), `collection "notes": query is no longer a config block; use listing`) {
		t.Fatalf("expected collection query migration error, got: %v", err)
	}
}

func TestCollection_unknownReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/book.yaml": projecttest.MinimalSchema,
		"bases/local.yaml":  projecttest.LocalBase(map[string]string{"notes": "schema: book\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Collection("missing"); ok {
		t.Errorf("expected no collection named 'missing'")
	}
}

func TestSchemaNames_returnsSortedNames(t *testing.T) {
	dir := t.TempDir()
	projecttest.WriteProject(t, dir, map[string]string{
		"schemas/zebra.yaml":  projecttest.MinimalSchema,
		"schemas/apple.yaml":  projecttest.MinimalSchema,
		"schemas/middle.yaml": projecttest.MinimalSchema,
	})
	cfg, err := project.Load(dir)
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
	projecttest.WriteProject(t, dir, map[string]string{
		"bases/local.yaml": projecttest.LocalBase(map[string]string{"notes": "path: notes\nchecks:\n  - kind: markdown_writing_tells\n"}),
	})
	cfg, err := project.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	notes, _ := cfg.Collection("notes")
	if len(notes.Checks) != 1 || notes.Checks[0].Kind != checks.CheckMarkdownWritingTells {
		t.Fatalf("expected one markdown_writing_tells check, got %+v", notes.Checks)
	}
}
