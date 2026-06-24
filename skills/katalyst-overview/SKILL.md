---
name: katalyst-overview
description: >-
  Orientation and routing for katalyst: what it is, its model and vocabulary
  (projects, collections, items, schemas, checks), and which katalyst skill to
  use for a goal. Use when a user mentions katalyst and you need to know what it
  does or where to start, when content needs structure/consistency enforcement,
  or to pick among the katalyst task skills. Does no task work itself — it routes.
---

# Katalyst overview

Katalyst is a content-consistency layer for a body of markdown (and, later, other
storage). It lets you declare the structure your content should have and checks
that it holds — so a knowledge base stays consistent as people and agents add to
it. This skill explains the model and points you at the skill for the job; it
does not do the work itself.

## Model and vocabulary

- **Project** — the whole workspace: a repo root with a `.katalyst/` config
  directory that declares everything below. `katalyst init` creates it.
- **Collection** — a named group of like items (a directory plus a filename
  pattern), owning the checks its items must pass.
- **Item** — one unit of content in a collection (one markdown file, in the
  filesystem backend), addressed as `<collection>/<item>`.
- **Schema** — the declared shape of a collection's items (JSON Schema today):
  the fields they must have and the constraints those fields obey.
- **Check** — one configured constraint run against items; a failed check is a
  **violation**. `katalyst check` runs them; `katalyst check-types list` shows
  what kinds exist.

The CLI surface: `init` (scaffold), `inspect` (profile what exists), `check`
(run checks), `fix` (canonicalize frontmatter), and `collection`/`item`/`schema`
subcommands to explore config and content.

## The workflow, and which skill to use

Katalyst spans a workflow. Match the user's goal to a skill:

| Goal | Skill |
|---|---|
| Take stock of existing content, see what's there | **katalyst-catalog** |
| Name the collections the content is made of | **katalyst-identify-collections** |
| Define each collection's schema and checks | **katalyst-define-schemas** |
| Make checks run automatically from now on | **katalyst-deploy** (routes to the hook or CLI-gating setup) |
| Migrate content when a schema or storage changes | *katalyst-migrate-schema / katalyst-migrate-storage (not yet available)* |

Typical first-time path: catalog → identify-collections → define-schemas →
deploy. A user who already has a `.katalyst/` config and just wants enforcement
can go straight to **katalyst-deploy**.

## Prerequisites for the task skills

Every task skill needs the `katalyst` CLI on `PATH`. If it is missing, run the
bundled bootstrap (`./bootstrap.sh`) to install it from the latest release.
