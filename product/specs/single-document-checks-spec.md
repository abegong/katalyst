# Single-document checks — Tiers 1–2

> **Status: planning.** Expands the markdown check family with deterministic
> checks that read one document. Two sub-classes: structural checks that round
> out today's syntactic coverage, and *derived* checks that recompute a value
> from the body and compare it to frontmatter. No engine changes — every check
> is a pure function of one `Document`.

## Overview

Today's markdown family answers questions about a single file's shape: is there
an H1, do heading levels step by one, does a fenced block name its language.
This spec keeps that envelope — one file in, violations out — and grows two
kinds of check inside it:

- **Structural** checks extend the existing syntactic coverage (heading case,
  duplicate headings, empty links, image alt text, trailing whitespace).
- **Derived** checks recompute a value from the body and assert a relationship
  to a frontmatter field — a `word_count` field matching the actual body, a
  `toc` list matching the real heading tree.

Both run as `checks.Check` implementations with no new context. The line between
them is conceptual, not architectural: a derived check is a structural check
whose expected value comes from frontmatter instead of a config literal.

## Value

The derived class closes a gap the current engine can't express: *drift between
metadata and body*. A `reading_time:` or `word_count:` field is a lie the
moment the body changes; nothing today catches it. Structural checks are table
stakes for a markdown linter and the cheapest way to broaden adoption — each is
a few lines against the existing line scanner.

## Current state

The engine ships 18 checks in three families (`internal/checks/`,
[domain model](../../docs/content/explanation/domain-model.md)).
A check is `Run(Context) []Violation`; `Context` carries `FilePath`, the parsed
`*frontmatter.Document`, and `Meta` (`internal/checks/checks.go`). Markdown
checks read `ctx.Doc.Body` and `ctx.Meta` and emit `{Path, Message, Line}`.

The body parser is line-based: `markdownLines` splits on `\n`, `heading`
recognizes ATX `#`-prefixed headings only, `firstH1` scans for the first `# `
(`internal/checks/markdown.go`). It does **not** model Setext headings,
inline-formatting normalization, or — outside the fence check's own
`inFence` bookkeeping — code-block context.

Only one check today relates frontmatter to body: `markdown_title_matches_h1`.
It is rigid — H1 only, exact string equality, no `level` knob
(`markdown.go:13-59`). The brainstorm that motivated this spec flagged that
rigidity; this spec folds the fix in (see Design → Generalize title matching).

Every check kind needs a `checks.Descriptor` in `internal/checks/registry.go`;
`registry_test.go` enforces parity, and `cmd/gendocs` renders the rule
reference from it. A check cannot ship undocumented (domain-model invariant).
The `fix` command is frontmatter-only and never touches body bytes
(domain-model invariant 1, "body bytes are sacred").

## Design

### Two sub-classes, one family

Keep the `markdown` family. Do not split derived checks into a fourth family —
they share the parser, the context, and the reporting model with the existing
markdown checks. The family intro already reads "validate relationships between
frontmatter metadata and markdown body content," which is exactly what derived
checks do.

### A shared, fence-aware line iterator

The single piece of shared infrastructure this spec introduces. Several
proposed checks (heading case, no-bare-urls, trailing whitespace) must **not**
fire on text inside fenced code blocks, and the current per-check `inFence`
pattern doesn't generalize. Add one helper alongside `markdownLines`:

```go
// bodyLines returns each body line tagged with whether it sits inside a fenced
// code block, so checks can skip code without re-implementing fence tracking.
func bodyLines(body []byte, bodyLine int) []bodyLine // {Line int; Text string; InFence bool}
```

Existing checks keep working unchanged; new checks opt in. This is deliberately
not a full Markdown AST — see Rejected alternatives.

### Generalize title matching

Replace the H1-only check with a heading-matching check, keeping the old kind as
a deprecated alias so existing configs don't break:

- New kind `markdown_title_matches_heading` with fields:
  - `field` (default `title`) — the frontmatter key.
  - `level` (default `1`, accepts `any`) — which heading level to match.
  - `match` (default `exact`, accepts `case-insensitive`, `slug`,
    `strip-inline`) — the comparison mode, fixing the Setext/normalization gaps.
- `markdown_title_matches_h1` stays registered, normalizes to the new check with
  `level: 1, match: exact`, and is marked deprecated in its descriptor.

### Structural checks (catalog)

Each is a `markdown_*` kind with a registry descriptor. Fields noted where
non-obvious; all read one document.

| Kind | Enforces |
|---|---|
| `markdown_heading_case` | Heading text matches a case style. `style: sentence\|title\|lower`. |
| `markdown_no_duplicate_headings` | No two headings share text. `scope: document\|siblings`. |
| `markdown_max_heading_depth` | Deepest heading ≤ `max`. |
| `markdown_no_empty_links` | No `[]()` / empty-target links. |
| `markdown_image_alt_text_required` | Every image has non-empty alt text. |
| `markdown_no_trailing_whitespace` | No trailing spaces/tabs (outside fences). |
| `markdown_line_length` | Body lines ≤ `max` (skips fenced code and tables). |
| `markdown_no_todo_markers` | No `TODO`/`FIXME`/configurable `tokens` in body. |

### Derived checks (catalog)

Each recomputes a value from the body and asserts a relationship to
`ctx.Meta[field]`. The expected value is computed; the check fails on mismatch.

| Kind | Derives & asserts |
|---|---|
| `markdown_word_count_matches` | `field` equals body word count (± `tolerance`). |
| `markdown_reading_time_matches` | `field` equals `ceil(words / wpm)`; `wpm` default 200. |
| `markdown_toc_matches_headings` | `field` (a list) equals the heading tree (text, optionally `levels`). |
| `markdown_heading_count_matches` | `field` equals the count of headings at `level`. |

Derived checks reuse the structural parser output — `word_count` and
`reading_time` share one tokenizer; `toc` and `heading_count` share the heading
walk. Build the primitives once, layer the checks on top.

### Fixability

Many structural checks are mechanically auto-fixable (trailing whitespace,
heading case, list markers); derived checks are auto-*correctable* (rewrite the
frontmatter field to the computed value). But `fix` is frontmatter-only today,
and body-mutating fixes would breach the "body bytes are sacred" invariant.

Decision: this spec ships **check-only**. Auto-fix is deferred to its own spec
that decides whether `Check` grows an optional `Fix` capability and whether the
invariant is relaxed for an opt-in, lossless subset. Derived-field correction
(frontmatter-only) is the natural first candidate because it stays within the
current `fix` charter — call it out there, not here.

## Open questions

1. **Parser ceiling.** The fence-aware line iterator covers the checks in this
   spec. Do we commit to line-based parsing for the whole tier, or is there a
   check here (e.g. `markdown_line_length` excluding tables, nested lists) that
   already justifies adopting `goldmark`? Resolve before building the iterator —
   it's the fork in the road for Tier 1–2 *and* the structural half of every
   later tier. _Leaning: stay line-based; revisit only if a concrete check
   can't be expressed cleanly._
2. **Derived-value typing.** `word_count` in frontmatter is an int; `toc` is a
   list; `reading_time` might be `"5 min"`. Does each derived check own its
   parse/normalize step, or do we want a small shared "coerce Meta value to
   comparable" helper? Fold the answer into Design once settled.

## Rejected alternatives

- **A fourth "derived" family.** Derived checks share parser, context, and
  reporting with markdown checks; a separate family would duplicate the family
  intro and split related kinds across two doc pages for no user benefit.
- **Adopting a full Markdown AST (goldmark) up front.** Real value for a handful
  of checks, but it pulls a dependency into the hot path and rewrites every
  existing markdown check. The fence-aware iterator buys most of the correctness
  (skip code blocks) at a fraction of the cost. Revisit per Open question 1.
- **Porting markdownlint rule-for-rule.** Katalyst's value is the
  frontmatter↔body relationship, not generic markdown style. Ship the derived
  checks that only Katalyst can express, plus the structural checks that pull
  their weight — not the long tail.
