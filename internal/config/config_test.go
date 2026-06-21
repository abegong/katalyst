package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katabase-ai/katalyst/internal/config"
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
		"collections/books.yaml": `path: notes/books
schema: book
`,
		"collections/people.yaml": `path: notes/people
pattern: "*.markdown"
schema: person
`,
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

	// Collections are sorted by name: books, people.
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
		"schemas/book.yaml":      minimalSchema,
		"collections/notes.yaml": "schema: book\n",
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

func TestLoad_noConfigFile_usesConventionDefaults(t *testing.T) {
	// A project with a .katalyst/ dir but no config.yaml loads via the
	// default convention + yaml discovery.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":      minimalSchema,
		"collections/notes.yaml": "schema: book\n",
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

func TestLoad_rejectsUnknownSchemaInCollection(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"collections/notes.yaml": `path: notes
schema: nonexistent
`,
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
		"collections/notes.yaml": "path: notes\n",
	})
	_, err := config.Load(dir)
	if err == nil {
		t.Fatalf("expected error for collection with no checks")
	}
	if !strings.Contains(err.Error(), "no checks") {
		t.Errorf("expected 'no checks' message, got: %v", err)
	}
}

func TestLoad_parsesChecks(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml": minimalSchema,
		"collections/notes.yaml": `path: notes
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
`,
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
		"collections/notes.yaml": `path: notes
checks:
  - kind: not-real
`,
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
		"collections/notes.yaml": `path: notes
checks:
  - kind: object
`,
	})
	_, err := config.Load(dir)
	if err == nil {
		t.Fatalf("expected error for missing object schema")
	}
	if !strings.Contains(err.Error(), "requires") {
		t.Fatalf("expected malformed payload error, got: %v", err)
	}
}

func TestLoad_explicitDiscovery_readsDefs(t *testing.T) {
	// In explicit mode, the directory scan is ignored and the defs maps
	// in config.yaml are authoritative.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"my-schemas/book.yaml": minimalSchema,
		"config.yaml": `schemas:
  discovery: explicit
  defs:
    book: ./.katalyst/my-schemas/book.yaml
collections:
  discovery: explicit
  defs:
    notes:
      path: notes
      schema: book
`,
		// A stray file in the convention dir must be ignored.
		"schemas/ignored.yaml": minimalSchema,
	})
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg.Schemas["ignored"]; ok {
		t.Errorf("explicit discovery must ignore the schemas/ dir scan")
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
		"config.yaml": "schemas:\n  discovery: explicit\n",
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "defs") {
		t.Fatalf("expected explicit-requires-defs error, got: %v", err)
	}
}

func TestLoad_formatJSON_scansJSONFiles(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.json":      `{"type":"object"}`,
		"config.yaml":            "schemas:\n  format: json\n",
		"collections/notes.yaml": "schema: book\n",
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
	// Schemas explicit + json; collections convention + yaml.
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.json": `{"type":"object"}`,
		"config.yaml": `schemas:
  discovery: explicit
  format: json
  defs:
    book: ./.katalyst/schemas/book.json
`,
		"collections/notes.yaml": "schema: book\n",
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
		"schemas/book.yaml":      minimalSchema,
		"collections/notes.yaml": "schema: book\n",
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
		"collections/notes.yaml": "schema: book\n",
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
		"collections/notes.yaml": `schema: book
query:
  filterTypeMismatch: error
`,
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
		"collections/notes.yaml": `schema: book
query:
  filterTypeMismatch: bogus
`,
	})
	_, err := config.Load(dir)
	if err == nil || !strings.Contains(err.Error(), "filterTypeMismatch") {
		t.Fatalf("expected filterTypeMismatch validation error, got: %v", err)
	}
}

func TestCollection_unknownReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"schemas/book.yaml":      minimalSchema,
		"collections/notes.yaml": "schema: book\n",
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
