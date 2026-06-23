package markdownbodytext

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// MarkdownWritingTells flags likely "AI slop" tells in a document's
// frontmatter and body and reports them as warnings, never errors. It is a
// review aid: many hits are fine in context, so it never fails a run. The
// catalog covers decorative punctuation, a vocabulary of overused words,
// and stock phrases. There is deliberately no allow list, judging each hit
// is the reader's job, and a smarter (LLM/ML) classifier that makes the
// judgment itself is future work (issue #57).
type MarkdownWritingTells struct{}

// tell pairs a human-readable label with the pattern that detects it.
type tell struct {
	label string
	re    *regexp.Regexp
}

// tellCatalog is built once at package init. Patterns are case-insensitive
// where that matters and word-bounded for vocabulary so "robustness" does
// not trip "robust".
var tellCatalog = buildTellCatalog()

func buildTellCatalog() []tell {
	var out []tell

	// Decorative punctuation. Technical typography (arrows, math signs) is
	// intentionally absent, it carries meaning and is not a tell.
	punct := []struct{ label, pat string }{
		{"em dash", `—`},
		{"en dash", `–`},
		{"curly double quote", `[“”]`},
		{"curly single quote", `[‘’]`},
		{"ellipsis character", `…`},
		{"non-breaking space", `\x{00A0}`},
		{"decorative emoji", `[\x{1F000}-\x{1FAFF}\x{2600}-\x{27BF}\x{2B00}-\x{2BFF}\x{FE0F}]`},
	}
	for _, p := range punct {
		out = append(out, tell{p.label, regexp.MustCompile(p.pat)})
	}

	// Overused words. Ordinary technical words in their own right; the
	// warning is a prompt to check they are doing work.
	words := []string{
		"delve", "delved", "delving", "delves",
		"leverage", "leverages", "leveraging",
		"seamless", "seamlessly",
		"robust", "comprehensive", "intricate", "holistic",
		"showcase", "showcases", "showcasing",
		"boast", "boasts", "boasting",
		"tapestry", "vibrant", "bustling", "pivotal",
		"meticulous", "meticulously",
		"elevate", "elevates", "empower", "empowers", "empowering",
		"foster", "fosters", "fostering",
		"harness", "harnesses", "harnessing",
		"unlock", "unlocks", "unlocking",
		"streamline", "streamlines", "streamlining",
		"underscore", "underscores", "underscoring",
		"myriad", "plethora", "paradigm", "synergy",
		"realm", "landscape", "ecosystem", "supercharge",
	}
	for _, w := range words {
		out = append(out, tell{
			"overused word: " + w,
			regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b`),
		})
	}

	// Stock phrases and constructions.
	phrases := []struct{ label, pat string }{
		{"\"not just X, it's Y\"", `(?i)it'?s not just\b.{0,40}?\bit'?s`},
		{"\"not only X but also\"", `(?i)not only\b.{0,40}?\bbut also`},
		{"\"whether you're X or Y\"", `(?i)whether you'?re\b.{0,40}?\bor\b`},
		{"\"in today's ... world\"", `(?i)in today'?s\b.{0,30}?\b(fast-paced|digital|modern)\b`},
		{"\"at the end of the day\"", `(?i)at the end of the day`},
		{"\"when it comes to\"", `(?i)when it comes to`},
		{"\"it's worth noting\"", `(?i)it'?s worth noting`},
		{"\"it's important to ...\"", `(?i)it (is|'?s) important to (note|remember|understand)`},
		{"\"needless to say\"", `(?i)needless to say`},
		{"\"that being said\"", `(?i)that being said`},
		{"\"rest assured\"", `(?i)rest assured`},
		{"\"look no further\"", `(?i)look no further`},
		{"\"let's dive/explore\"", `(?i)let'?s (dive|explore|take a look)`},
		{"\"dive into\"", `(?i)dive (in|into)\b`},
		{"\"embark on a journey\"", `(?i)embark on (a|your)\b`},
		{"\"in conclusion/summary\"", `(?i)in (conclusion|summary)\b`},
		{"\"a testament to\"", `(?i)a testament to`},
		{"\"navigate the complexities\"", `(?i)navigat(e|ing) the (complexities|landscape)`},
		{"\"plays a crucial role\"", `(?i)plays? a (crucial|pivotal|vital|key|significant) role`},
		{"\"unlock the potential\"", `(?i)unlock(ing)? the (power|potential)`},
		{"\"cutting-edge\"", `(?i)cutting[- ]edge`},
		{"\"game-changer\"", `(?i)game[- ]?changer`},
		{"\"ever-evolving\"", `(?i)ever[- ](evolving|changing|growing)`},
	}
	for _, p := range phrases {
		out = append(out, tell{p.label, regexp.MustCompile(p.pat)})
	}

	return out
}

func (MarkdownWritingTells) Run(ctx checks.Context) []checks.Violation {
	var out []checks.Violation
	scan := func(text string, line int) {
		for _, t := range tellCatalog {
			if loc := t.re.FindString(text); loc != "" {
				out = append(out, checks.Violation{
					Path:     "/",
					Line:     line,
					Severity: checks.SeverityWarning,
					Message:  fmt.Sprintf("writing tell (%s): %q (review and revise if it reads as filler)", t.label, loc),
				})
			}
		}
	}

	if ctx.Doc != nil {
		// Frontmatter raw bytes start one line below the opening fence.
		if len(ctx.Doc.Frontmatter) > 0 {
			for i, text := range strings.Split(string(ctx.Doc.Frontmatter), "\n") {
				scan(text, i+2)
			}
		}
		for _, ln := range checks.MarkdownLines(ctx.Doc.Body, ctx.Doc.BodyLine) {
			scan(ln.Text, ln.Line)
		}
	}
	return out
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckMarkdownWritingTells,
		Family:    "markdownBodyText",
		Slug:      "writing-tells",
		Severity:  "warning",
		Title:     "Writing tells",
		Summary:   "Warn on likely AI-writing tells (em dashes, decorative emoji, overused words, stock phrases) for human review.",
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_writing_tells`,
	}, func(ch config.CheckInstance) checks.Check {
		return MarkdownWritingTells{}
	}, nil)
}
