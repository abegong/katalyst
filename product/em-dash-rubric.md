# Em-dash removal rubric (draft)

Status: **applied.** The repo has been swept per this rubric (category 1 uses
colons; the docs dogfood keeps the `markdown_writing_tells` warning). The
document now stands as the convention and the reference for future edits. The
em-dash guidance in `.cursor/skills/write-docs/SKILL.md` has been reversed to
match.

## Why this exists

Em dashes read as a tell of unedited AI writing, and we want to scrub the ones
that are filler. But they are not uniformly bad, and the repo currently treats
them as house style:

- `.cursor/skills/write-docs/SKILL.md` says: *"Em dashes. Use the `—`
  character with a space on each side (`a term — its definition`)."*
- `.cursor/skills/write-domain-model/SKILL.md` refers to the
  *"bold-term-em-dash definitions used across code, docs, and copy."*

So there are ~405 em dashes in the repo on purpose. The
`markdown_writing_tells` check now surfaces every one as a **warning** (never a
build failure). This rubric turns that pile of warnings into per-case
decisions, and adopting it means **reversing the skill guidance above**.

## How to use this

For each flagged em dash, find the first category below that matches and apply
its action. Categories are ordered from most mechanical (apply without
thinking) to most judgment-heavy (a human should decide). If two match, prefer
the earlier (more confident) one.

| # | Category | Trigger (unambiguous test) | Action | Confidence |
|---|----------|----------------------------|--------|------------|
| 1 | Term–gloss list item | A list bullet or bold/italic lead-in of the form `**Term** — gloss` | Replace ` — ` with a colon inside the emphasis: `**Term:** gloss` | High / mechanical |
| 2 | Numeric or date range | Dash sits between two numbers/dates/endpoints with no spaces | Replace with `-` (or the word "to") | High / mechanical |
| 3 | Generated-doc text | The em dash is in `docs/content/reference/**` (generated) | Do not edit the file; fix the template in `cmd/gendocs` / the descriptor, then `make docs-gen` | High / mechanical |
| 4 | Code literal / UI glyph | The em dash is a Go string literal used as a value (e.g. `def = "—"`) or asserted in a test | Replace with `"-"` and update the paired test in the same change | High, but edit code+test together |
| 5 | Paired parenthetical | Two em dashes bracket a removable clause: `X — aside — Y` | Commas if the aside has no internal commas; parentheses if it does | Medium / rule-based |
| 6 | Single aside that explains/lists | One em dash; the text after it explains, restates, or enumerates what precedes | Colon: `X: explanation` | Medium / rule-based |
| 7 | Single contrastive aside | One em dash followed by a contrast, often with `but`/`not`/`yet` | Comma: `X, but Y` | Medium / rule-based |
| 8 | Two independent clauses | Both sides of a single em dash stand alone as sentences, and the line is long | Split into two sentences (period), or rewrite | Low / judgment |
| - | Leave as-is | See "What to leave alone" | No change | n/a |

## Categories in detail

### 1. Term–gloss list item (mechanical) — highest value

This is the dominant intentional pattern and the safest to convert. A colon is
the standard ASCII definition-list form and reads identically.

Trigger: a bullet or emphasized lead-in immediately followed by ` — ` and a
description.

Real examples:

```
- **Specifics** — More detail on the likely implementation.
- *Speed* — fast enough to run on every write.
- *Markdown content* — required sections, naming conventions, templates, etc.
```

Becomes:

```
- **Specifics:** More detail on the likely implementation.
- *Speed:* fast enough to run on every write.
- *Markdown content:* required sections, naming conventions, templates, etc.
```

Rule: move the colon inside the bold/italic so the term reads as a label.

### 2. Numeric or date range (mechanical)

Trigger: dash between two endpoints (`1900—2100`, `pp. 3—9`). Rare in this
repo today, but include it for completeness.

Action: `1900-2100`, or "1900 to 2100" if a hyphen could be misread.

### 3. Generated-doc text (mechanical, but fix upstream)

About 119 of the 405 em dashes live under
`docs/content/reference/check-types/` and `.../inspectors/`, which are
generated. They come from `cmd/gendocs` templates and registry descriptor
strings. Examples of the source:

```go
fmt.Fprintf(&b, "- [%s]({{< relref \"%s.md\" >}}) — %s\n", d.Title, d.Slug, plain(d.Summary))
fmt.Fprint(&b, "**Scope:** collection — runs once per collection over all its items.\n\n")
```

Action: edit the template/descriptor (apply category 6 or 7 to the literal),
then `make docs-gen`. Never hand-edit a generated page; CI (`docs-gen-check`)
would revert it.

### 4. Code literal / UI glyph

The em dash is data, not prose. Two real cases:

```go
// cmd/check_types.go — table "no value" placeholder
def = "—"
// cmd/check_types_test.go — the test that asserts the placeholder
if strings.Count(line, "—") < 2 { ... }
```

Action: change the glyph to `"-"` and update the asserting test in the same
commit so they stay in sync.

### 5. Paired parenthetical (rule-based)

Trigger: two em dashes on one line fencing a clause that can be lifted out.

Real examples and the rule:

- No internal commas -> use commas:
  `The name — not the path — is the stable handle`
  -> `The name, not the path, is the stable handle`
- Internal commas (commas would be ambiguous) -> use parentheses:
  `whoever reads the evidence — a human or an agent — not the tool`
  -> `whoever reads the evidence (a human or an agent), not the tool`

### 6. Single aside that explains or lists (rule-based)

Trigger: one em dash; the right side defines, restates, or gives examples of
the left side. The signal: you could insert "namely" or "that is."

Real examples:

- `The output is evidence — counts` (`internal/inspect/inspect.go`)
  -> `The output is evidence: counts`
- `a directory of markdown — a vault, a docs tree, a knowledge base`
  -> `a directory of markdown: a vault, a docs tree, a knowledge base`

Action: colon.

### 7. Single contrastive aside (rule-based)

Trigger: one em dash; the right side qualifies or contrasts the left, usually
introduced by `but`, `not`, `yet`, `never`.

Real examples:

- `rely on it for anything important yet — but I'd genuinely love your feedback`
  -> `...important yet, but I'd genuinely love your feedback`
- `counts and distributions — never recommendations`
  -> `counts and distributions, never recommendations`

Action: comma.

### 8. Two independent clauses (judgment)

Trigger: the em dash joins two clauses that each stand alone, and the sentence
is already long. A comma would splice; a colon implies explanation it may not
carry.

Real example:

- `The progression isn't arbitrary — each tier is driven by a class of query`
  -> two sentences: `The progression isn't arbitrary. Each tier is driven by a
  class of query.`

Action: split into sentences, or rewrite. This is the only category that
routinely needs an author's eye; do not automate it.

## Similar characters (apply opportunistically when found)

| Character | Treatment |
|-----------|-----------|
| en dash `–` | hyphen `-` (category 2) |
| curly quotes `“ ” ‘ ’` | straight `" '` |
| ellipsis `…` | `...` |
| non-breaking space | normal space |
| decorative emoji | drop, except a sanctioned leading callout icon (the warning sign `⚠️`) |

## What to leave alone

- **Legitimate typography that carries meaning:** arrows (`→`, `↔`), math
  signs (`≥`, `≤`, `×`), the middle dot, angle quotes. These are not tells.
- **Accented letters in proper names:** `Diátaxis`, author names.
- **Em dashes inside fenced or inline code** that show literal content, and
  em dashes in quoted external text where fidelity matters.
- The sanctioned `⚠️` callout icon.

## Rollout notes (when we decide to apply this)

1. Reverse the em-dash guidance in `.cursor/skills/write-docs/SKILL.md` and the
   `write-domain-model` skill so the convention matches the rubric.
2. Apply categories 1-4 first (mechanical, high confidence); they likely cover
   the majority of the 405 occurrences.
3. Work categories 5-7 with the rule, spot-checking.
4. Hand-review category 8.
5. Regenerate the reference (`make docs-gen`) after touching gendocs/registry.
6. Re-run `katalyst check`; the `markdown_writing_tells` warning count is the
   progress meter.

## Decisions

- Category 1 uses a colon (`**Term:** gloss`), not a spaced hyphen. Resolved.
- `markdown_writing_tells` stays wired into the docs dogfood. The sweep took
  the warning count from 319 to ~13, all of which are intentional: the `⚠️`
  callout icons, em dashes and ellipses inside code blocks, and a few
  legitimate words. Resolved.

## Open questions

- A smarter classifier (issue #57) could collapse categories 5-8 into a single
  judged decision; worth it before the next manual pass?
- The judgment categories were applied to most, not every, instance. A later
  pass (human or classifier) can revisit the remainder.
