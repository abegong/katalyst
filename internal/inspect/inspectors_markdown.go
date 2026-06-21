package inspect

import "strings"

// MarkdownHeadingShape reports body heading conventions across the corpus: how
// many files have exactly one H1, how many have their first H1 matching a
// `title` field, and how many contain a heading-level jump.
type MarkdownHeadingShape struct{}

func (MarkdownHeadingShape) Name() string { return "markdown_heading_shape" }

func (MarkdownHeadingShape) Inspect(c Corpus) Evidence {
	bodies, singleH1, h1MatchesTitle, hasJump := 0, 0, 0, 0
	for _, f := range c.Files {
		if f.Doc == nil {
			continue
		}
		bodies++
		hs := headings(f.Doc.Body)
		h1Count, prev := 0, 0
		firstH1, foundH1, jump := "", false, false
		for _, h := range hs {
			if h.level == 1 {
				h1Count++
				if !foundH1 {
					firstH1, foundH1 = h.text, true
				}
			}
			if prev > 0 && h.level > prev+1 {
				jump = true
			}
			prev = h.level
		}
		if h1Count == 1 {
			singleH1++
		}
		if jump {
			hasJump++
		}
		if foundH1 {
			if title, ok := meta(f)["title"].(string); ok && strings.TrimSpace(title) == firstH1 {
				h1MatchesTitle++
			}
		}
	}
	data := map[string]any{
		"bodies":           bodies,
		"single_h1":        singleH1,
		"h1_matches_title": h1MatchesTitle,
		"has_level_jump":   hasJump,
	}
	return Evidence{Inspector: "markdown_heading_shape", Scope: c.Scope, N: len(c.Files), Data: data}
}

// MarkdownSections reports recurring section headings (level 2 and deeper) and
// how many files contain each — the signal behind a required-section check.
type MarkdownSections struct{}

func (MarkdownSections) Name() string { return "markdown_sections" }

func (MarkdownSections) Inspect(c Corpus) Evidence {
	counts := map[string]int{}
	for _, f := range c.Files {
		if f.Doc == nil {
			continue
		}
		seen := map[string]bool{}
		for _, h := range headings(f.Doc.Body) {
			if h.level < 2 || seen[h.text] {
				continue
			}
			seen[h.text] = true
			counts[h.text]++
		}
	}
	data := make(map[string]any, len(counts))
	for text, n := range counts {
		data[text] = n
	}
	return Evidence{Inspector: "markdown_sections", Scope: c.Scope, N: len(c.Files), Data: data}
}

// MarkdownCodeFences reports how many fenced code blocks open across the corpus
// and how many of those carry a language tag.
type MarkdownCodeFences struct{}

func (MarkdownCodeFences) Name() string { return "markdown_code_fences" }

func (MarkdownCodeFences) Inspect(c Corpus) Evidence {
	opening, tagged, filesWith := 0, 0, 0
	for _, f := range c.Files {
		if f.Doc == nil {
			continue
		}
		inFence, fileHas := false, false
		for _, line := range strings.Split(string(f.Doc.Body), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "```") {
				continue
			}
			if inFence {
				inFence = false
				continue
			}
			inFence = true
			opening++
			fileHas = true
			if strings.TrimSpace(strings.TrimPrefix(trimmed, "```")) != "" {
				tagged++
			}
		}
		if fileHas {
			filesWith++
		}
	}
	data := map[string]any{
		"opening_fences":    opening,
		"with_language":     tagged,
		"files_with_fences": filesWith,
	}
	return Evidence{Inspector: "markdown_code_fences", Scope: c.Scope, N: len(c.Files), Data: data}
}

// mdHeading is one ATX heading: its level and trimmed text.
type mdHeading struct {
	level int
	text  string
}

// headings extracts ATX headings ("# ", "## ", …) from a markdown body. It is
// deliberately local to inspect rather than reaching into internal/checks,
// whose equivalent helper is unexported.
func headings(body []byte) []mdHeading {
	var out []mdHeading
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		level := 0
		for level < len(trimmed) && trimmed[level] == '#' {
			level++
		}
		if level == 0 || level > 6 || len(trimmed) <= level || trimmed[level] != ' ' {
			continue
		}
		out = append(out, mdHeading{level: level, text: strings.TrimSpace(trimmed[level+1:])})
	}
	return out
}
