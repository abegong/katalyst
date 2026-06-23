package plaintext_test

import (
	"regexp"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/checktest"
	"github.com/abegong/katalyst/internal/checks/plaintext"
)

// textCtx parses src and wraps it in a Context for a text rule.
func textCtx(t *testing.T, src string) checks.Context {
	t.Helper()
	return checks.Context{FilePath: "notes/n.md", Doc: checktest.MustParseDoc(t, src)}
}

func TestTextForbids_findsAndLocatesMatch(t *testing.T) {
	// Body starts at line 4; "second TODO" is line 5.
	ctx := textCtx(t, "---\ntitle: T\n---\nfirst line\nsecond TODO line\n")
	v := plaintext.TextForbids{Re: regexp.MustCompile(`\bTODO\b`), Pattern: `\bTODO\b`}.Run(ctx)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(v))
	}
	if v[0].Line != 5 {
		t.Errorf("expected line 5, got %d", v[0].Line)
	}
	if v[0].Path != "/" {
		t.Errorf("expected body path /, got %q", v[0].Path)
	}
}

func TestTextForbids_passesWhenAbsent(t *testing.T) {
	ctx := textCtx(t, "---\ntitle: T\n---\nall clear here\n")
	if v := (plaintext.TextForbids{Re: regexp.MustCompile(`\bTODO\b`), Pattern: `TODO`}).Run(ctx); len(v) != 0 {
		t.Fatalf("expected no violations, got %d", len(v))
	}
}

func TestTextForbids_perLineReportsEachSpan(t *testing.T) {
	ctx := textCtx(t, "---\nt: x\n---\nTODO one\nclean\nTODO two\n")
	v := plaintext.TextForbids{Re: regexp.MustCompile(`TODO`), Pattern: `TODO`, Target: "line"}.Run(ctx)
	if len(v) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(v))
	}
	if v[0].Line != 4 || v[1].Line != 6 {
		t.Errorf("expected lines 4 and 6, got %d and %d", v[0].Line, v[1].Line)
	}
}

func TestTextRequires_anyMatchesOnOneSpan(t *testing.T) {
	ctx := textCtx(t, "---\nt: x\n---\nintro\n## Sources\n- a\n")
	if v := (plaintext.TextRequires{Re: regexp.MustCompile(`Sources`), Pattern: `Sources`, Target: "line"}).Run(ctx); len(v) != 0 {
		t.Fatalf("any: expected pass, got %d violations", len(v))
	}
	ctx2 := textCtx(t, "---\nt: x\n---\nintro only\n")
	v := plaintext.TextRequires{Re: regexp.MustCompile(`Sources`), Pattern: `Sources`}.Run(ctx2)
	if len(v) != 1 {
		t.Fatalf("any: expected 1 violation, got %d", len(v))
	}
	if v[0].Line != 4 { // body start line
		t.Errorf("expected body-start line 4, got %d", v[0].Line)
	}
}

func TestTextRequires_allRequiresEverySpan(t *testing.T) {
	// Every line must contain "x".
	pass := textCtx(t, "---\nt: x\n---\nx1\nx2\n")
	if v := (plaintext.TextRequires{Re: regexp.MustCompile(`x`), Pattern: `x`, Target: "line", All: true}).Run(pass); len(v) != 0 {
		t.Fatalf("all: expected pass, got %d violations", len(v))
	}
	fail := textCtx(t, "---\nt: x\n---\nx1\ny2\n")
	v := plaintext.TextRequires{Re: regexp.MustCompile(`x`), Pattern: `x`, Target: "line", All: true}.Run(fail)
	if len(v) != 1 {
		t.Fatalf("all: expected 1 violation, got %d", len(v))
	}
	if v[0].Line != 5 { // the y2 line
		t.Errorf("expected failing line 5, got %d", v[0].Line)
	}
}

func TestTextRequires_anyEqualsAllForBody(t *testing.T) {
	ctx := textCtx(t, "---\nt: x\n---\nhas Sources somewhere\n")
	any := plaintext.TextRequires{Re: regexp.MustCompile(`Sources`), Pattern: `Sources`, Target: "body"}.Run(ctx)
	all := plaintext.TextRequires{Re: regexp.MustCompile(`Sources`), Pattern: `Sources`, Target: "body", All: true}.Run(ctx)
	if len(any) != 0 || len(all) != 0 {
		t.Fatalf("body any/all should both pass; got %d/%d", len(any), len(all))
	}
}

func TestTextDenylist_flagsLiteralsMetacharsInert(t *testing.T) {
	ctx := textCtx(t, "---\nt: x\n---\nthis has FIXME inside\n")
	v := plaintext.TextDenylist{Values: []string{"TODO", "FIXME"}}.Run(ctx)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(v))
	}

	// "a.b" is a literal: it must not match "axb" (the dot is inert).
	inert := textCtx(t, "---\nt: x\n---\naxb\n")
	if v := (plaintext.TextDenylist{Values: []string{"a.b"}}).Run(inert); len(v) != 0 {
		t.Fatalf("metachar should be inert; got %d violations", len(v))
	}
	literal := textCtx(t, "---\nt: x\n---\na.b\n")
	if v := (plaintext.TextDenylist{Values: []string{"a.b"}}).Run(literal); len(v) != 1 {
		t.Fatalf("literal a.b should match; got %d violations", len(v))
	}
}

func TestTextSpans_firstLineSkipsLeadingBlanks(t *testing.T) {
	ctx := textCtx(t, "---\nt: x\n---\n\n\nHeadline\n")
	// first-line is "Headline" (line 6), not the blank lines.
	v := plaintext.TextForbids{Re: regexp.MustCompile(`Headline`), Pattern: `Headline`, Target: "first-line"}.Run(ctx)
	if len(v) != 1 || v[0].Line != 6 {
		t.Fatalf("expected first-line hit on line 6, got %+v", v)
	}
}

func TestTextSpans_matchedLinesUsesSelect(t *testing.T) {
	ctx := textCtx(t, "---\nt: x\n---\n- TODO bullet\nparagraph TODO\n- clean bullet\n")
	// Only bullet lines (select ^-) are checked, so the paragraph TODO is ignored.
	sel := regexp.MustCompile(`^-`)
	v := plaintext.TextForbids{Re: regexp.MustCompile(`TODO`), Pattern: `TODO`, Target: "matched-lines", Select: sel}.Run(ctx)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation (bullet only), got %d", len(v))
	}
	if v[0].Line != 4 {
		t.Errorf("expected line 4, got %d", v[0].Line)
	}
}

func TestTextForbids_frontmatterlessFileLintsFromLineOne(t *testing.T) {
	ctx := textCtx(t, "plain TODO file\n")
	v := plaintext.TextForbids{Re: regexp.MustCompile(`TODO`), Pattern: `TODO`}.Run(ctx)
	if len(v) != 1 || v[0].Line != 1 {
		t.Fatalf("expected line-1 hit on frontmatter-less file, got %+v", v)
	}
}

func TestTextForbids_bodyTargetReportsMatchLine(t *testing.T) {
	// target body: the whole body is one span; the match is on body line 2 (line 5).
	ctx := textCtx(t, "---\nt: x\n---\nfirst\nsecond TODO\nthird\n")
	v := plaintext.TextForbids{Re: regexp.MustCompile(`TODO`), Pattern: `TODO`, Target: "body"}.Run(ctx)
	if len(v) != 1 || v[0].Line != 5 {
		t.Fatalf("expected body match on line 5, got %+v", v)
	}
}
