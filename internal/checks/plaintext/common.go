// Package plaintext holds the check types that validate body content as raw
// text, independent of markdown structure (requires, forbids, denylist). Each
// check type lives in its own file with its Descriptor and self-registration.
package plaintext

import (
	"regexp"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
)

// span is a slice of body text a text rule is evaluated against, paired with
// the 1-based line where the slice begins.
type span struct {
	Text string
	Line int
}

// textSpans builds the spans a text rule evaluates, selected by target (and,
// for "matched-lines", the precompiled select regex). An empty target defaults
// to "body". The body and line numbers come straight from the parsed document,
// so a frontmatter-less file (Body == whole file, BodyLine == 1) lints with
// lines counted from 1.
func textSpans(ctx checks.Context, target string, sel *regexp.Regexp) []span {
	body := ctx.Doc.Body
	bodyLine := ctx.Doc.BodyLine
	switch target {
	case "", "body":
		return []span{{Text: string(body), Line: bodyLine}}
	case "first-line":
		for _, ln := range checks.MarkdownLines(body, bodyLine) {
			if strings.TrimSpace(ln.Text) != "" {
				return []span{{Text: ln.Text, Line: ln.Line}}
			}
		}
		return nil
	case "line":
		lines := checks.MarkdownLines(body, bodyLine)
		// Drop a single trailing empty line produced by a final newline, so
		// "every line" rules are not defeated by the terminator.
		if n := len(lines); n > 0 && lines[n-1].Text == "" {
			lines = lines[:n-1]
		}
		out := make([]span, 0, len(lines))
		for _, ln := range lines {
			out = append(out, span{Text: ln.Text, Line: ln.Line})
		}
		return out
	case "matched-lines":
		var out []span
		for _, ln := range checks.MarkdownLines(body, bodyLine) {
			if sel != nil && sel.MatchString(ln.Text) {
				out = append(out, span{Text: ln.Text, Line: ln.Line})
			}
		}
		return out
	}
	return nil
}

// lineOf returns the 1-based line of an offset within a span's text.
func lineOf(s span, offset int) int {
	return s.Line + strings.Count(s.Text[:offset], "\n")
}

// CompileSelect compiles the matched-lines line-filter regex, or returns nil
// when no select is configured. The pattern was validated at config load, so a
// compile failure here is impossible. The fix command reuses it to rebuild a
// rule's selector.
func CompileSelect(sel string) *regexp.Regexp {
	if sel == "" {
		return nil
	}
	return regexp.MustCompile(sel)
}
