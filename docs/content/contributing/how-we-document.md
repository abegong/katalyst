+++
title = "How we document"
weight = 10
+++

# How we document

## Goals

- Users and contributors both find what they need in one published tree.
- One source of truth per topic — no drift between duplicate files.
- Reference that is correct by construction (generated from code).
- Docs live close to what they describe.

## Where each kind of doc lives

Katalyst keeps documentation in a few homes.

### `docs/` — the published site (Hugo)

The durable home for everything a user needs, organized by
[Diátaxis](https://diataxis.fr/) plus a flat `contributing/` area:

- **`how-to/`** — task-oriented recipes.
- **`reference/`** — information-oriented lookup: configuration, the
  generated rule reference, the glossary, the command surface.
- **`deep-dives/`** — understanding-oriented "why" (the Diátaxis *explanation*
  quadrant): connectors, progressive operations, and cross-cutting **design
  rationale**. The orientation pages — Why Katalyst, Core concepts — sit at the
  top level. Subsystem-specific rationale lives in per-package `README.md` files
  under `internal/`.
- **`contributing/`** — project and process records (this file,
  [How we plan]({{< relref "how-we-plan.md" >}}), and the page templates). Not
  a Diátaxis quadrant.

### `product/` — in-flight specs and plans only

`product/specs/{slug}-spec.md` and `-plan.md` for changes **not yet
merged**. A spec is deleted when its work lands and its durable content
graduates into `docs/`. Nothing evergreen lives in `product/` — it is
staging, not a home. See [How we plan]({{< relref "how-we-plan.md" >}}).

### `AGENTS.md` — code-writing conventions

Rules for anyone *writing code* in the repo: commands, layout, testing
style, code style. **What goes here:** naming conventions, required
patterns, gotchas, and the *why* behind a code constraint. **What doesn't:**
conceptual explanations of how the system works (→ `docs/deep-dives/`),
user-facing usage (→ `docs/`), or API-level detail (→ Go doc comments).

Katalyst keeps a **root `AGENTS.md`** plus co-located per-package files where
a package has rules that don't belong at the root. Examples live in tests,
not a separate examples file: a `*_test.go` is the canonical, executable
example.

### Go doc comments — code-level API docs

Package- and symbol-level documentation lives in the code as Go doc
comments, not in Markdown.

## Generated reference

Rule pages under `docs/reference/rules/` are **generated** from the checks
registry (`internal/checks/registry.go`) by `cmd/gendocs`. Do not edit them
by hand — run `make docs-gen` and commit the result. CI fails if a
registered check has no page, so a new check cannot ship undocumented. To add
a rule, see [add-katalyst-rule](../../.cursor/skills/add-katalyst-rule/SKILL.md).

## Templates

New reference and explanation pages start from a template under
`templates/`. Each carries the Diátaxis "this page IS X, is NOT Y"
guardrail. The templates are marked `draft = true` so the public build
excludes them; they are in-repo for contributors only.

- [Reference template](templates/reference.md)
- [Explanation template](templates/explanation.md)

Tutorial and how-to templates are derived from the first real page of each
type rather than guessed up front.

## Style

- **Keep `AGENTS.md` lean** — conventions, not walls of text.
- **Don't repeat root standards** in co-located docs; document only what's
  specific to that location.
- **Update docs in the same change** that establishes a convention or ships
  a feature; for a check, regenerate the reference.
- **Vocabulary is shared.** Use the [glossary]({{< relref "../reference/glossary.md" >}})
  as the source of truth (frontmatter vs. metadata, schema vs. validator,
  collection, item, check) across code, docs, and user-facing copy.
- **Match the existing pages'** TOML `+++` frontmatter and `{{</* relref */>}}`
  cross-links.

## Tool-specific files

`AGENTS.md` is the source of truth for conventions. Other tools read their
own files; keep them thin and pointed at `AGENTS.md`.

- **`.cursor/skills/`** — reusable skills (e.g. `add-katalyst-rule`). Skills
  are *actions*, not conventions; conventions stay in `AGENTS.md`.
- **`.claude/`** — Claude Code local settings, not a documentation source.
