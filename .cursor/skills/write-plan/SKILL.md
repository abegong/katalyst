---
name: write-plan
description: >-
  Drafts an implementation plan in product/specs/{slug}-plan.md following the
  project's structure and prose style. Use when the user asks to "write a plan",
  "plan this out", "create a plan for X", or needs a new plan under
  product/specs/.
---

# Write a plan

Drafts a new `product/specs/{slug}-plan.md`. A plan always references a finished
spec — if the spec has open questions, resolve those first. For the full
lifecycle, see `docs/contributing/how-we-plan.md`.

## Sections

This list is the template — there is no separate template file.

- **Spec link** — Second line of the file: `> Spec: [Feature](./slug-spec.md)`.
  Always present.
- **Current State** — Specific files and their current behavior that this plan
  builds on or replaces. Cite exact paths (`cmd/…`, `internal/…`).
- **Sequencing** — A table of phases: Phase / Focus / Scope. One row per phase.
  Add a sentence after the table if the ordering needs explanation.
- **Phases** — One `### Phase N` per row. Each phase has a one-sentence **Goal**
  and numbered sub-steps. Each sub-step has a **File:** line (mark new files
  `(new)`) and what to do and why.
- **Key Files** — Table of File / Role for every file touched or created.
- **Architecture Decisions** — Table of Decision / Choice / Rationale for
  non-obvious choices. At graduation, the locked rationale folds into the
  relevant `docs/explanation/` page (there is no central decisions log).
- **Out of Scope** — What this plan explicitly defers. Prevents scope creep.

## Tests first

katalyst is TDD: new behavior arrives with a failing test (`AGENTS.md`). Make
that structural in the plan — the first phase usually scaffolds the spec's test
checklist as pending/failing tests, and later phases make them pass. CLI tests
drive the real Cobra root via `cmd.NewRootCmd()`; disk work scaffolds into
`t.TempDir()`. Each sub-step is concrete enough to execute without re-reading
the spec.

## Naming

```
product/specs/{slug}-plan.md
```

Slug matches the spec (e.g. `cli-plan.md` pairs with `cli-spec.md`).

## Prose style

Follow `.cursor/skills/write-docs/SKILL.md`. Plans are more mechanical than
specs — lean into that. Each sub-step is imperative and file-specific.

## Workflow

1. **Read the spec.** It's the source of truth for what and why. Read it in
   full. If it has open questions, stop and resolve them first.
2. **Read the codebase.** Plans cite exact paths. Read the relevant files for
   real paths, current behavior, and patterns to follow.
3. **Draft top-down.** Current State → Sequencing → Phases → Key Files →
   Architecture Decisions → Out of Scope. Fill the Sequencing table before
   expanding the Phases — it forces a clear phase structure before the detail.
4. **Write the file.** Create `product/specs/{slug}-plan.md`.
5. **Sanity check.** Every sub-step has a concrete path. Architecture Decisions
   capture choices a future reader might question. Out of Scope names anything
   adjacent that might attract scope creep.

## Reference

- Process: `docs/contributing/how-we-plan.md`
- Testing conventions: `AGENTS.md`
- Example spec to plan against: `product/specs/cli-spec.md`
- Prose guide: `.cursor/skills/write-docs/SKILL.md`
