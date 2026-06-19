# Docs reorganization

> **Status: planning.** Restructures `docs/` and `product/`. Ephemeral —
> retires when the reorg lands; durable outputs become the migrated pages, the
> rewritten process docs, and `documentation-guide.md`.

## Overview

`docs/` becomes the durable home for everything a user or contributor needs,
organized by [Diátaxis](https://diataxis.fr/). `product/` holds only in-flight
specs that retire when their branch merges. Most current `product/` content is
design rationale, not tactical delivery, so it migrates to `docs/explanation`
with cleanup and reconciliation along the way.

## Value

The two trees have drifted into contradiction (see Current State), and nothing
tells a reader which page is current. Grounding `docs/` in a typed structure,
generating reference from code, and keeping `product/` ephemeral makes the
documentation correct by construction and gives contributors and triage agents
one place to learn scope, vocabulary, and rationale.

## Current State

`docs/` is a published Hugo site (`+++` frontmatter, `{{< relref >}}`); `product/`
is internal design notes. They overlap without cross-referencing, and three
contradictions have accumulated:

- **Command names conflict.** `docs/commands.md` documents the shipped verbs
  (`validate`, `fmt`, `create/read/update/delete`, matching `cmd/`).
  `product/cli-spec.md` specifies a rebuild that renames them (`check`, `fix`,
  `item add/get/...`) and says it "supersedes" them — with no marker for which
  is current.
- **Config model conflicts.** `cmd/`, `docs/configuration.md`, and `decisions.md`
  D1 use top-level `rules:`. `cli-spec.md` replaces it with `collections:`,
  contradicting both `docs/` and its sibling `decisions.md`.
- **`product/` is stale vs. code.** `decisions-to-make.md` and `roadmap.md` omit
  the shipped CRUD commands and the 15-check engine; `domain-model.md`'s validate
  lifecycle is JSON-Schema-only and never mentions the markdown/filesystem checks
  `docs/` already documents.

`product/how-we-document.md` and `product/how-we-plan.md` already define the
current taxonomy (added in `669bd18`): `docs/` is **user-facing only**; `product/`
is the development team's home for evergreen architecture, the decision log
(`decisions.md` + `decisions-to-make.md`), `roadmap.md`, and specs under
`product/specs/`. This reorg deliberately revises that taxonomy — it makes `docs/`
serve contributors too and pulls the durable `product/` content into it (Q1).

One layout migration is half-done: `669bd18` nested specs under `product/specs/`
but left a stale duplicate at `product/cli-spec.md` (the two copies differ only in
unfixed `../` links). The canonical copy is `product/specs/cli-spec.md`; the
top-level one should be deleted.

## Design

### Audience model

Keep the four Diátaxis quadrants for the user-facing docs, plus a flat,
non-Diátaxis `contributing/` area for project and process records (roadmap, the
process docs, the doc guide). Contributor *conceptual* material — the domain model, the general model,
connectors — lives in `explanation/`, which serves both audiences.

`AGENTS.md` stays out of `docs/`. Root and co-located per-package `AGENTS.md`
files remain the home for code-writing conventions — the one part of the old
`product/`-centric split this reorg keeps (Q1).

### `docs/` layout

```
docs/
  _index.md                    # portal: "use it" vs "contribute" entry points
  tutorials/                   # learning-oriented, guided
    getting-started.md
  how-to/                      # task-oriented
    configure-rules.md
    add-a-schema.md
    validate-in-ci.md
  reference/                   # information-oriented, generated where possible
    cli/                       # GENERATED from Cobra
    configuration.md
    rules/                     # GENERATED from the checks registry
    glossary.md                # from the domain-model Vocabulary table
  explanation/                 # understanding-oriented, the "why"
    manifesto.md
    general-model.md
    progressive-operations.md
    domain-model.md
    domain-model-mapping.md
    connectors.md              # labeled "future / not shipped"
    technical-spec.md
  contributing/                # project + process records, not Diátaxis
    documentation-guide.md     # guidelines + decision tree + templates index
    how-we-document.md         # the taxonomy (rewritten)
    how-we-plan.md             # spec lifecycle (rewritten)
    roadmap.md                 # published (Q3)
```

### Migration map

| Current file | Destination | Action / reconciliation |
|---|---|---|
| `docs/getting-started.md` | `tutorials/getting-started.md` | Expand quickstart into a guided lesson |
| `docs/commands.md` | stays (out of scope) | CLI reference + Cobra generation land with the rebuild |
| `docs/configuration.md` | `reference/configuration.md` | Describe shipped `rules:` surface; lift steps into `how-to/` |
| `docs/rules/*` | `reference/rules/` | Generate from the checks registry |
| `docs/manifesto.md` | `explanation/manifesto.md` | Move as-is |
| `docs/technical-spec.md` | `explanation/technical-spec.md` | Trim overlap with `general-model`/`roadmap` |
| `product/general-model.md` | `explanation/general-model.md` | Move as-is |
| `product/progressive-operations.md` | `explanation/progressive-operations.md` | Move as-is |
| `product/domain-model.md` | `explanation/domain-model.md` (+ Vocabulary → `reference/glossary.md`) | Fold in the 15-check engine |
| `product/domain-model-mapping.md` | `explanation/domain-model-mapping.md` | Move as-is |
| `product/connectors.md` | `explanation/connectors.md` | Move; label "future, not shipped" |
| `product/decisions.md` | distributed into `explanation/` | No ADR log; fold each decision's rationale into its topic page (reconcile D1/D2 to cli-spec) |
| `product/decisions-to-make.md` | GitHub issues / in-flight spec | Open questions become issues; directional answers fold into `explanation/` |
| `product/roadmap.md` | `contributing/roadmap.md` (published) | Reconcile to cli-spec surface |
| `product/how-we-document.md` | `contributing/` (rewritten) | Encodes the new taxonomy |
| `product/how-we-plan.md` | `contributing/` (rewritten) | New decision-log path + graduation targets |
| `product/cli-spec.md` (top-level) | — | Delete; stale duplicate of the canonical copy |
| `product/specs/cli-spec.md` | stays | Canonical; retires when the `check`/`fix` rebuild ships |

### `product/` as spec staging

`product/specs/{slug}-spec.md` per `write-spec`, with a `Status` lifecycle of
`planning` / `implementing` / `done` / `shelved`. A branch is not done until its
docs are reconciled: reference content regenerates into `reference/`, decision
rationale merges into the relevant `explanation/` page, and the spec file is
deleted. `cli-spec.md` is the live case — it dissolves when the `check`/`fix` +
`collections:` rebuild merges.

### Generated reference

1. **CLI ← Cobra (deferred).** `cobra/doc.GenMarkdownTree` writing
   `reference/cli/*.md` lands with the rebuild, not this reorg — see the
   Reconciliation policy.
2. **Rules ← checks registry.** A generator walks the kinds registered in
   `internal/checks/` and emits one `reference/rules/<kind>.md` each. CI fails if
   a registered check has no page, so a new check can't ship undocumented.
3. **Examples ← tests.** Example commands and outputs in tutorials/how-to come
   from `testscript`/txtar files under `cmd/testdata/`, CI-verified (Q4). The
   harness is a **separate PR after the `check`/`fix` rebuild**; until then
   examples are hand-written against the cli-spec surface.

### Content guidelines and templates

`contributing/documentation-guide.md` carries the "where does this go?" decision
tree, a `product/specs/` template, and a glossary/style sheet seeded from the
`domain-model.md` Vocabulary table. Of the four quadrant templates (each with the
Diátaxis "this page IS X, is NOT Y" guardrail), phase 1 ships only **reference and
explanation**; the tutorial and how-to templates are derived from the first real
page of each type (Q5).

### Reconciliation policy

`product/specs/cli-spec.md` is authoritative for *direction* — `check`/`fix`,
`item`/`collection`, `collections:`. But this reorg is **decoupled from the
rebuild** (Q2 follow-up): surface-specific docs describe **shipped reality**
(`validate`/`fmt`, `rules:`, the shipped 15-check engine), and reconciling them to
the cli-spec surface is the rebuild branch's job, not this one's. Where an
explanation page must mention the future surface, it points at `cli-spec.md` and
marks it planned. `domain-model.md` still gains the 15-check engine it omits —
that engine is shipped.

**CLI reference is out of scope.** `reference/cli/` and the Cobra generator land
with the rebuild, which regenerates them at the new surface. `docs/commands.md`
stays put until then. This keeps the structural reorg independent of the rebuild.

### Decision rationale has no central log

No ADR folder, no `decisions.md`. The *why* behind a choice lives on the
`explanation/` page for its topic — Diátaxis puts understanding there — written
into the prose, not a separate ledger. The historical role `decisions.md` played
(rejected alternatives, "why not X") moves there too: the explanation page notes
the prior approach and why it changed.

The four current decisions reconcile as:

- D1 (config format/location) and D2 (schema precedence) → the configuration
  explanation, describing the shipped `rules:` surface and its rationale; the
  rebuild updates it to `collections:`/`check` when that ships.
- D3 (no auto-`fix` value injection) and D4 (opinionated `fix`) → the `fix`
  command's explanation.

Open questions get no standing file: while a change is in flight they live in its
`product/specs/` spec; otherwise they are GitHub issues. `decisions-to-make.md`
dissolves that way.

### Process-doc and skill rewrites

The taxonomy lives in `product/how-we-document.md` and `product/how-we-plan.md`,
and the skills encode it. Revising the taxonomy means rewriting all of them in the
same change:

- **`how-we-document.md`** — its "where each kind of doc lives" matrix asserts
  `docs/` is user-only and `product/` owns architecture, decisions, and roadmap.
  Both flip under this reorg.
- **`how-we-plan.md`** — drops the "record locked choices in `decisions.md` with a
  D-number" step; resolutions fold into the relevant `explanation/` page instead,
  and the graduation targets change accordingly.
- **`write-spec`, `write-docs`, `add-katalyst-rule`** — drop their
  `product/decisions.md` references, point process/taxonomy links at
  `docs/contributing/`, and point the rules-reference link at
  `docs/reference/rules/`.

These rewrites are in scope, not follow-up: leaving them stale recreates the drift
this reorg removes.

## Open Questions

_None — all resolved._ For the record:

- **Q1 — Overturn `docs/` = user-only:** yes. `docs/` serves users and
  contributors; durable `product/` content moves in; `product/` keeps only
  ephemeral specs. `AGENTS.md` (root + co-located) stays the code-conventions
  home. `how-we-document.md`, `how-we-plan.md`, and the three skills are rewritten
  to match.
- **Q2 — Rebuild naming:** `cli-spec.md` is authoritative. Reconcile `docs/`,
  the D1/D2 rationale, and `domain-model.md` to `check`/`fix`, `item`/`collection`,
  and `collections:`.
- **Q3 — `contributing/` published:** yes, rendered on the Hugo site.
- **Q4 — Tests-as-docs:** example commands/outputs come from CI-verified
  testscript/txtar tests; the harness is a separate PR after the rebuild;
  hand-written until then.
- **Q5 — Templates:** ship the minimal subset (reference + explanation) and derive
  the rest from the first real page of each type.
- **ADR log:** removed. Decision rationale lives distributed in `explanation/`;
  open questions are issues or in-flight specs.

## Rejected alternatives

- **Two parallel audience trees** (`users/` and `contributors/`, each with its own
  quadrants). Duplicates navigation and forces every page to pick a side;
  rejected for one shared Diátaxis tree plus a flat `contributing/`.
- **A central decision log** (`decisions.md` or a `contributing/decisions/` ADR
  folder). Rejected: rationale drifts from the topic it explains and readers must
  cross-reference. The "why" lives on each topic's `explanation/` page instead,
  where the reader already is.
- **Folding the roadmap into a Diátaxis quadrant.** It is a project record, not one
  of the four documentation types; forcing it in distorts the taxonomy. It lives in
  `contributing/`.
