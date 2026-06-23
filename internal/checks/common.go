package checks

import "strings"

// MarkdownLine is one line of a document body paired with its 1-based source
// line number, the unit body-scanning check types (markdown headings, text
// spans) iterate over.
type MarkdownLine struct {
	Line int
	Text string
}

// MarkdownLines splits a body into numbered lines, counting from bodyLine (the
// document's first body line). It is shared by the markdownbodytext and
// plaintext families, which both scan the body line by line.
func MarkdownLines(body []byte, bodyLine int) []MarkdownLine {
	raw := strings.Split(string(body), "\n")
	out := make([]MarkdownLine, 0, len(raw))
	for i, text := range raw {
		out = append(out, MarkdownLine{
			Line: bodyLine + i,
			Text: text,
		})
	}
	return out
}

// AnchoredPattern wraps a user pattern so it must match the whole string. It is
// shared by the filesystem name_regex check (at build time) and the engine.
func AnchoredPattern(p string) string { return "^(?:" + p + ")$" }
