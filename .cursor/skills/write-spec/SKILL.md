---
name: write-spec
description: >-
  Drafts a feature or architecture spec in product/specs/{slug}-spec.md
  following the project's structure and prose style. Use when the user asks to
  "write a spec", "spec this out", "create a spec for X", or needs a new spec
  under product/specs/.
---

# Write a spec

Drafts a new `product/specs/{slug}-spec.md`. Writing a spec is iterative —
expect to go back and forth with the user to resolve open questions before it's
ready. For the full lifecycle (spec → plan → implement → graduate), see
`docs/contributing/how-we-plan.md`.

For small, well-understood changes, don't write a spec at all — a GitHub issue
capturing the decision is often enough.

## Sections

This list is the template — there is no separate template file.

- **Status** — A one-line blockquote at the top: what the spec is and its
  lifecycle status (`planning` / `implementing` / `done` / `shelved`).
- **Overview** — What and why. 2–4 sentences. Don't restate the title.
- **Value** — Why it's valuable to users or the project. Omit if self-evident
  from Overview.
- **Current State** — What exists today and what's wrong. Cite specific files
  (`cmd/…`, `internal/…`) and pain points.
- **Design** — The core decisions: domain model, architecture, command surface,
  user flows. This is the heart. Use subsections only when each earns space.
  Capture decisions that are expensive to reverse.
- **Open Questions** — Numbered unresolved decisions that block finalizing
  Design. Surface them early; fold each resolution into Design. The locked
  rationale graduates into the code it explains (package docs, with `doc.go`
  when long) or, for cross-cutting concepts, into the relevant
  `docs/deep-dives/` page when the work ships; there is no central decisions
  log. A spec isn't done until this section is empty or `_None._`.
- **Documentation updates** — The docs the work touches, listed so they land in
  the same change instead of drifting after (see
  `docs/contributing/how-we-document.md`). Cover **developer docs** — `AGENTS.md`
  (root and any per-package file), agent skills under `.cursor/skills/`, and Go
  doc comments (package docs; a `doc.go` when long) — and **user docs** — the
  Hugo pages under `docs/` (how-to, reference, deep-dives, getting-started). Name
  the specific pages and what changes on each. For a new check, regenerate the
  rule reference with `make docs-gen` rather than editing it by hand. `_None._`
  only when the change is purely internal.
- **Test checklist** (optional) — When the spec doubles as the build contract
  (as `product/specs/cli-spec.md` does), end with a checklist of behaviors the
  pending tests assert. Otherwise this lives in the plan.

### Optional sections

- **Appendix** — `## Appendix: {topic}` for content worth recording but heavy
  enough to drag the main flow inline (failure-mode enumerations, dependency
  analyses). Use sparingly; three appendices means the Design isn't doing its
  job.
- **Rejected alternatives** — Include when you spent real time weighing
  options, so coworkers and your future self don't re-litigate them. One or two
  sentences per alternative — the option and why it lost. Skip strawmen. Note:
  once the spec is retired, rejected alternatives live with the thing they
  explain: package docs (`doc.go` when long) for subsystem choices, or the
  relevant `docs/deep-dives/` page for cross-cutting concepts.

## Naming

```
product/specs/{slug}-spec.md
```

Slug is lowercase kebab-case matching the feature (e.g. `cli-spec.md`,
`connectors` would be `connector-layer-spec.md`).

## Prose style

Follow `.cursor/skills/write-docs/SKILL.md`. The short version: lead with what
the thing **is**; declarative, not hedged; short sentences; no marketing
adjectives; skip the obvious.

## Workflow

1. **Read context.** Read the relevant code and the deep-dive docs the
   feature builds on — `docs/reference/glossary.md`,
   `docs/deep-dives/core-concepts.md`, the topic's deep-dive page, and any
   related spec under `product/specs/`. Note the rationale and domain
   vocabulary the feature should extend.
2. **Draft.** Fill Status, Overview, Current State, and Design. Add Value if not
   self-evident. Add Open Questions only for genuinely unresolved decisions, and
   number each so answers can refer to it directly. List the Documentation
   updates the work will require.
3. **Write the file.** Create `product/specs/{slug}-spec.md`.
4. **Sanity check.** Design captures expensive-to-reverse decisions with
   specific files cited. No section is filler.

## What makes a great spec

A great spec is **complementary** — it builds on the existing domain model
and deep-dive pages, and shows how the new work fits what's already
there. Cite the prior rationale and patterns it extends. When a spec **diverges** from
an established pattern, it says so explicitly with a specific reason. "We're not
doing X here because Y" beats silent inconsistency.

## Reference

- Process: `docs/contributing/how-we-plan.md`
- Documentation homes: `docs/contributing/how-we-document.md`
- Example: `product/specs/cli-spec.md`
- Prose guide: `.cursor/skills/write-docs/SKILL.md`
- Vocabulary: `docs/reference/glossary.md` and `docs/deep-dives/core-concepts.md`
