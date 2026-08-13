package cmd_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

const fixNotesConfig = `path: notes
checks:
  - kind: markdown_requires_h1
`

func setupFixRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": fixNotesConfig}),
	})
	chdir(t, dir)
	return dir
}

func setupFixRepoWith(t *testing.T, notesConfig string) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": notesConfig}),
	})
	chdir(t, dir)
	return dir
}

func TestFix_textForbidsFix_rewritesOnlyMatch(t *testing.T) {
	dir := setupFixRepoWith(t, `path: notes
checks:
  - kind: text_forbids
    target: first-line
    pattern: '\.(\s*)$'
    fix: '$1'
`)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nt: 1\n---\n# Title.\nkeep this.\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	// First line loses its period; the later "keep this." line is untouched.
	want := "---\nt: 1\n---\n# Title\nkeep this.\n"
	if string(got) != want {
		t.Errorf("after fix:\n got: %q\nwant: %q", got, want)
	}
}

func TestFix_textForbidsFix_badTemplateFails(t *testing.T) {
	dir := setupFixRepoWith(t, `path: notes
checks:
  - kind: text_forbids
    pattern: TODO
    fix: TODO-DONE
`)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nt: 1\n---\nhas TODO here\n")

	_, stderr, err := runRoot(t, "fix", "notes/doc")
	if err == nil {
		t.Fatal("expected fix to fail on a template that does not resolve the violation")
	}
	if !strings.Contains(stderr, "fix did not resolve the violation") {
		t.Errorf("expected re-check failure message, got stderr: %q", stderr)
	}
	got, _ := os.ReadFile(p)
	if !strings.Contains(string(got), "has TODO here") {
		t.Errorf("file must be untouched on failure, got %q", got)
	}
}

func TestFix_textForbidsWithoutFix_preservesBody(t *testing.T) {
	dir := setupFixRepoWith(t, `path: notes
checks:
  - kind: text_forbids
    pattern: TODO
`)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\n# Body TODO\nkeep\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	// Frontmatter is still canonicalized; the body (TODO and all) is verbatim.
	want := "---\napple: 2\nzebra: 1\n---\n# Body TODO\nkeep\n"
	if string(got) != want {
		t.Errorf("after fix:\n got: %q\nwant: %q", got, want)
	}
}

func TestFix_normalizesAndPreservesBody(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\n# Body\nverbatim\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	want := "---\napple: 2\nzebra: 1\n---\n# Body\nverbatim\n"
	if string(got) != want {
		t.Errorf("after fix:\n got: %q\nwant: %q", got, want)
	}
}

func TestFix_isIdempotent(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\n# Body\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("first fix: %v", err)
	}
	first, _ := os.ReadFile(p)
	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("second fix: %v", err)
	}
	second, _ := os.ReadFile(p)
	if string(first) != string(second) {
		t.Errorf("fix not idempotent:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestFix_checkFlag_reportsWithoutWriting(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	original := "---\nzebra: 1\napple: 2\n---\n# Body\n"
	mustWrite(t, p, original)

	stdout, _, err := runRoot(t, "fix", "--check", "notes/doc")
	if err == nil {
		t.Fatalf("expected --check to exit non-zero when a change is pending")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(stdout, "doc.md") {
		t.Errorf("expected the would-change path in stdout, got: %q", stdout)
	}
	got, _ := os.ReadFile(p)
	if string(got) != original {
		t.Errorf("--check modified the file: %q", got)
	}
}

func TestFix_projectFlagSelectsProject(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{"notes": fixNotesConfig}),
	})
	outside := t.TempDir()
	chdir(t, outside)
	p := filepath.Join(dir, "notes", "doc.md")
	mustWrite(t, p, "---\nzebra: 1\napple: 2\n---\n# Body\n")

	if _, _, err := runRoot(t, "fix", "--project", dir, "notes/doc"); err != nil {
		t.Fatalf("fix --project: %v", err)
	}
	got, _ := os.ReadFile(p)
	want := "---\napple: 2\nzebra: 1\n---\n# Body\n"
	if string(got) != want {
		t.Errorf("after fix --project:\n got: %q\nwant: %q", got, want)
	}
}

func TestFix_checkFlag_cleanExitsZero(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\napple: 2\nzebra: 1\n---\n# Body\n")

	if _, _, err := runRoot(t, "fix", "--check", "notes/doc"); err != nil {
		t.Fatalf("--check on canonical file should exit 0: %v", err)
	}
}

// D3 guardrail: fix normalizes but never injects a value for a missing
// required key.
func TestFix_neverInjectsMissingKeys(t *testing.T) {
	dir := setupFixRepo(t)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\ntitle: Dune\n---\n# Dune\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), "year") {
		t.Errorf("fix injected a missing key; content: %q", got)
	}
}

func TestFix_wholeProjectAndReportsChangedOnly(t *testing.T) {
	dir := setupFixRepo(t)
	dirty := filepath.Join(dir, "notes/dirty.md")
	clean := filepath.Join(dir, "notes/clean.md")
	mustWrite(t, dirty, "---\nz: 1\na: 2\n---\n# Body\n")
	mustWrite(t, clean, "---\na: 2\nz: 1\n---\n# Body\n")

	stdout, _, err := runRoot(t, "fix")
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	if !strings.Contains(stdout, "dirty.md") {
		t.Errorf("expected changed file listed, got: %q", stdout)
	}
	if strings.Contains(stdout, "clean.md") {
		t.Errorf("did not expect unchanged file listed, got: %q", stdout)
	}
}

func TestFix_rootRunChecksDelegatedChildWhenFixFileNearest(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        fix: file_nearest
`,
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{
			"notes": "path: notes\nchecks:\n  - kind: markdown_requires_h1\n",
		}),
	})
	chdir(t, dir)
	p := filepath.Join(child, "notes", "dirty.md")
	original := "---\nz: 1\na: 2\n---\n# Body\n"
	mustWrite(t, p, original)

	stdout, _, err := runRoot(t, "fix", "--check")
	if err == nil {
		t.Fatalf("expected delegated child --check to report pending fix")
	}
	if !strings.Contains(stdout, "dirty.md") {
		t.Errorf("expected delegated child path in stdout, got: %q", stdout)
	}
	got, _ := os.ReadFile(p)
	if string(got) != original {
		t.Errorf("--check modified child file: %q", got)
	}
}

func TestFix_rootRunRewritesDelegatedChildWhenFixFileNearest(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        fix: file_nearest
`,
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{
			"notes": "path: notes\nchecks:\n  - kind: markdown_requires_h1\n",
		}),
	})
	chdir(t, dir)
	p := filepath.Join(child, "notes", "dirty.md")
	mustWrite(t, p, "---\nz: 1\na: 2\n---\n# Body\n")

	stdout, _, err := runRoot(t, "fix")
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	if !strings.Contains(stdout, "dirty.md") {
		t.Errorf("expected delegated child path in stdout, got: %q", stdout)
	}
	got, _ := os.ReadFile(p)
	want := "---\na: 2\nz: 1\n---\n# Body\n"
	if string(got) != want {
		t.Errorf("after delegated fix:\n got: %q\nwant: %q", got, want)
	}
}

func TestFix_fileNearestSkipsParentRewriteInsideDelegatedSubtree(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        fix: file_nearest
`,
		"bases/local.yaml": baseLocal(map[string]string{
			"rootNotes": `path: ongoing/blog/notes
checks:
  - kind: text_forbids
    target: first-line
    pattern: child
    fix: parent
`,
		}),
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{
			"notes": `path: notes
checks:
  - kind: text_forbids
    target: first-line
    pattern: child
    fix: kid
`,
		}),
	})
	chdir(t, dir)
	p := filepath.Join(child, "notes", "story.md")
	mustWrite(t, p, "---\ntitle: Story\n---\n# child story\n")

	if _, _, err := runRoot(t, "fix"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), "# parent story") {
		t.Fatalf("parent fix rewrote delegated child file:\n%s", got)
	}
	if !strings.Contains(string(got), "# kid story") {
		t.Fatalf("expected child fix to rewrite delegated file, got:\n%s", got)
	}
}

func TestFix_rootNearestPreservesParentRewriteInsideDelegatedSubtree(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        collections: file_nearest
        fix: root_nearest
`,
		"bases/local.yaml": baseLocal(map[string]string{
			"rootNotes": `path: ongoing/blog/notes
checks:
  - kind: text_forbids
    target: first-line
    pattern: child
    fix: parent
`,
		}),
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{
			"notes": `path: notes
checks:
  - kind: text_forbids
    target: first-line
    pattern: child
    fix: kid
`,
		}),
	})
	chdir(t, dir)
	p := filepath.Join(child, "notes", "story.md")
	mustWrite(t, p, "---\ntitle: Story\n---\n# child story\n")

	if _, _, err := runRoot(t, "fix"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	if !strings.Contains(string(got), "# parent story") {
		t.Fatalf("expected parent fix under root_nearest, got:\n%s", got)
	}
}

func TestFix_disableNestedConfigSkipsDelegatedChildFix(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        fix: file_nearest
`,
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{
			"notes": "path: notes\nchecks:\n  - kind: markdown_requires_h1\n",
		}),
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(child, "notes", "dirty.md"), "---\nz: 1\na: 2\n---\n# Body\n")

	if _, _, err := runRoot(t, "fix", "--check", "--disable-nested-config"); err != nil {
		t.Fatalf("disabled nested config should skip child fix, got: %v", err)
	}
}

func TestFix_configFlagDisablesNestedChildFix(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        fix: file_nearest
`,
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": baseLocal(map[string]string{
			"notes": "path: notes\nchecks:\n  - kind: markdown_requires_h1\n",
		}),
	})
	chdir(t, dir)
	mustWrite(t, filepath.Join(child, "notes", "dirty.md"), "---\nz: 1\na: 2\n---\n# Body\n")

	if _, _, err := runRoot(t, "fix", "--check", "--config", filepath.Join(dir, ".katalyst")); err != nil {
		t.Fatalf("--config should disable nested child fix, got: %v", err)
	}
}

func TestFix_rejectsComposeAuthority(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml": `nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        fix: compose
`,
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	child := filepath.Join(dir, "ongoing", "blog")
	writeProject(t, child, map[string]string{
		"bases/local.yaml": "type: filesystem\nroot: .\ncollections: {}\n",
	})
	chdir(t, dir)

	_, _, err := runRoot(t, "fix")
	if err == nil {
		t.Fatalf("expected fix: compose to be rejected")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Fatalf("expected usage exit for fix: compose, got %v", err)
	}
}

// --- SQLite content-column coverage -----------------------------------------
//
// fix gates on whether a collection exposes a text body, not on the backend
// name: a sqlite collection that maps a content column has one, a collection
// of attributes alone does not.

const sqliteFixBase = `type: sqlite
path: content.sqlite
collections:
  notes:
    table: notes
    id: slug
    attributes:
      title: title
    content:
      kind: markdown
      column: body
    checks:
      - kind: text_forbids
        target: first-line
        pattern: '\.(\s*)$'
        fix: '$1'
`

const sqliteFixBaseBadTemplate = `type: sqlite
path: content.sqlite
collections:
  notes:
    table: notes
    id: slug
    attributes:
      title: title
    content:
      kind: markdown
      column: body
    checks:
      - kind: text_forbids
        pattern: TODO
        fix: TODO-DONE
`

const sqliteFixBaseNoContent = `type: sqlite
path: content.sqlite
collections:
  notes:
    table: notes
    id: slug
    attributes:
      title: title
    checks:
      - kind: object_required_field
        field: title
`

// setupSQLiteFixRepo scaffolds a one-row sqlite project whose notes collection
// is declared by baseYAML, seeding the body column with body.
func setupSQLiteFixRepo(t *testing.T, baseYAML, body string) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{"bases/db.yaml": baseYAML})

	db, err := sql.Open("sqlite", filepath.Join(dir, "content.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE notes (slug TEXT PRIMARY KEY, title TEXT, body TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO notes (slug, title, body) VALUES ('dune', 'Dune', ?)`, body); err != nil {
		t.Fatalf("seed row: %v", err)
	}
	chdir(t, dir)
	return dir
}

func TestFix_sqliteContentColumn_appliesTextFix(t *testing.T) {
	dir := setupSQLiteFixRepo(t, sqliteFixBase, "# Dune.\nkeep this.\n")

	if _, _, err := runRoot(t, "fix", "notes/dune"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got := sqliteScalar(t, dir, `SELECT body FROM notes WHERE slug = 'dune'`)
	// The first line loses its period; the later "keep this." line is untouched.
	if want := "# Dune\nkeep this.\n"; got != want {
		t.Errorf("body after fix = %#v, want %q", got, want)
	}
}

func TestFix_sqliteContentColumn_leavesAttributesAlone(t *testing.T) {
	dir := setupSQLiteFixRepo(t, sqliteFixBase, "# Dune.\n")

	if _, _, err := runRoot(t, "fix", "notes/dune"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	if got := sqliteScalar(t, dir, `SELECT title FROM notes WHERE slug = 'dune'`); got != "Dune" {
		t.Errorf("fix must not rewrite attribute columns, title = %#v", got)
	}
}

func TestFix_sqliteCheckFlag_reportsWithoutWriting(t *testing.T) {
	dir := setupSQLiteFixRepo(t, sqliteFixBase, "# Dune.\n")

	stdout, _, err := runRoot(t, "fix", "--check", "notes/dune")
	if err == nil {
		t.Fatal("expected --check to exit non-zero when an item would change")
	}
	if !strings.Contains(stdout, "notes/dune") {
		t.Errorf("expected the changed item reported, got stdout %q", stdout)
	}
	if got := sqliteScalar(t, dir, `SELECT body FROM notes WHERE slug = 'dune'`); got != "# Dune.\n" {
		t.Errorf("--check must not write, body = %#v", got)
	}
}

func TestFix_sqliteWithoutContentColumn_requiresTextContent(t *testing.T) {
	setupSQLiteFixRepo(t, sqliteFixBaseNoContent, "")

	_, stderr, err := runRoot(t, "fix", "notes/dune")
	if err == nil {
		t.Fatal("expected fix to refuse a collection that maps no text content")
	}
	if !strings.Contains(stderr, "requires a text content mapping") {
		t.Errorf("expected the capability message, got stderr %q", stderr)
	}
	// The item label embeds the db filename, so assert on the old message
	// rather than the bare word: the gate is a capability, not a backend name.
	if strings.Contains(stderr, "not supported for sqlite") {
		t.Errorf("the message must name the missing capability, not the backend: %q", stderr)
	}
}

func TestFix_sqliteBadTemplateFails(t *testing.T) {
	dir := setupSQLiteFixRepo(t, sqliteFixBaseBadTemplate, "has TODO here\n")

	_, stderr, err := runRoot(t, "fix", "notes/dune")
	if err == nil {
		t.Fatal("expected fix to fail on a template that does not resolve the violation")
	}
	if !strings.Contains(stderr, "fix did not resolve the violation") {
		t.Errorf("expected re-check failure message, got stderr %q", stderr)
	}
	if got := sqliteScalar(t, dir, `SELECT body FROM notes WHERE slug = 'dune'`); got != "has TODO here\n" {
		t.Errorf("row must be untouched on failure, body = %#v", got)
	}
}

// TestFix_filesystemOutput_unchangedByBackendRouting pins the filesystem result
// byte-for-byte across both halves of fix (a text fix and a frontmatter
// reorder), so routing fix through the project layer cannot silently change it.
func TestFix_filesystemOutput_unchangedByBackendRouting(t *testing.T) {
	dir := setupFixRepoWith(t, `path: notes
checks:
  - kind: text_forbids
    target: first-line
    pattern: '\.(\s*)$'
    fix: '$1'
`)
	p := filepath.Join(dir, "notes/doc.md")
	mustWrite(t, p, "---\nzeta: 1\nalpha: 2\n---\n# Title.\nkeep this.\n")

	if _, _, err := runRoot(t, "fix", "notes/doc"); err != nil {
		t.Fatalf("fix: %v", err)
	}
	got, _ := os.ReadFile(p)
	want := "---\nalpha: 2\nzeta: 1\n---\n# Title\nkeep this.\n"
	if string(got) != want {
		t.Errorf("filesystem fix output changed:\n got: %q\nwant: %q", got, want)
	}
}
