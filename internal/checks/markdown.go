package checks

import (
	"fmt"
	"strings"
)

// MarkdownTitleMatchesH1 checks that a frontmatter field matches the first H1.
type MarkdownTitleMatchesH1 struct {
	Field string
}

func (m MarkdownTitleMatchesH1) Run(ctx Context) []Violation {
	ptr := "/" + m.Field
	raw, ok := ctx.Meta[m.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing frontmatter field %q", m.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	title, ok := raw.(string)
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("frontmatter field %q must be a string", m.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}

	h1, h1Line, found := firstH1(ctx.Doc.Body, ctx.Doc.BodyLine)
	if !found {
		return []Violation{{
			Path:    "/",
			Message: "missing H1 heading in markdown body",
			Line:    0,
		}}
	}
	if strings.TrimSpace(title) == h1 {
		return nil
	}
	return []Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("%q does not match first H1 %q", title, h1),
		Line:    h1Line,
	}}
}

func firstH1(body []byte, bodyLine int) (string, int, bool) {
	lines := strings.Split(string(body), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")), bodyLine + i, true
		}
	}
	return "", 0, false
}
