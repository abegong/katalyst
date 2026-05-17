package frontmatter_test

import (
	"strings"
	"testing"

	"github.com/katabase-ai/katabridge/internal/frontmatter"
)

func TestFormat_sortsTopLevelKeys(t *testing.T) {
	src := "---\nzebra: 1\napple: 2\nmiddle: 3\n---\nbody\n"

	got, err := frontmatter.Format([]byte(src))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}

	want := "---\napple: 2\nmiddle: 3\nzebra: 1\n---\nbody\n"
	if string(got) != want {
		t.Errorf("Format mismatch:\n got: %q\nwant: %q", string(got), want)
	}
}

func TestFormat_preservesBodyVerbatim(t *testing.T) {
	// The leading "\n" between the closing fence and "# Heading" is
	// consumed by Parse as part of the closing-fence terminator, so
	// it doesn't survive a round-trip through Format. The interior
	// of the body (including triple-backtick fenced code containing
	// YAML-looking text) MUST round-trip verbatim.
	bodyAsSeen := "# Heading\n\n```code\nbecause: yaml\n```\n\nMore body.\n"
	src := "---\nb: 2\na: 1\n---\n" + bodyAsSeen

	got, err := frontmatter.Format([]byte(src))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}

	if !strings.HasSuffix(string(got), bodyAsSeen) {
		t.Errorf("body changed.\n got: %q\nwant suffix: %q", string(got), bodyAsSeen)
	}
}

func TestFormat_ensuresSingleTrailingNewline(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"no trailing newline", "---\na: 1\n---\nbody"},
		{"many trailing newlines", "---\na: 1\n---\nbody\n\n\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := frontmatter.Format([]byte(tc.src))
			if err != nil {
				t.Fatalf("Format: %v", err)
			}
			s := string(got)
			if !strings.HasSuffix(s, "\n") {
				t.Errorf("expected trailing newline, got: %q", s)
			}
			if strings.HasSuffix(s, "\n\n") {
				t.Errorf("expected exactly one trailing newline, got: %q", s)
			}
		})
	}
}

func TestFormat_noFrontmatter_isNoop(t *testing.T) {
	src := "# Just a heading\n\nNo frontmatter here.\n"
	got, err := frontmatter.Format([]byte(src))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if string(got) != src {
		t.Errorf("expected verbatim output for input without frontmatter, got: %q", got)
	}
}

func TestFormat_emptyFrontmatter_normalizes(t *testing.T) {
	src := "---\n---\nbody\n"
	got, err := frontmatter.Format([]byte(src))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	want := "---\n{}\n---\nbody\n"
	if string(got) != want {
		t.Errorf("Format mismatch:\n got: %q\nwant: %q", string(got), want)
	}
}
