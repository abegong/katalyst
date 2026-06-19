+++
title = "Documentation guide"
weight = 10
+++

# Documentation guide

This guide answers one question for every change: **where does this
documentation go?** Katalyst keeps two trees, and putting a page in the
wrong one is how they drift apart.

- `docs/` is the durable, published home for everything a user or
  contributor needs. It is organized by
  [Diátaxis](https://diataxis.fr/) plus a flat `contributing/` area.
- `product/` holds only **in-flight specs** that retire when their branch
  merges (see [How we plan]({{< relref "how-we-plan.md" >}})).

Code-writing conventions are the one exception: they live in `AGENTS.md`
(root and co-located per-package), not in `docs/`.

## Where does this go?

Walk the tree top to bottom; stop at the first match.

1. **Is it a convention for writing code in this repo?** → root or
   co-located `AGENTS.md`. Not `docs/`.
2. **Is it a record of an in-flight change** (a spec or plan for work not
   yet merged)? → `product/specs/{slug}-spec.md` and `-plan.md`. It is
   deleted when the work lands and its durable content graduates into
   `docs/`.
3. **Otherwise it is durable documentation — pick the Diátaxis quadrant by
   what the reader is doing:**

| The reader is… | Quadrant | Folder |
|---|---|---|
| learning Katalyst by doing | **Tutorial** | `docs/tutorials/` |
| accomplishing a specific task | **How-to** | `docs/how-to/` |
| looking up a fact (config keys, check kinds, terms) | **Reference** | `docs/reference/` |
| trying to understand *why* | **Explanation** | `docs/explanation/` |
| reading a project/process record (roadmap, this guide) | — (not Diátaxis) | `docs/contributing/` |

The four quadrants are distinct on purpose. The common failure is mixing
them: a reference page that drifts into a tutorial, or an explanation page
that becomes a how-to. Each [template](#templates) names what its page **is
not** to keep the boundary sharp.

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
registered check has no page, so a new check cannot ship undocumented. To
add a rule, see [add-katalyst-rule](../../.cursor/skills/add-katalyst-rule/SKILL.md).

## Templates

New reference and explanation pages start from a template under
`templates/`. Each carries the Diátaxis "this page IS X, is NOT Y"
guardrail. The templates are marked `draft = true` so the public build
excludes them; they are in-repo for contributors only.

- [Reference template](templates/reference.md)
- [Explanation template](templates/explanation.md)

Tutorial and how-to templates are derived from the first real page of each
type rather than guessed up front.

## Glossary and style

Use the project [glossary]({{< relref "../reference/glossary.md" >}}) as the
source of truth for vocabulary (frontmatter vs. metadata, schema vs.
validator, collection, item, check). Match the existing pages' TOML `+++`
frontmatter and `{{</* relref */>}}` cross-links.
