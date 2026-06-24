package document_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/storage/collection/document"
)

// --- TOML ---------------------------------------------------------------

func TestParse_extractsTOMLFrontmatter(t *testing.T) {
	src := strings.Join([]string{
		"+++",
		`title = "Dune"`,
		"year = 1965",
		`tags = ["sci-fi", "classic"]`,
		"+++",
		"",
		"# Dune",
		"",
		"A story about spice.",
	}, "\n")

	doc, err := document.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !doc.HasFrontmatter {
		t.Fatalf("expected HasFrontmatter=true")
	}
	if doc.Format != document.KindTOML {
		t.Errorf("Format = %v, want KindTOML", doc.Format)
	}

	// TOML decodes integers as int64.
	want := map[string]any{
		"title": "Dune",
		"year":  int64(1965),
		"tags":  []any{"sci-fi", "classic"},
	}
	if !reflect.DeepEqual(doc.Meta, want) {
		t.Errorf("Meta mismatch:\n got: %#v\nwant: %#v", doc.Meta, want)
	}

	wantBody := "\n# Dune\n\nA story about spice."
	if string(doc.Body) != wantBody {
		t.Errorf("Body mismatch:\n got: %q\nwant: %q", string(doc.Body), wantBody)
	}

	wantFM := "title = \"Dune\"\nyear = 1965\ntags = [\"sci-fi\", \"classic\"]\n"
	if string(doc.Frontmatter) != wantFM {
		t.Errorf("Frontmatter mismatch:\n got: %q\nwant: %q", string(doc.Frontmatter), wantFM)
	}
}

func TestParse_emptyTOMLFrontmatter(t *testing.T) {
	doc, err := document.Parse([]byte("+++\n+++\nbody\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !doc.HasFrontmatter || doc.Format != document.KindTOML {
		t.Fatalf("expected TOML frontmatter, got HasFrontmatter=%v Format=%v", doc.HasFrontmatter, doc.Format)
	}
	if len(doc.Meta) != 0 {
		t.Errorf("expected empty Meta, got %#v", doc.Meta)
	}
	if string(doc.Body) != "body\n" {
		t.Errorf("Body mismatch: got %q", string(doc.Body))
	}
}

func TestParse_unterminatedTOML(t *testing.T) {
	_, err := document.Parse([]byte("+++\ntitle = \"Dune\"\n\n# Body\n"))
	if !errors.Is(err, document.ErrUnterminated) {
		t.Fatalf("expected ErrUnterminated, got %v", err)
	}
}

func TestParse_invalidTOML(t *testing.T) {
	_, err := document.Parse([]byte("+++\ntitle = = =\n+++\nbody\n"))
	if !errors.Is(err, document.ErrInvalidTOML) {
		t.Fatalf("expected ErrInvalidTOML, got %v", err)
	}
}

func TestFormat_TOMLRoundTrip(t *testing.T) {
	// TOML's canonical form sorts top-level keys and uses its own scalar
	// styling, but the "+++" fences and the format itself are preserved.
	src := "+++\nzebra = 2\napple = 1\n+++\nbody\n"
	got, err := reencode(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := "+++\napple = 1\nzebra = 2\n+++\nbody\n"
	if string(got) != want {
		t.Errorf("Format mismatch:\n got: %q\nwant: %q", string(got), want)
	}

	// Re-parsing the formatted output yields the same metadata: round-trip
	// is meaning-preserving and never rewrites TOML as another format.
	reparsed, err := document.Parse(got)
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}
	if reparsed.Format != document.KindTOML {
		t.Errorf("re-parsed Format = %v, want KindTOML", reparsed.Format)
	}
}

// --- JSON ---------------------------------------------------------------

func TestParse_extractsJSONFrontmatter(t *testing.T) {
	src := strings.Join([]string{
		"{",
		`  "title": "Dune",`,
		`  "year": 1965,`,
		`  "tags": ["sci-fi", "classic"]`,
		"}",
		"",
		"# Dune",
		"",
		"A story about spice.",
	}, "\n")

	doc, err := document.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if !doc.HasFrontmatter {
		t.Fatalf("expected HasFrontmatter=true")
	}
	if doc.Format != document.KindJSON {
		t.Errorf("Format = %v, want KindJSON", doc.Format)
	}

	// JSON decodes numbers as float64.
	want := map[string]any{
		"title": "Dune",
		"year":  float64(1965),
		"tags":  []any{"sci-fi", "classic"},
	}
	if !reflect.DeepEqual(doc.Meta, want) {
		t.Errorf("Meta mismatch:\n got: %#v\nwant: %#v", doc.Meta, want)
	}

	wantBody := "\n# Dune\n\nA story about spice."
	if string(doc.Body) != wantBody {
		t.Errorf("Body mismatch:\n got: %q\nwant: %q", string(doc.Body), wantBody)
	}

	// The raw Frontmatter block includes the braces for JSON.
	if !strings.HasPrefix(string(doc.Frontmatter), "{") || !strings.HasSuffix(string(doc.Frontmatter), "}") {
		t.Errorf("expected Frontmatter to include braces, got %q", string(doc.Frontmatter))
	}
}

func TestParse_emptyJSONFrontmatter(t *testing.T) {
	doc, err := document.Parse([]byte("{}\nbody\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !doc.HasFrontmatter || doc.Format != document.KindJSON {
		t.Fatalf("expected JSON frontmatter, got HasFrontmatter=%v Format=%v", doc.HasFrontmatter, doc.Format)
	}
	if len(doc.Meta) != 0 {
		t.Errorf("expected empty Meta, got %#v", doc.Meta)
	}
	if string(doc.Body) != "body\n" {
		t.Errorf("Body mismatch: got %q", string(doc.Body))
	}
}

// Braces appearing inside string values must not be mistaken for the
// closing fence.
func TestParse_JSONBraceInString(t *testing.T) {
	src := "{\n  \"title\": \"a } brace\"\n}\nbody\n"
	doc, err := document.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if doc.Meta["title"] != "a } brace" {
		t.Errorf("title mismatch: got %#v", doc.Meta["title"])
	}
	if string(doc.Body) != "body\n" {
		t.Errorf("Body mismatch: got %q", string(doc.Body))
	}
}

func TestParse_unterminatedJSON(t *testing.T) {
	_, err := document.Parse([]byte("{\n  \"title\": \"Dune\"\n\n# Body\n"))
	if !errors.Is(err, document.ErrUnterminated) {
		t.Fatalf("expected ErrUnterminated, got %v", err)
	}
}

func TestParse_invalidJSON(t *testing.T) {
	_, err := document.Parse([]byte("{\n  \"title\": ,\n}\nbody\n"))
	if !errors.Is(err, document.ErrInvalidJSON) {
		t.Fatalf("expected ErrInvalidJSON, got %v", err)
	}
}

func TestFormat_JSONRoundTrip(t *testing.T) {
	// JSON's canonical form sorts keys and indents two spaces; the brace
	// delimiters and the format itself are preserved.
	src := "{\n  \"zebra\": 2,\n  \"apple\": 1\n}\nbody\n"
	got, err := reencode(src)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := "{\n  \"apple\": 1,\n  \"zebra\": 2\n}\nbody\n"
	if string(got) != want {
		t.Errorf("Format mismatch:\n got: %q\nwant: %q", string(got), want)
	}

	reparsed, err := document.Parse(got)
	if err != nil {
		t.Fatalf("re-Parse: %v", err)
	}
	if reparsed.Format != document.KindJSON {
		t.Errorf("re-parsed Format = %v, want KindJSON", reparsed.Format)
	}
}

// --- Cross-format -------------------------------------------------------

// YAML "---" frontmatter is still detected as YAML; the new formats don't
// disturb the original path.
func TestParse_yamlStillDetected(t *testing.T) {
	doc, err := document.Parse([]byte("---\ntitle: Dune\n---\nbody\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if doc.Format != document.KindYAML {
		t.Errorf("Format = %v, want KindYAML", doc.Format)
	}
}

func TestKind_String(t *testing.T) {
	cases := map[document.Kind]string{
		document.KindYAML: "yaml",
		document.KindTOML: "toml",
		document.KindJSON: "json",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

// reencode parses src and re-serializes it canonically — the round trip the
// fix operation performs, exercised here at the codec level (Parse + Encode).
func reencode(src string) ([]byte, error) {
	doc, err := document.Parse([]byte(src))
	if err != nil {
		return nil, err
	}
	return document.Encode(doc)
}
