+++
title = "How we document"
weight = 10
+++

# How we document

Like most knowledge bases, Katalyst's documentation must balance two objectives:

- Users, agents, and contributors can easily find what they need.
- Content is comprehensive, has no internal contradictions, and is always up to date.

We do this by dividing documentation across four areas, each with a specific purpose.

- **Published documentation (Hugo):** everything users and agents need to
  understand and use Katalyst - tutorials, how-to, reference, and deep-dives explaining the reasoning behind them.
- **Go doc comments:** code-level API and design detail, co-located with the code
- **`AGENTS.md` files:** conventions, required patterns, and gotchas for writing
  code in the repo, plus a pointer to each subsystem's architecture deep-dive - a
  lean root file plus per-package files.
- **Specs and plans:** `product/` staging for in-flight ideas;
  each is deleted once its durable content graduates into the homes above.

By clearly delineating when and where to update documentation, this approach lets us minimize duplication and the risk of content drift, while still serving a wide variety of needs. That said, some overlap in content is unavoidable. Some judgement calls about what information belongs where will always be necessary. Making these judgement calls is up to the project maintainers.

## Four types of documentation

### 1. Published documentation (Hugo)

The durable home for everything a user needs, organized by
[Diátaxis](https://diataxis.fr/) plus a flat `contributing/` area:

- **`how-to/`:** task-oriented recipes.
- **`reference/`:** information-oriented lookup: configuration, the
  generated check-type reference, the glossary, the command surface.
- **`deep-dives/`:** understanding-oriented "why" (the Diátaxis *explanation*
  quadrant): the vision and scope, the domain model, the storage layer,
  progressive operations, and **design rationale at the behavioral altitude** -
  any *why* a user can observe, whatever subsystem it touches. A short **Why
  Katalyst?** orientation page sits at the top level. The narrower *why* that
  only matters once you are reading a package's code lives with that code (see
  the Go doc comments / `AGENTS.md` home below), not a per-feature explanation
  page that will drift.
- **`contributing/`:** project and process records (this file,
  [How we plan]({{< relref "how-we-plan.md" >}}),
  [How we test]({{< relref "how-we-test.md" >}}), and the page templates). Not
  a Diátaxis quadrant.

### 2. Go doc comments

Code-level API and symbol documentation lives in the code as Go doc comments,
not in the Hugo tree. Keep the design narrative here **minimal** - a short
orienting comment, with a dedicated `doc.go` only when a package needs a brief
tour (`internal/inspect/doc.go` is the worked example); it surfaces in `go doc`.
A package's architecture and design rationale do **not** live here: they go in
the subsystem's `deep-dives/` page, which the package's `AGENTS.md` points to.
Use godoc headings (`# Heading`), prose, and short lists; tables and diagrams
belong in the deep-dive or the reference.

### 3. `AGENTS.md` files

Conventions for anyone *writing code* in the repo, plus a **pointer** to the
subsystem's architecture deep-dive. **What goes here:** naming conventions,
required patterns, gotchas, the *why* behind a code constraint, and a link to
the package's `deep-dives/` page. **What doesn't:** the architecture narrative
itself (→ `docs/deep-dives/`), user-facing usage (→ `docs/`), the behavioral
*why* a user can observe (→ `docs/deep-dives/`), or API-level symbol detail
(→ Go doc comments).

Katalyst keeps a **root `AGENTS.md`** plus co-located per-package files where a
package has rules of its own or a deep-dive worth pointing at. Keep each file
concise and to the point. Examples live in tests, not a separate examples file:
a `*_test.go` is the canonical, executable example.

### 4. Specs and plans

`product/specs/{slug}-spec.md` and `-plan.md` for changes **not yet
merged**. A spec is deleted when its work lands and its durable content
graduates into `docs/`. Nothing evergreen lives in `product/`, it is
staging, not a home. See [How we plan]({{< relref "how-we-plan.md" >}}).

## Generated reference

Check-type pages under `docs/reference/check-types/` are **generated** from the
checks registry (`internal/checks/registry.go`) by `cmd/gendocs`. Do not edit
them by hand, run `make docs-gen` and commit the result. CI fails if a
registered check type has no page, so a new check type cannot ship
undocumented. To add a check type, see
[add-katalyst-check-type](../../.cursor/skills/add-katalyst-check-type/SKILL.md).

Worked examples (small corpora run through a real `katalyst` command) are also
generated, from the example registry, and embedded into the pages that own each
feature. They are tests that double as documentation; see
[How we test]({{< relref "how-we-test.md" >}}) for the mechanism.

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

- **Keep `AGENTS.md` concise:** conventions plus a pointer to the architecture deep-dive, not the architecture itself.
- **Don't repeat root standards** in co-located docs; document only what's
  specific to that location.
- **Update docs in the same change** that establishes a convention or ships
  a feature; for a check, regenerate the reference.
- **Vocabulary is shared.** Use the [glossary]({{< relref "../reference/glossary.md" >}})
  as the source of truth (frontmatter vs. metadata, schema vs. validator,
  collection, item, check) across code, docs, and user-facing copy.
- **Match the existing pages'** TOML `+++` frontmatter and `{{</* relref */>}}`
  cross-links.

## Watch for AI-writing tells

The [`markdown_writing_tells`]({{< relref "../reference/check-types/markdown-body-text/writing-tells.md" >}})
check surfaces likely "AI slop": decorative punctuation, overused words, and
stock phrases, as **warnings**: it reports each hit but never fails the run.
It is a review aid, not a gate; many hits are fine in context, and there is no
allow list. A hit is a prompt to look, and the fix for each is a judgment call.

The docs collection runs it (see `.katalyst/collections/pages.yaml`), so
`katalyst check` prints each tell as a `warning:` line and still exits 0. How
to act on a flagged em dash (and which conventions to keep) is being worked out
in the em-dash rubric draft under `product/`.

## Tool-specific files

`AGENTS.md` is the source of truth for code-writing conventions, and points to
the architecture deep-dives. Other tools read their own files; keep them thin
and pointed at `AGENTS.md`.

- **`.cursor/skills/`:** reusable skills (e.g. `add-katalyst-rule`). Skills
  are *actions*, not conventions; conventions stay in `AGENTS.md`.
- **`.claude/`:** Claude Code local settings, not a documentation source.

## Building and deploying docs

How the site is **built, previewed, and published** (the publish/preview/
validate split, the GitHub Pages "Actions" source invariant) is infra detail,
so it lives next to the workflow files in
[`.github/workflows/AGENTS.md`](https://github.com/abegong/katalyst/blob/main/.github/workflows/AGENTS.md),
not in this user-facing tree. Read it before touching `deploy-docs.yml`.
