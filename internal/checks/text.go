package checks

import (
	"fmt"
	"regexp"
	"strings"
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
func textSpans(ctx Context, target string, sel *regexp.Regexp) []span {
	body := ctx.Doc.Body
	bodyLine := ctx.Doc.BodyLine
	switch target {
	case "", "body":
		return []span{{Text: string(body), Line: bodyLine}}
	case "first-line":
		for _, ln := range markdownLines(body, bodyLine) {
			if strings.TrimSpace(ln.Text) != "" {
				return []span{{Text: ln.Text, Line: ln.Line}}
			}
		}
		return nil
	case "line":
		lines := markdownLines(body, bodyLine)
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
		for _, ln := range markdownLines(body, bodyLine) {
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

// TextRequires asserts that an unanchored regex appears in the body. With
// All (match: all) every selected span must match; otherwise (match: any) at
// least one must.
type TextRequires struct {
	Re      *regexp.Regexp
	Pattern string
	Target  string
	Select  *regexp.Regexp
	All     bool
}

func (t TextRequires) Run(ctx Context) []Violation {
	spans := textSpans(ctx, t.Target, t.Select)
	if t.All {
		var out []Violation
		for _, s := range spans {
			if !t.Re.MatchString(s.Text) {
				out = append(out, Violation{
					Path:    "/",
					Message: fmt.Sprintf("required text /%s/ not found", t.Pattern),
					Line:    s.Line,
				})
			}
		}
		return out
	}
	for _, s := range spans {
		if t.Re.MatchString(s.Text) {
			return nil
		}
	}
	return []Violation{{
		Path:    "/",
		Message: fmt.Sprintf("required text /%s/ not found", t.Pattern),
		Line:    ctx.Doc.BodyLine,
	}}
}

// TextForbids asserts that an unanchored regex appears in no selected span.
type TextForbids struct {
	Re      *regexp.Regexp
	Pattern string
	Target  string
	Select  *regexp.Regexp
	// Fix is an optional replacement template applied to the matched text by
	// the fix command; empty means report-only.
	Fix string
}

func (t TextForbids) Run(ctx Context) []Violation {
	var out []Violation
	for _, s := range textSpans(ctx, t.Target, t.Select) {
		if loc := t.Re.FindStringIndex(s.Text); loc != nil {
			out = append(out, Violation{
				Path:    "/",
				Message: fmt.Sprintf("forbidden text /%s/ found", t.Pattern),
				Line:    lineOf(s, loc[0]),
			})
		}
	}
	return out
}

// ApplyFix returns body with the forbidden pattern replaced by the rule's fix
// template (regexp capture syntax) across its selected spans. It replaces only
// matched substrings, never whole spans; a rule with no fix returns body
// unchanged.
func (t TextForbids) ApplyFix(body []byte) []byte {
	if t.Fix == "" {
		return body
	}
	if t.Target == "" || t.Target == "body" {
		return t.Re.ReplaceAll(body, []byte(t.Fix))
	}
	lines := strings.Split(string(body), "\n")
	switch t.Target {
	case "first-line":
		for i := range lines {
			if strings.TrimSpace(lines[i]) != "" {
				lines[i] = t.Re.ReplaceAllString(lines[i], t.Fix)
				break
			}
		}
	case "line":
		for i := range lines {
			lines[i] = t.Re.ReplaceAllString(lines[i], t.Fix)
		}
	case "matched-lines":
		for i := range lines {
			if t.Select != nil && t.Select.MatchString(lines[i]) {
				lines[i] = t.Re.ReplaceAllString(lines[i], t.Fix)
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// TextDenylist forbids any of a list of literal substrings. Matching is
// literal (regex metacharacters are inert) via strings.Index.
type TextDenylist struct {
	Values []string
	Target string
	Select *regexp.Regexp
}

func (t TextDenylist) Run(ctx Context) []Violation {
	var out []Violation
	for _, s := range textSpans(ctx, t.Target, t.Select) {
		for _, v := range t.Values {
			if idx := strings.Index(s.Text, v); idx >= 0 {
				out = append(out, Violation{
					Path:    "/",
					Message: fmt.Sprintf("forbidden term %q found", v),
					Line:    lineOf(s, idx),
				})
			}
		}
	}
	return out
}
