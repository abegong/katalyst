package inspect

import "strings"

// mdInput is one markdown document for the markdown_body primitive: its body
// bytes and the title field (for the H1-matches-title facet).
type mdInput struct {
	Body  []byte
	Title string
}

// markdownBody profiles a set of markdown bodies, reporting two facets: heading
// shape (single-H1, H1-matches-title, level-jump rates) and recurring sections
// (level-2+ headings and how many documents contain each). This is the
// markdown_body primitive; the former markdown_* inspectors are facets of it.
func markdownBody(docs []mdInput) map[string]any {
	bodies, singleH1, h1MatchesTitle, hasJump := 0, 0, 0, 0
	sections := map[string]int{}
	for _, d := range docs {
		bodies++
		h1Count, prev := 0, 0
		firstH1, foundH1, jump := "", false, false
		seen := map[string]bool{}
		for _, h := range headings(d.Body) {
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
			if h.level >= 2 && !seen[h.text] {
				seen[h.text] = true
				sections[h.text]++
			}
		}
		if h1Count == 1 {
			singleH1++
		}
		if jump {
			hasJump++
		}
		if foundH1 && d.Title != "" && strings.TrimSpace(d.Title) == firstH1 {
			h1MatchesTitle++
		}
	}
	return map[string]any{
		"heading_shape": map[string]any{
			"bodies":           bodies,
			"single_h1":        singleH1,
			"h1_matches_title": h1MatchesTitle,
			"has_level_jump":   hasJump,
		},
		"sections": toAnyMap(sections),
	}
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
