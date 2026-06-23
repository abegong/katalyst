// Package markdownbodytext holds the check types that validate relationships
// between frontmatter metadata and markdown body content (headings, sections,
// code fences). Each check type lives in its own file with its Descriptor and
// self-registration.
package markdownbodytext

import (
	"strings"

	"github.com/abegong/katalyst/internal/checks"
)

// firstH1 returns the text, 1-based line, and presence of the body's first H1.
func firstH1(body []byte, bodyLine int) (string, int, bool) {
	for _, line := range checks.MarkdownLines(body, bodyLine) {
		trimmed := strings.TrimSpace(line.Text)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")), line.Line, true
		}
	}
	return "", 0, false
}

// heading parses an ATX heading line, returning its level (1-6), trimmed text,
// and whether the line is a heading at all.
func heading(line string) (level int, text string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	level = 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if len(trimmed) <= level || trimmed[level] != ' ' {
		return 0, "", false
	}
	return level, strings.TrimSpace(trimmed[level+1:]), true
}
