+++
title = "How we document"
weight = 20
+++

# How we document

## Goals

- Users and contributors both find what they need in one published tree.
- One source of truth per topic — no drift between duplicate files.
- Reference that is correct by construction (generated from code).
- Docs live close to what they describe.

## Where each kind of doc lives

Katalyst has three homes for documentation.

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
  general and domain models, connectors, and **design rationale**. There is
  no central decision log; the *why* behind a choice lives on the
  explanation page for its topic.
- **`contributing/`** — project and process records (this file,
  [How we plan]({{< relref "how-we-plan.md" >}}), the
  [roadmap]({{< relref "roadmap.md" >}}), the [documentation
  guide]({{< relref "documentation-guide.md" >}})). Not a Diátaxis quadrant.

The [documentation guide]({{< relref "documentation-guide.md" >}}) has the
"where does this page go?" decision tree.

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

## What goes where (quick matrix)

| You're documenting… | It goes in… |
|---|---|
| A rule for writing code here | `AGENTS.md` (root or co-located) |
| How a subsystem works / a domain concept | `docs/explanation/` |
| Why a choice was made | `docs/explanation/` (on the topic's page) |
| How to use the CLI (lookup) | `docs/reference/` |
| How to accomplish a task | `docs/how-to/` |
| A first lesson for new users | `docs/getting-started.md` (top-level) |
| A change being designed, not yet merged | a spec in `product/specs/` |
| An open design question | a GitHub issue, or the in-flight spec |
| What a package/function does | Go doc comments |

## Generated reference

Rule pages under `docs/reference/rules/` are generated from
`internal/checks/registry.go` by `cmd/gendocs` (`make docs-gen`). Never edit
them by hand; CI fails if a registered check has no page. Adding a check
means adding its `Descriptor` — see the `add-katalyst-rule` skill.

## Guidelines

- **Keep `AGENTS.md` lean** — conventions, not walls of text.
- **Don't repeat root standards** in co-located docs; document only what's
  specific to that location.
- **Update docs in the same change** that establishes a convention or ships
  a feature; for a check, regenerate the reference.
- **Vocabulary is shared.** Use the terms in the
  [glossary]({{< relref "../reference/glossary.md" >}}) consistently across
  code, docs, and user-facing copy.

## Tool-specific files

`AGENTS.md` is the source of truth for conventions. Other tools read their
own files; keep them thin and pointed at `AGENTS.md`.

- **`.cursor/skills/`** — reusable skills (e.g. `add-katalyst-rule`). Skills
  are *actions*, not conventions; conventions stay in `AGENTS.md`.
- **`.claude/`** — Claude Code local settings, not a documentation source.
