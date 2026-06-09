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
	lines := markdownLines(body, bodyLine)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line.Text)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")), line.Line, true
		}
	}
	return "", 0, false
}

// MarkdownRequiresH1 checks that the body has at least one H1.
type MarkdownRequiresH1 struct{}

func (m MarkdownRequiresH1) Run(ctx Context) []Violation {
	_, _, ok := firstH1(ctx.Doc.Body, ctx.Doc.BodyLine)
	if ok {
		return nil
	}
	return []Violation{{
		Path:    "/",
		Message: "missing H1 heading in markdown body",
	}}
}

// MarkdownSingleH1 checks that only one H1 is present.
type MarkdownSingleH1 struct{}

func (m MarkdownSingleH1) Run(ctx Context) []Violation {
	h1Lines := make([]int, 0)
	for _, line := range markdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		if level, _, ok := heading(line.Text); ok && level == 1 {
			h1Lines = append(h1Lines, line.Line)
		}
	}
	if len(h1Lines) <= 1 {
		return nil
	}
	return []Violation{{
		Path:    "/",
		Message: "markdown body must contain only one H1 heading",
		Line:    h1Lines[1],
	}}
}

// MarkdownNoHeadingLevelJumps checks that heading levels increase at most by one.
type MarkdownNoHeadingLevelJumps struct{}

func (m MarkdownNoHeadingLevelJumps) Run(ctx Context) []Violation {
	prevLevel := 0
	for _, line := range markdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		level, _, ok := heading(line.Text)
		if !ok {
			continue
		}
		if prevLevel > 0 && level > prevLevel+1 {
			return []Violation{{
				Path:    "/",
				Message: fmt.Sprintf("heading level jump from H%d to H%d is not allowed", prevLevel, level),
				Line:    line.Line,
			}}
		}
		prevLevel = level
	}
	return nil
}

// MarkdownRequiredSection checks that a specific heading exists.
type MarkdownRequiredSection struct {
	Heading string
}

func (m MarkdownRequiredSection) Run(ctx Context) []Violation {
	target := strings.TrimSpace(m.Heading)
	for _, line := range markdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		_, text, ok := heading(line.Text)
		if !ok {
			continue
		}
		if text == target {
			return nil
		}
	}
	return []Violation{{
		Path:    "/",
		Message: fmt.Sprintf("missing required section heading %q", target),
	}}
}

// MarkdownCodeFenceHasLanguage checks that fenced code blocks specify a language.
type MarkdownCodeFenceHasLanguage struct{}

func (m MarkdownCodeFenceHasLanguage) Run(ctx Context) []Violation {
	inFence := false
	for _, line := range markdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
		trimmed := strings.TrimSpace(line.Text)
		if !strings.HasPrefix(trimmed, "```") {
			continue
		}
		if inFence {
			inFence = false
			continue
		}
		lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
		if lang == "" {
			return []Violation{{
				Path:    "/",
				Message: "code fence opening must include a language",
				Line:    line.Line,
			}}
		}
		inFence = true
	}
	return nil
}

type markdownLine struct {
	Line int
	Text string
}

func markdownLines(body []byte, bodyLine int) []markdownLine {
	raw := strings.Split(string(body), "\n")
	out := make([]markdownLine, 0, len(raw))
	for i, text := range raw {
		out = append(out, markdownLine{
			Line: bodyLine + i,
			Text: text,
		})
	}
	return out
}

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
