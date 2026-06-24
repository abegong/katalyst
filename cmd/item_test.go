package cmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupItemRepo creates a repo with a single `notes` collection backed by
// the book object schema (title+year required), and chdirs in.
func setupItemRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeProject(t, dir, map[string]string{
		"config.yaml":              schemaFormatJSON,
		"schemas/book.json":        bookSchemaFixture,
		"schemas/strict-book.json": strictBookSchemaFixture,
		"storage/local.yaml":       storageLocal(map[string]string{"notes": objectNotesConfig}),
	})
	chdir(t, dir)
	return dir
}

func TestItemAdd_writesFrontmatterAndEmptyBody(t *testing.T) {
	dir := setupItemRepo(t)
	if _, _, err := runRoot(t, "item", "add", "notes/dune", "title=Dune", "year=1965"); err != nil {
		t.Fatalf("item add: %v", err)
	}
	p := filepath.Join(dir, "notes/dune.md")
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("expected file written: %v", err)
	}
	// YAML-scalar typing: year is an integer, not a quoted string.
	if !strings.Contains(string(got), "year: 1965") {
		t.Errorf("expected integer year, got:\n%s", got)
	}
	// Empty body: nothing after the closing fence.
	if !strings.HasSuffix(string(got), "---\n") {
		t.Errorf("expected empty body after frontmatter, got:\n%s", got)
	}
}

func TestItemAdd_refusesExisting_exit2(t *testing.T) {
	dir := setupItemRepo(t)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: x\nyear: 1\n---\n")
	_, _, err := runRoot(t, "item", "add", "notes/dune", "title=Changed", "year=2")
	if err == nil {
		t.Fatalf("expected refuse-overwrite error")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestItemAdd_validationFailureWritesNothing_exit1(t *testing.T) {
	dir := setupItemRepo(t)
	strict := filepath.Join(dir, ".katalyst/schemas/strict-book.json")
	_, _, err := runRoot(t, "item", "add", "--schema", strict, "notes/dune", "title=Dune", "year=1965")
	if err == nil {
		t.Fatalf("expected strict validation failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	if !strings.Contains(err.Error(), "isbn") {
		t.Errorf("expected isbn in error, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes/dune.md")); !os.IsNotExist(err) {
		t.Errorf("expected nothing written on validation failure")
	}
}

func TestItemAdd_noValidateBypasses(t *testing.T) {
	dir := setupItemRepo(t)
	strict := filepath.Join(dir, ".katalyst/schemas/strict-book.json")
	if _, _, err := runRoot(t, "item", "add", "--schema", strict, "--no-validate", "notes/dune", "title=Dune"); err != nil {
		t.Fatalf("expected --no-validate to succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes/dune.md")); err != nil {
		t.Errorf("expected file written: %v", err)
	}
}

func TestItemGet_defaultPrintsFrontmatterAndBody(t *testing.T) {
	dir := setupItemRepo(t)
	content := "---\ntitle: Dune\nyear: 1965\n---\n# Dune\nbody\n"
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), content)

	stdout, _, err := runRoot(t, "item", "get", "notes/dune")
	if err != nil {
		t.Fatalf("item get: %v", err)
	}
	if stdout != content {
		t.Errorf("default get mismatch:\n got: %q\nwant: %q", stdout, content)
	}
}

func TestItemGet_frontmatterAndBodyFlags(t *testing.T) {
	dir := setupItemRepo(t)
	mustWrite(t, filepath.Join(dir, "notes/dune.md"), "---\ntitle: Dune\nyear: 1965\n---\n# Dune\nbody\n")

	fm, _, err := runRoot(t, "item", "get", "--frontmatter", "notes/dune")
	if err != nil {
		t.Fatalf("get --frontmatter: %v", err)
	}
	snapshot(t, "item/get-frontmatter.txt", fm)

	body, _, err := runRoot(t, "item", "get", "--body", "notes/dune")
	if err != nil {
		t.Fatalf("get --body: %v", err)
	}
	snapshot(t, "item/get-body.txt", body)
}

func TestItemGet_unknownItem_exit2(t *testing.T) {
	setupItemRepo(t)
	_, _, err := runRoot(t, "item", "get", "notes/ghost")
	if err == nil {
		t.Fatalf("expected error for unknown item")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestItemUpdate_mergesKeysBodyUntouched(t *testing.T) {
	dir := setupItemRepo(t)
	p := filepath.Join(dir, "notes/dune.md")
	mustWrite(t, p, "---\ntitle: Dune\nyear: 1965\n---\n# Dune\noriginal body\n")

	if _, _, err := runRoot(t, "item", "update", "notes/dune", "year=1969"); err != nil {
		t.Fatalf("item update: %v", err)
	}
	got, _ := os.ReadFile(p)
	if !strings.Contains(string(got), "year: 1969") {
		t.Errorf("expected updated year, got:\n%s", got)
	}
	if !strings.Contains(string(got), "original body") {
		t.Errorf("expected body untouched, got:\n%s", got)
	}
}

func TestItemUpdate_strictFailureLeavesFileUnchanged(t *testing.T) {
	dir := setupItemRepo(t)
	p := filepath.Join(dir, "notes/dune.md")
	before := "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n"
	mustWrite(t, p, before)

	strict := filepath.Join(dir, ".katalyst/schemas/strict-book.json")
	_, _, err := runRoot(t, "item", "update", "--schema", strict, "notes/dune", "title=Changed")
	if err == nil {
		t.Fatalf("expected strict update failure")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	after, _ := os.ReadFile(p)
	if string(after) != before {
		t.Errorf("file modified despite validation failure:\n%s", after)
	}
}

func TestItemDelete_removesOneAndMany(t *testing.T) {
	dir := setupItemRepo(t)
	a := filepath.Join(dir, "notes/a.md")
	b := filepath.Join(dir, "notes/b.md")
	mustWrite(t, a, "---\ntitle: A\nyear: 1\n---\n")
	mustWrite(t, b, "---\ntitle: B\nyear: 2\n---\n")

	if _, _, err := runRoot(t, "item", "delete", "notes/a", "notes/b"); err != nil {
		t.Fatalf("item delete: %v", err)
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected %s removed", p)
		}
	}
}

func TestItemDelete_missing_exit2(t *testing.T) {
	setupItemRepo(t)
	_, _, err := runRoot(t, "item", "delete", "notes/ghost")
	if err == nil {
		t.Fatalf("expected error for missing item")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestItemList_showsIdsAndStatus(t *testing.T) {
	dir := setupItemRepo(t)
	mustWrite(t, filepath.Join(dir, "notes/good.md"), "---\ntitle: Good\nyear: 1\n---\n# Good\n")
	mustWrite(t, filepath.Join(dir, "notes/bad.md"), "---\ntitle: Bad\n---\n# Bad\n") // missing year

	stdout, _, err := runRoot(t, "item", "list", "notes")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	// The fixture pins the id + status table (good: ok, bad: missing-year error).
	snapshot(t, "item/list.txt", stdout)
}

func TestItemList_wrongDepth_exit2(t *testing.T) {
	setupItemRepo(t)
	_, _, err := runRoot(t, "item", "list", "notes/dune")
	if err == nil {
		t.Fatalf("expected wrong-depth usage error")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

// seedBooks writes three valid items into the notes collection.
func seedBooks(t *testing.T, dir string) {
	t.Helper()
	mustWrite(t, filepath.Join(dir, "notes/dune.md"),
		"---\ntitle: Dune\nyear: 1965\nstatus: published\ntags: [sci-fi]\n---\n# Dune\n\nSpice TODO.\n")
	mustWrite(t, filepath.Join(dir, "notes/hobbit.md"),
		"---\ntitle: The Hobbit\nyear: 1937\nstatus: published\n---\n# The Hobbit\n")
	mustWrite(t, filepath.Join(dir, "notes/wip.md"),
		"---\ntitle: WIP\nyear: 2025\nstatus: draft\n---\n# WIP\n")
}

// listIDs returns the first column (item id) of each non-empty output line.
func listIDs(out string) []string {
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		got = append(got, strings.Fields(line)[0])
	}
	return got
}

func TestItemList_filter(t *testing.T) {
	dir := setupItemRepo(t)
	seedBooks(t, dir)

	stdout, _, err := runRoot(t, "item", "list", "notes", "--filter", "year>=1965")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	if got := strings.Join(listIDs(stdout), ","); got != "dune,wip" {
		t.Errorf("filter year>=1965 = %q, want dune,wip", got)
	}

	// Two filters are ANDed.
	stdout, _, _ = runRoot(t, "item", "list", "notes", "--filter", "status=published", "--filter", "year>=1950")
	if got := strings.Join(listIDs(stdout), ","); got != "dune" {
		t.Errorf("ANDed filters = %q, want dune", got)
	}

	// Membership and absence.
	stdout, _, _ = runRoot(t, "item", "list", "notes", "--filter", "tags=sci-fi")
	if got := strings.Join(listIDs(stdout), ","); got != "dune" {
		t.Errorf("tags membership = %q, want dune", got)
	}
	stdout, _, _ = runRoot(t, "item", "list", "notes", "--filter", "!tags")
	if got := strings.Join(listIDs(stdout), ","); got != "hobbit,wip" {
		t.Errorf("!tags = %q, want hobbit,wip", got)
	}
}

func TestItemList_grep(t *testing.T) {
	dir := setupItemRepo(t)
	seedBooks(t, dir)

	// Body grep: only dune's body has TODO.
	stdout, _, err := runRoot(t, "item", "list", "notes", "--grep", "TODO", "--grep-in", "body")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	if got := strings.Join(listIDs(stdout), ","); got != "dune" {
		t.Errorf("grep body TODO = %q, want dune", got)
	}

	// Case-insensitive grep on frontmatter status.
	stdout, _, _ = runRoot(t, "item", "list", "notes", "--grep", "DRAFT", "-i", "--grep-in", "frontmatter")
	if got := strings.Join(listIDs(stdout), ","); got != "wip" {
		t.Errorf("grep -i frontmatter = %q, want wip", got)
	}
}

func TestItemList_sortAndLimit(t *testing.T) {
	dir := setupItemRepo(t)
	seedBooks(t, dir)

	stdout, _, err := runRoot(t, "item", "list", "notes", "--sort", "-year", "--limit", "2")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	if got := strings.Join(listIDs(stdout), ","); got != "wip,dune" {
		t.Errorf("sort -year limit 2 = %q, want wip,dune", got)
	}

	// Skip after sort.
	stdout, _, _ = runRoot(t, "item", "list", "notes", "--sort", "-year", "--skip", "1")
	if got := strings.Join(listIDs(stdout), ","); got != "dune,hobbit" {
		t.Errorf("sort -year skip 1 = %q, want dune,hobbit", got)
	}
}

func TestItemList_emptyResult_exit0(t *testing.T) {
	dir := setupItemRepo(t)
	seedBooks(t, dir)
	stdout, _, err := runRoot(t, "item", "list", "notes", "--filter", "year=9999")
	if err != nil {
		t.Fatalf("empty result should be exit 0, got: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty output, got: %q", stdout)
	}
}

func TestItemList_badQuery_exit2(t *testing.T) {
	dir := setupItemRepo(t)
	seedBooks(t, dir)
	cases := [][]string{
		{"item", "list", "notes", "--filter", "=oops"},
		{"item", "list", "notes", "--grep", "("},
		{"item", "list", "notes", "--limit", "-1"},
		{"item", "list", "notes", "--grep-in", "nowhere", "--grep", "x"},
		{"item", "list", "notes", "--sort", "-"},
	}
	for _, args := range cases {
		_, _, err := runRoot(t, args...)
		var coded interface{ Code() int }
		if !errors.As(err, &coded) || coded.Code() != 2 {
			t.Errorf("%v: expected exit 2, got: %v", args, err)
		}
	}
}

func TestItemList_typeMismatch_flagAndConfig(t *testing.T) {
	dir := setupItemRepo(t)
	seedBooks(t, dir)
	// A non-numeric year breaks the schema (so it's an "error" item) but
	// we only care that filtering tolerates or rejects the mismatch.
	mustWrite(t, filepath.Join(dir, "notes/odd.md"), "---\ntitle: Odd\nyear: \"sometime\"\n---\n# Odd\n")

	// Default skip: the odd item is silently excluded, exit 0.
	if _, _, err := runRoot(t, "item", "list", "notes", "--filter", "year>=1900"); err != nil {
		t.Fatalf("default skip should not error: %v", err)
	}

	// --on-type-mismatch error → exit 2.
	_, _, err := runRoot(t, "item", "list", "notes", "--filter", "year>=1900", "--on-type-mismatch", "error")
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit 2 under error mode, got: %v", err)
	}
}

func TestItemList_sortMissing_flag(t *testing.T) {
	dir := setupItemRepo(t)
	mustWrite(t, filepath.Join(dir, "notes/has.md"), "---\ntitle: Has\nyear: 2000\n---\n# Has\n")
	mustWrite(t, filepath.Join(dir, "notes/none.md"), "---\ntitle: None\nyear: 1\nrank: 5\n---\n# None\n")
	// has.md lacks rank; sort by rank ascending.
	stdout, _, err := runRoot(t, "item", "list", "notes", "--sort", "rank", "--sort-missing", "last")
	if err != nil {
		t.Fatalf("item list: %v", err)
	}
	if got := strings.Join(listIDs(stdout), ","); got != "none,has" {
		t.Errorf("sort-missing last = %q, want none,has", got)
	}
	stdout, _, _ = runRoot(t, "item", "list", "notes", "--sort", "rank", "--sort-missing", "lowest")
	if got := strings.Join(listIDs(stdout), ","); got != "has,none" {
		t.Errorf("sort-missing lowest = %q, want has,none", got)
	}
}
