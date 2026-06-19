+++
title = "How we document"
weight = 10
+++

# How we document

This is the one page on where Katalyst documentation lives and how to add to
it. Putting a page in the wrong home is how trees drift apart, so when in
doubt, start here.

## Goals

- Users and contributors both find what they need in one published tree.
- One source of truth per topic — no drift between duplicate files.
- Reference that is correct by construction (generated from code).
- Docs live close to what they describe.

## Where each kind of doc lives

Katalyst keeps documentation in a few homes; everything durable belongs in
`docs/`.

### `docs/` — the published site (Hugo)

The durable home for everything a user **or** contributor needs, organized by
[Diátaxis](https://diataxis.fr/) plus a flat `contributing/` area:

- **Tutorials** — learning-oriented, guided lessons. A single top-level
  `getting-started.md` today; promote to a `tutorials/` section once there's
  more than one.
- **`how-to/`** — task-oriented recipes.
- **`reference/`** — information-oriented lookup: configuration, the
  generated rule reference, the glossary, the command surface.
- **`explanation/`** — understanding-oriented "why": the manifesto, the
  general and domain models, connectors, and **design rationale**.
- **`contributing/`** — project and process records (this file,
  [How we plan]({{< relref "how-we-plan.md" >}}), the
  [roadmap]({{< relref "roadmap.md" >}}), and the page templates). Not a
  Diátaxis quadrant.

### `AGENTS.md` — code-writing conventions

Rules for anyone *writing code* in the repo: commands, layout, testing
style, code style. **What goes here:** naming conventions, required
patterns, gotchas, and the *why* behind a code constraint. **What doesn't:**
conceptual explanations of how the system works (→ `docs/explanation/`),
user-facing usage (→ `docs/`), or API-level detail (→ Go doc comments).

Katalyst keeps a **root `AGENTS.md`** plus co-located per-package files where
a package has rules that don't belong at the root. Examples live in tests,
not a separate examples file: a `*_test.go` is the canonical, executable
example.

### `product/` — in-flight specs only

`product/specs/{slug}-spec.md` and `-plan.md` for changes **not yet
merged**. A spec is deleted when its work lands and its durable content
graduates into `docs/`. Nothing evergreen lives in `product/` — it is
staging, not a home. See [How we plan]({{< relref "how-we-plan.md" >}}).

### Go doc comments — code-level API docs

Package- and symbol-level documentation lives in the code as Go doc
comments, not in Markdown.

## Where does this go?

Walk top to bottom; stop at the first match.

1. **Is it a convention for writing code in this repo?** → root or
   co-located `AGENTS.md`. Not `docs/`.
2. **Is it a record of an in-flight change** (a spec or plan for work not
   yet merged)? → `product/specs/`. Deleted when the work lands and its
   durable content graduates into `docs/`.
3. **Is it package- or function-level API detail?** → a Go doc comment.
4. **Otherwise it is durable documentation — pick the Diátaxis quadrant by
   what the reader is doing:**

| The reader is… | Quadrant | Folder |
|---|---|---|
| learning Katalyst by doing | **Tutorial** | `docs/getting-started.md` (top-level; add a `docs/tutorials/` section once there's more than one) |
| accomplishing a specific task | **How-to** | `docs/how-to/` |
| looking up a fact (config keys, check kinds, terms) | **Reference** | `docs/reference/` |
| trying to understand *why* | **Explanation** | `docs/explanation/` |
| reading a project/process record (roadmap, this guide) | — (not Diátaxis) | `docs/contributing/` |

The four quadrants are distinct on purpose. The common failure is mixing
them: a reference page that drifts into a tutorial, or an explanation page
that becomes a how-to. Each [template](#templates) names what its page **is
not**, to keep the boundary sharp.

## Decision rationale has no central log

There is no `decisions.md` and no ADR folder. The *why* behind a choice
lives on the `explanation/` page for its topic, written into the prose. When
a choice supersedes a previous approach, the explanation page notes the old
approach and why it changed — that is where a reader is already looking for
"why."

Open questions get no standing file. While a change is in flight they live
in its `product/specs/` spec; otherwise they are GitHub issues.

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
