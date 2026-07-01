package filesystem_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/checktest"
	"github.com/abegong/katalyst/internal/checks/filesystem"
)

func TestNameMatchesFieldRun_detectsMismatch(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: dune-messiah\n---\n# Dune Messiah\n")
	violations := filesystem.NameMatchesField{Field: "slug"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune-messiah"},
	})
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Line != 2 {
		t.Fatalf("expected line 2 for slug key, got %d", violations[0].Line)
	}
}

func TestNameMatchesFieldRun_slugifyMatchesTitle(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ntitle: My First Note\n---\n# My First Note\n")
	violations := filesystem.NameMatchesField{Field: "title", Transform: "slugify"}.Run(checks.Context{
		FilePath: "notes/my-first-note.md",
		Doc:      doc,
		Meta:     map[string]any{"title": "My First Note"},
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
		doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
		violations := filesystem.NameCase{Style: tc.style}.Run(checks.Context{
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
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := filesystem.NameCase{Style: "kebab", Target: filesystem.TargetParentDir}.Run(checks.Context{
		FilePath: "notes/My Folder/ok-name.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "parent directory") {
		t.Fatalf("expected parent-dir kebab violation, got %v", violations)
	}
}

func TestNameCaseRun_pathSegmentsInclusive(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	// A bad mid-path segment AND a bad basename should both flag.
	violations := filesystem.NameCase{Style: "kebab", Target: filesystem.TargetPathSegments}.Run(checks.Context{
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
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := filesystem.NameAffix{Prefix: "book-", Suffix: "-draft"}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "x"},
	})
	if len(violations) != 2 {
		t.Fatalf("expected 2 affix violations, got %d: %v", len(violations), violations)
	}
}

func TestPathCharsetRun_denyReproducesNoSpaces(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := filesystem.PathCharset{Deny: []string{" "}}.Run(checks.Context{
		FilePath: "notes/my dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "must not contain") {
		t.Fatalf("expected deny-space violation, got %v", violations)
	}
}

func TestPathCharsetRun_allowWhitelist(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	violations := filesystem.PathCharset{Allow: []string{"abcdefghijklmnopqrstuvwxyz-."}}.Run(checks.Context{
		FilePath:       "/proj/notes/My_File.md",
		CollectionRoot: "/proj/notes",
		Doc:            doc,
		Meta:           map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "disallowed character") {
		t.Fatalf("expected allow-whitelist violation, got %v", violations)
	}
}

func TestFilesystemExtensionInRun_detectsDisallowed(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: dune\n---\n# Dune\n")
	violations := filesystem.FilesystemExtensionIn{Values: []string{".md"}}.Run(checks.Context{
		FilePath: "notes/dune.txt",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "extension") {
		t.Fatalf("expected extension violation, got %v", violations)
	}
}

func TestNameRegexRun_anchored(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	re := regexp.MustCompile(checks.AnchoredPattern(`[0-9]{4}-[a-z-]+`))
	// "2024-dune" matches fully; "x-2024-dune" must NOT (anchored).
	pass := filesystem.NameRegex{Re: re, Pattern: "p"}.Run(checks.Context{
		FilePath: "notes/2024-dune.md", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(pass) != 0 {
		t.Fatalf("expected pass for anchored match, got %v", pass)
	}
	fail := filesystem.NameRegex{Re: re, Pattern: "p"}.Run(checks.Context{
		FilePath: "notes/x-2024-dune.md", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(fail) != 1 {
		t.Fatalf("expected anchored mismatch to fail, got %v", fail)
	}
}

func TestNameLengthRun_bounds(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	max := 5
	violations := filesystem.NameLength{Max: &max}.Run(checks.Context{
		FilePath: "notes/toolongname.md", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "at most") {
		t.Fatalf("expected max-length violation, got %v", violations)
	}
}

func TestPathDepthRun_flat(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\nslug: x\n---\n# x\n")
	max := 0
	// A nested file violates max depth 0; a root file does not.
	nested := filesystem.PathDepth{Max: &max}.Run(checks.Context{
		FilePath: "/proj/notes/sub/dune.md", CollectionRoot: "/proj/notes", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(nested) != 1 || !strings.Contains(nested[0].Message, "depth") {
		t.Fatalf("expected depth violation for nested file, got %v", nested)
	}
	flat := filesystem.PathDepth{Max: &max}.Run(checks.Context{
		FilePath: "/proj/notes/dune.md", CollectionRoot: "/proj/notes", Doc: doc, Meta: map[string]any{"slug": "x"},
	})
	if len(flat) != 0 {
		t.Fatalf("expected no violation for flat file, got %v", flat)
	}
}

func TestParentDirMatchesFieldRun_detectsMismatch(t *testing.T) {
	doc := checktest.MustParseDoc(t, "---\ncategory: recipes\n---\n# x\n")
	violations := filesystem.ParentDirMatchesField{Field: "category"}.Run(checks.Context{
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
	doc := checktest.MustParseDoc(t, "---\ncover: cover.png\nextras:\n  - a.png\n  - b.png\n---\n# x\n")
	violations := filesystem.ReferencedFilesExist{Fields: []string{"cover", "extras"}}.Run(checks.Context{
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
	doc := checktest.MustParseDoc(t, "---\nslug: dune\n---\n# Dune\n")
	violations := filesystem.FilesystemParentDirIn{Values: []string{"books"}}.Run(checks.Context{
		FilePath: "notes/dune.md",
		Doc:      doc,
		Meta:     map[string]any{"slug": "dune"},
	})
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "parent directory") {
		t.Fatalf("expected parent-dir violation, got %v", violations)
	}
}

func TestUniqueFilename_flagsCollision(t *testing.T) {
	ctx := checks.CollectionContext{Items: []checks.ItemContext{
		{FilePath: "notes/a/dune.md"},
		{FilePath: "notes/b/dune.md"},
		{FilePath: "notes/c/other.md"},
	}}
	violations := filesystem.UniqueFilename{}.RunCollection(ctx)
	if len(violations) != 1 {
		t.Fatalf("expected 1 collision violation, got %d: %v", len(violations), violations)
	}
	// Names both colliding paths.
	if !strings.Contains(violations[0].Message, "notes/a/dune.md") ||
		!strings.Contains(violations[0].Message, "notes/b/dune.md") {
		t.Fatalf("expected both paths named, got %q", violations[0].Message)
	}
}

func TestUnmatchedFilesRunCollection_groupsDisallowedSubtrees(t *testing.T) {
	root := filepath.Join(t.TempDir(), "docs")
	violations := filesystem.UnmatchedFiles{}.RunCollection(checks.CollectionContext{
		Root: root,
		Items: []checks.ItemContext{
			{FilePath: filepath.Join(root, "ongoing/page.md")},
		},
		Unmatched: []string{
			"one-time/a.md",
			"one-time/deep/b.md",
			"ongoing/stray.tmp",
			"sunday-school/a.md",
			"sunday-school/b.md",
		},
		Include: []string{"README.md", "ongoing/*.md", "episodic/**"},
	})
	if len(violations) != 3 {
		t.Fatalf("expected 3 violations, got %d: %v", len(violations), violations)
	}
	wantFiles := []string{"one-time/", "ongoing/stray.tmp", "sunday-school/"}
	for i, want := range wantFiles {
		if violations[i].File != want {
			t.Errorf("violation %d file = %q, want %q", i, violations[i].File, want)
		}
	}
	for _, i := range []int{0, 2} {
		if !strings.Contains(violations[i].Message, "2 files") {
			t.Errorf("grouped violation %d should include file count, got %q", i, violations[i].Message)
		}
	}
	if strings.Contains(violations[1].Message, "files") {
		t.Errorf("single unmatched file should keep singular message, got %q", violations[1].Message)
	}
}

func TestUnmatchedFilesRunCollection_verboseReportsEachFile(t *testing.T) {
	violations := filesystem.UnmatchedFiles{}.RunCollection(checks.CollectionContext{
		Unmatched: []string{
			"one-time/a.md",
			"one-time/deep/b.md",
		},
		Verbose: true,
	})
	if len(violations) != 2 {
		t.Fatalf("expected 2 violations, got %d: %v", len(violations), violations)
	}
	if violations[0].File != "one-time/a.md" || violations[1].File != "one-time/deep/b.md" {
		t.Fatalf("verbose output should keep individual files, got %v", violations)
	}
}

func TestIndexFileRequired_flagsMissing(t *testing.T) {
	root := t.TempDir()
	withIndex := filepath.Join(root, "has")
	without := filepath.Join(root, "missing")
	if err := os.MkdirAll(withIndex, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(without, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withIndex, "_index.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := checks.CollectionContext{Items: []checks.ItemContext{
		{FilePath: filepath.Join(withIndex, "a.md")},
		{FilePath: filepath.Join(without, "b.md")},
	}}
	violations := filesystem.IndexFileRequired{}.RunCollection(ctx)
	if len(violations) != 1 || violations[0].File != without {
		t.Fatalf("expected one missing-index violation for %q, got %v", without, violations)
	}
}
