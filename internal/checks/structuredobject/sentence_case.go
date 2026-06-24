package structuredobject

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project/config"
)

// ObjectSentenceCase checks that a string field reads as sentence case rather
// than Title Case: the first word is capitalized and every following word is
// lowercase, except all-caps tokens (acronyms like CI, H1) and an allowlist of
// proper nouns. It is the rule behind the docs table-of-contents convention
// ("Progressive operations", not "Progressive Operations").
type ObjectSentenceCase struct {
	Field string
	// Allow is the set of words permitted to keep a leading capital mid-title
	// (proper nouns, e.g. Katalyst). Matched case-sensitively against a word
	// stripped of surrounding punctuation.
	Allow map[string]bool
}

func (o ObjectSentenceCase) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + o.Field
	v, ok := ctx.Meta[o.Field]
	if !ok {
		return nil // presence is the job of object_required_field / the schema
	}
	s, ok := v.(string)
	if !ok {
		return nil // type is the job of object_field_type / the schema
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	line := checks.LookupLine(ctx.Doc.Lines, ptr)

	// The first word must start with a capital letter (or carry no letters,
	// e.g. a leading number); a lowercase opener is not sentence case.
	if first, lead, ok := firstLetter(words[0]); ok && unicode.IsLower(lead) {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be sentence case: first word %q should be capitalized", o.Field, first),
			Line:    line,
		}}
	}

	// Every following word should be lowercase, unless it is an acronym (no
	// lowercase letters, e.g. CI, H1) or an allowlisted proper noun.
	for _, w := range words[1:] {
		word, lead, ok := firstLetter(w)
		if !ok || !unicode.IsUpper(lead) {
			continue
		}
		if o.Allow[word] || isAcronym(word) {
			continue
		}
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("field %q must be sentence case: %q should not be capitalized", o.Field, word),
			Line:    line,
		}}
	}
	return nil
}

// firstLetter strips surrounding punctuation from a word and returns the
// trimmed word plus its first letter. ok is false when the word has no letters
// (pure punctuation or digits), in which case it carries no case to judge.
func firstLetter(w string) (word string, lead rune, ok bool) {
	word = strings.TrimFunc(w, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, r := range word {
		if unicode.IsLetter(r) {
			return word, r, true
		}
	}
	return word, 0, false
}

// isAcronym reports whether a word carries no lowercase letters, so an all-caps
// or letter+digit token like CI or H1 is left alone.
func isAcronym(word string) bool {
	for _, r := range word {
		if unicode.IsLower(r) {
			return false
		}
	}
	return true
}

func init() {
	register(checks.Descriptor{
		CheckType: config.CheckObjectSentenceCase,
		Family:    "structuredObject",
		Slug:      "sentence-case",
		Title:     "Sentence case",
		Summary:   "Require a string field to read as sentence case, not Title Case.",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key whose string value must be sentence case."},
			{Name: "allow", Required: false, Desc: "Proper nouns permitted to keep a leading capital mid-title (e.g. `Katalyst`). All-caps acronyms (CI, H1) are always allowed."},
		},
		ConfigExample: `collections:
  pages:
    path: docs/content
    checks:
      - kind: object_sentence_case
        field: title
        allow: [Katalyst]`,
	}, func(ch config.CheckInstance) checks.Check {
		allow := make(map[string]bool, len(ch.Allow))
		for _, a := range ch.Allow {
			allow[a] = true
		}
		return ObjectSentenceCase{Field: ch.Field, Allow: allow}
	}, nil)
}
