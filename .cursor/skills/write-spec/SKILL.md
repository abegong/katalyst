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
`product/how-we-plan.md`.

For small, well-understood changes, don't write a spec at all — a new entry in
`product/decisions-to-make.md` that graduates to `product/decisions.md` is often
the whole "spec."

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
  Design. Surface them early; fold each resolution into Design and record the
  locked choice in `product/decisions.md` with a D-number. A spec isn't done
  until this section is empty or `_None._`.
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
  `product/decisions.md` is also katalyst's long-term home for rejected
  alternatives once a spec is retired.

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

1. **Read context.** Read the relevant code and the `product/` design docs the
   feature builds on — `domain-model.md`, `decisions.md`, `roadmap.md`, and any
   related spec under `product/specs/`. Note the decisions and domain vocabulary
   the feature should extend.
2. **Draft.** Fill Status, Overview, Current State, and Design. Add Value if not
   self-evident. Add Open Questions only for genuinely unresolved decisions, and
   number each so answers can refer to it directly.
3. **Write the file.** Create `product/specs/{slug}-spec.md`.
4. **Sanity check.** Design captures expensive-to-reverse decisions with
   specific files cited. No section is filler.

## What makes a great spec

A great spec is **complementary** — it builds on the existing domain model,
decisions, and roadmap, and shows how the new work fits what's already there.
Cite the prior decisions and patterns it extends. When a spec **diverges** from
an established pattern, it says so explicitly with a specific reason. "We're not
doing X here because Y" beats silent inconsistency.

## Reference

- Process: `product/how-we-plan.md`
- Example: `product/specs/cli-spec.md`
- Prose guide: `.cursor/skills/write-docs/SKILL.md`
- Vocabulary: `product/domain-model.md`
