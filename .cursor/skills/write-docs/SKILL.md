---
name: write-docs
description: >-
  Tone and style guide for writing or revising any developer-facing prose in
  this repo — AGENTS.md, the deep-dive and reference docs under docs/, the
  Hugo user-docs site under docs/, the project README, skill SKILL.md files, and
  Go doc comments. Use whenever editing or adding prose to a `.md` file, writing a
  package doc comment, or when asked to "document this", "write a README",
  "update the docs", or the equivalent.
---

# Writing docs

Write like Zod, Tailwind, es-toolkit, and tRPC docs: direct, declarative, aimed
at a developer with work to do.

## Voice

- **Lead with what the thing is, in one sentence.** Not "This document
  describes…". Not "This skill is used for…".
- **Declarative, not hedged.** "Config lives at `katalyst.yaml` in the repo
  root." — not "Config typically tends to live…". Hedging words to cut:
  _typically_, _generally_, _usually_, _might_, _can be_, _tends to_. State the
  rule; note the exception once, sharply.
- **You, not we.** Address the reader directly. No editorial _we_.
- **Short sentences.** Three commas is two sentences.
- **No marketing adjectives.** _Powerful_, _robust_, _elegant_, _seamless_,
  _simple_. Strip them. Show what the thing does.
- **Skip the obvious.** Don't restate the heading in the first paragraph. Don't
  document what the code already shows. Don't pad for symmetry.
- **Link sideways.** Point at the source file or the sibling doc; don't inline
  their contents.
- **Add names sparingly.** Reuse the vocabulary in `docs/reference/glossary.md`
  before coining a term; add one only when the concept needs it. Descriptive for
  internal abstractions; evocative names reserved for user-facing features.

## Conventions

Mechanical choices — apply them consistently across every page.

- **Product vs. command.** Write the product as **Katalyst** in prose; reserve
  the code form `katalyst` for the CLI command or binary (`run katalyst check`).
  "Katalyst validates frontmatter" — not "`katalyst` validates frontmatter."
- **Em dashes.** Use the `—` character with a space on each side
  (`a term — its definition`). Don't write `---` or `--` in source.
- **Person.** Second person ("you") and declarative by default. Reserve the
  first person ("I", "my") for the About-me and other named-author sections;
  never the editorial "we" (see Voice).

## Shape

Default to this order. Cut any section that doesn't earn its tokens.

1. A one-sentence description of the thing.
2. A _quick start_ — the smallest complete example.
3. Only then: reference, patterns, gotchas, checklist.
4. A _reference_ list of canonical source files, if useful.

## Go doc comments

API docs for Go code are doc comments (`godoc`), not Markdown. Same voice: lead
with what the symbol *is*. `// Load finds katalyst.yaml by walking upward…` —
not `// This function loads the config.` Per `AGENTS.md`, comments explain
*why* (non-obvious intent, trade-offs, constraints), not *what*.

## Where each kind of doc lives

See `docs/contributing/how-we-document.md` for the full taxonomy: `AGENTS.md`
for code conventions, `docs/` for the published Hugo site (getting-started,
how-to, reference, deep-dives, contributing), `product/specs/` for in-flight
specs,
`README.md` for the overview, Go doc comments for the API. Rule reference pages
under `docs/reference/rules/` are generated — never hand-edit them.

## Exemplars

Read these if you've drifted. The snippets are TS, but the lesson is the prose
shape — language-agnostic.

### Documenting a naming convention — Tailwind's `group-*` variants

> When you need to style an element based on the state of some parent element,
> mark the parent with the `group` class, and use `group-*` variants like
> `group-hover` to style the target element […]
>
> Groups can be named however you like and don't need to be configured in any
> way — just name your groups directly in your markup […]

State the rule mechanically, then pre-empt in one sentence the single question a
reader will ask about it.

### Documenting a getting-started flow — Zod's "Basic usage"

> ## Defining a schema
>
> Before you can do anything else, you need to define a schema. […]
>
> ```ts
> const Player = z.object({ username: z.string(), xp: z.number() });
> ```
>
> ## Parsing data
>
> Given any Zod schema, use `.parse` to validate an input. […]

Structure a quick start as the linear sequence of steps a new contributor will
take — one sentence and one snippet per step, no preamble before the first
runnable example.

### Documenting a named concept — tRPC's "procedures"

> A procedure is a function which is exposed to the client, it can be one of:
>
> - a `Query` — used to fetch data, generally does not change any data
> - a `Mutation` — used to send data, often for create/update/delete purposes
> - a `Subscription` — you might not need this […]

Define the term in one sentence, list the variants in one line each, and link
out for anything that would derail the page.

### Documenting a single utility — es-toolkit's `differenceBy`

> Transforms elements of two arrays with a conversion function, computes their
> difference, and returns a new array.
>
> ```ts
> const result = differenceBy(firstArr, secondArr, mapper);
> ```

Lead with a literal description of what the function does, then the signature;
reserve any "use this when…" framing for where the reader is choosing between
siblings.

## Anti-patterns

- Opening with "This document describes…" / "In this section…" / "Welcome to…".
- A `## Overview` that paraphrases the title.
- Bullet lists where a sentence is shorter.
- Inlining source that a link to the file would replace.
- Explaining what the code obviously does, instead of *why* it does it.
- Editing generated agent files (anything under `.claude/`) instead of the
  source `AGENTS.md` and `.cursor/skills/`.

## Workflow

1. Read the surrounding docs first and match their tone and shape. Consistency
   beats ambition.
2. Draft the lede and the tightest possible quick start. Stop. Reread.
3. Cut every sentence that adds no information. Anything that stumbles when you
   read it aloud, rewrite or drop.
