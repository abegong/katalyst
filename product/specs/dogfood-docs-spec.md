# Dogfood katalyst on its own docs

> **Status: planning. Blocked on #40.** Configure a katalyst project over
> `docs/content/` and enforce it in CI, so the Hugo docs corpus is validated by
> the tool it documents (#28). Doing so requires the docs to first be
> *consistent* — the reorganizations (Explanation → Deep dives, the inspect
> series) left orphans, broken cross-links, and overlaps — so the cleanup pass
> (#29) is the precondition, not a separate effort. This spec covers both as one
> change. **It cannot start until #40 (TOML/JSON frontmatter support) merges**,
> because katalyst cannot read the docs' TOML frontmatter today.

## Overview

Katalyst validates structured markdown with frontmatter. The Hugo docs under
`docs/content/` *are* structured markdown with frontmatter — the most
representative corpus we own. This spec stands up a `.katalyst/` project over
`docs/content/`, wires `katalyst check` into the docs CI so frontmatter,
naming, and section-structure drift fail the build, and first cleans the corpus
so enforcement can be turned on against a green tree. Dogfooding is the forcing
function: turning the tool on its own docs surfaces both doc inconsistencies and
katalyst's own gaps, which graduate into follow-up issues.

## Value

- **Real-corpus validation.** A self-hosted corpus exercises the object,
  markdown, and filesystem check families against content that changes every
  PR — far more signal than `testdata/` fixtures, and it catches katalyst
  regressions the moment they break a real document.
- **Docs stay consistent as they grow.** Once `check` gates the docs build, a
  page that drops `title`, mis-cases a filename, or skips a section
  `_index.md` fails CI instead of merging. The Diátaxis structure stops
  eroding on its own.
- **Gap discovery.** The first honest attempt to point katalyst at `docs/`
  immediately surfaces a blocking gap (TOML frontmatter; see Current State)
  and the limits of the current check set. That is the dogfooding payoff —
  each gap becomes a tracked issue.

## Current State

### katalyst cannot parse the docs today (blocking)

`internal/frontmatter/frontmatter.go:13` states it plainly: *"Only YAML
frontmatter is supported today. TOML and JSON frontmatter [are not]."* The
parser keys on `---` fences (`format.go:35`). Every page under `docs/content/`
opens with a **TOML `+++` fence** (`title = "Configuration"`,
`weight = 10`). So `check` cannot read a single doc page as-is — this must be
resolved before any collection is configured. See Open Question 1.

### The config artifact is a `.katalyst/` directory, not a file

Issue #28 asks for "a `katalyst.yaml`", but that terminology is stale. The
loader (`internal/config/config.go:30`) discovers a **`.katalyst/` directory**
(`const Dir = ".katalyst"`) by walking up to the nearest ancestor that
contains it, then reads:

- `.katalyst/config.yaml` — optional; `query:` defaults and per-kind discovery.
- `.katalyst/schemas/<name>.{json,yaml}` — one file per named schema.
- `.katalyst/collections/<name>.yaml` — one file per named collection.

The reference page (`docs/content/reference/configuration.md`) is itself a
casualty of this migration: its top describes "a single `katalyst.yaml`" while
its own `query` section (lines 82–104) shows `.katalyst/config.yaml` and
`.katalyst/collections/books.yaml`. `internal/config/README.md` repeats the
stale single-file description, and the package doc comment
(`config.go:13`) cites two files that do not exist
(`product/specs/project-layout-spec.md`,
`docs/content/explanation/configuration.md`). Reconciling these is part of the
Documentation updates below.

### The corpus is inconsistent (the #29 work)

A survey of `docs/content/` found:

- **Stale `explanation/` quadrant.** `docs/content/explanation/` holds only
  `domain-model.md`, with no `_index.md` and no nav entry.
  `how-we-document.md` is explicit that **`deep-dives/` is the Diátaxis
  *explanation* quadrant** — so `explanation/` is a leftover from the
  Explanation → Deep dives rename (#17).
- **Three broken `relref` shortcodes:** `explanation/domain-model.md:15`
  (`configuration.md` → `../reference/configuration.md`),
  `explanation/domain-model.md:163` (`connectors.md` →
  `../deep-dives/connectors.md`), and `reference/inspectors/_index.md:10`
  (`commands.md` → `../commands.md`).
- **Dangling spec references.** `product/specs/inspect-spec.md`,
  `inspect-plan.md`, and `filesystem-checks-{spec,plan}.md` link to
  `docs/content/explanation/general-model.md` and `.../technical-spec.md`,
  neither of which exists.
- **domain-model vs core-concepts overlap.** `explanation/domain-model.md`
  (katalyst-specific architecture) and `deep-dives/core-concepts.md` (general
  theory, marked *work in progress*) cover adjacent ground; #29 calls for one
  canonical home per topic.
- **WIP / future pages shipping unmarked-as-draft.** `deep-dives/core-concepts.md`
  ("work in progress") and `deep-dives/connectors.md` ("future — not shipped")
  ship in the build with status only in body prose.

### Frontmatter conventions are already near-uniform

The corpus is regular enough to schema with little churn: TOML `+++` fences
everywhere; `title` on all 62 pages; `weight` on all but the root `_index.md`;
`bookCollapseSection = true` on section `_index.md` files; `draft = true` only
on `contributing/templates/*`; `aliases` only on the generated check-type
reference pages; filenames 100% kebab-case. The page *types* differ enough
(content page vs section index vs generated reference vs template) that one
schema will not fit all — see Design.

### CI already builds the docs

`.github/workflows/ci.yml` has a `docs` job running `make docs-build`, which
fails on `REF_NOT_FOUND` and other broken Hugo refs. **Hugo already enforces
link/relref integrity** — katalyst should not duplicate it. What Hugo does
*not* check, and katalyst will, is frontmatter shape, filename casing, and
required section structure.

## Design

### Division of labor: Hugo checks links, katalyst checks shape

| Concern | Enforced by | Why |
|---|---|---|
| `relref`/link targets resolve | `make docs-build` (Hugo, already in CI) | Hugo owns shortcode resolution; it already fails the build |
| Frontmatter presence/type/enum | `katalyst check` (object family) | Hugo ignores unknown keys; nothing else enforces them |
| Filename casing, extension | `katalyst check` (filesystem family) | Hugo is indifferent to filenames |
| Required section structure (`_index.md`) | `katalyst check` (filesystem family) | Hugo renders without it; we want it required |

This keeps katalyst's job to what it is uniquely good at and avoids
reimplementing a Hugo-aware link checker. (A katalyst check that understands
Hugo shortcodes is a possible later gap — see Gaps — but is **not** in scope:
`filesystem_referenced_files_exist` validates *path-valued frontmatter
fields*, not inline `relref` shortcodes.)

### Frontmatter format: teach katalyst to read TOML (and JSON), don't convert the docs

The blocking gap is resolved by extending katalyst, not by changing the
corpus. **#40 adds TOML (`+++`) and JSON frontmatter support to
`internal/frontmatter`**; this spec depends on it and starts only once it
merges. The docs stay on TOML — their idiomatic Hugo format — and katalyst
gains the ability to validate the formats real corpora (Hugo, Obsidian,
Jekyll) actually use, which is the more valuable outcome than a 62-file
conversion.

The rejected alternative — convert `docs/content/` from TOML to YAML — would
have unblocked dogfooding faster but solves the problem in the wrong layer:
katalyst that can only read YAML is a weaker tool, and the conversion is pure
churn. See Rejected alternatives.

### `.katalyst/` layout over the docs

The project root is `docs/content/`, so `.katalyst/` lives at
`docs/content/.katalyst/` and `make docs-build`/Hugo ignore the dotted dir.

**One collection, one unified schema.** Ideally collections would model the
page *types* — content pages (`title` + `weight`) vs section landing pages
(`title` + `bookCollapseSection`) — because their contracts differ. The engine
can't express that today, and the reason is structural:

- A collection carries exactly **one** schema + check list, applied to every
  item in its subtree.
- `internal/project` enumerates with `doublestar` (`project.go:61`), and
  **files not matching a collection's pattern are reported as errors** — via a
  **recursive** walk of the collection's directory (`Unmatched`,
  `project.go:84`).
- doublestar globs have **no negation**, so "all `.md` except `_index.md`" is
  not one pattern, and two collections rooted at the same subtree would each
  flag the other's files as unmatched.

So a single permissive schema covers the whole tree. One collection rooted at
`docs/content/` with `pattern: **/*.md` claims every page (no unmatched
errors), and its schema asserts the **common** contract:

- `title` — required string (true of all 62 pages).
- `weight` — integer **when present** (optional, because the root `_index.md`
  has none and section indexes vary).
- `bookCollapseSection` — boolean when present; `aliases` — array of strings
  when present; `draft` — boolean when present. Extra keys allowed.
- Filesystem checks: `filesystem_name_case` (kebab), `filesystem_extension_in`
  (`md`).

The cost is that `weight` can't be *required* only on content pages — the
common shape is the strongest schema a single collection can enforce. Closing
that gap (per-page-type / per-pattern check scoping) is **#41**, filed as a
follow-up; the unified schema ships now. See Open Question 2 (resolved).

**Generated check-type pages are checked too.** The pages under
`reference/check-types/` (written by `make docs-gen`) carry `aliases` and a
"GENERATED" banner. They are claimed by the same `**/*.md` collection and must
satisfy the unified schema — drift in generated output should fail the build
like any other page. No exclusion, no dedicated collection.

### Cleanup pass (#29), landed in the same change

Enforcement is turned on against a green tree, so the cleanup is a precondition:

1. **Retire `explanation/`.** Move `domain-model.md` into `deep-dives/`
   (the documented explanation quadrant); reconcile its overlap with
   `core-concepts.md` so each topic has one home; delete the empty
   `explanation/` dir.
2. **Fix the three broken `relref`s** listed in Current State (the Hugo build
   already fails on them, so this also unbreaks the `docs` CI job).
3. **Repoint the dangling spec links** (`general-model.md`,
   `technical-spec.md`) to the surviving pages, or drop them. These live in
   `product/specs/`, which is staging that gets deleted when work lands, so
   this is lowest-priority — but the references are wrong today.
4. **Resolve WIP pages.** Either mark `core-concepts.md` / `connectors.md`
   `draft = true` (excluded from the build) or commit to their status; pick
   one so the schema can assert it rather than tolerate ambiguity.

### Wiring into CI

Add a `katalyst check` step to the `docs` job in `.github/workflows/ci.yml`,
after `make docs-build`, running the freshly built `./bin/katalyst` against
`docs/content/`. The `docs` job already has Go set up. Gate on `check`; defer
`fix --check` to a follow-up once the corpus is confirmed canonical, to avoid
turning on two gates in one change. Update `docs/content/how-to/validate-in-ci.md`
to show the docs themselves as the worked example.

## Open Questions

Resolved (folded into the design above):

- **Frontmatter format → extend katalyst, don't convert the docs.** Add TOML
  and JSON frontmatter support (#40) and keep the docs on TOML. Blocks this
  spec: #40 must merge first.
- **Heterogeneous page types under one root → one collection, unified
  schema.** The engine can't apply different schemas to `_index.md` vs content
  pages in the same subtree (one schema per collection; recursive
  unmatched-as-error; no glob negation). Use a single `**/*.md` collection
  whose schema asserts the common contract (`title` required; `weight` and the
  other keys optional). Strict per-page-type enforcement is deferred to **#41**.
- **Generated check-type pages → checked, not excluded.** They satisfy the same
  unified schema; generated drift should fail like any page.
- **Scope → one spec.** #28 (enforce) and #29 (clean up) ship together because
  the cleanup gates enforcement. If the work grows, split the *plan* into a
  cleanup phase and an enforcement phase — not into two specs.

Still open: _None._

## Documentation updates

- **`docs/content/reference/configuration.md`** — rewrite the opening from "a
  single `katalyst.yaml`" to the real `.katalyst/` directory layout
  (`config.yaml`, `schemas/`, `collections/`); make it internally consistent
  with its own `query` section.
- **`internal/config/README.md`** — same correction; it still describes the
  single-file format.
- **`internal/config/config.go:13`** — fix the package doc comment's two
  dangling references (`project-layout-spec.md`,
  `explanation/configuration.md`).
- **`docs/content/how-to/validate-in-ci.md`** — use the docs project as the
  worked CI example; the `katalyst.yaml`-not-found exit-code note should read
  `.katalyst/`.
- **`docs/content/how-to/configure-rules.md`** and **`add-a-schema.md`** —
  audit for the same single-file staleness while in the area.
- **`docs/content/reference/glossary.md`** — no new vocabulary expected; verify
  "project root" still reads correctly after the config-format fix.
- **`AGENTS.md`** — note the dogfooded `docs/content/.katalyst/` project and
  that the docs CI job runs `katalyst check`, so doc changes must pass it.
- **Generated reference** — none; this change adds no check type, so
  `make docs-gen` output is unaffected.

## Gaps surfaced (per #28)

Filed:

- **#40 — TOML and JSON frontmatter support** in `internal/frontmatter`. The
  blocking gap; this spec depends on it and starts only after it merges.
- **#41 — Per-page-type / per-pattern check scoping within a collection.** Why
  the docs ship with one unified schema instead of strict `_index.md` vs
  content-page contracts. Non-blocking.

Candidates to file as the first real `check` run surfaces them:

- **Hugo-shortcode-aware link checking** — only if we ever want katalyst (not
  Hugo) to own `relref` integrity; today the Hugo build covers it.
- **`fix --check` gate on the docs** — a second CI gate, once `fix` round-trips
  TOML (part of #40) and the corpus is confirmed canonical.
- Any other check-set limitation the run exposes (e.g. a conditional /
  "exactly one of these keys" frontmatter shape the docs need but the 18 checks
  can't express).

## Rejected alternatives

- **Convert the docs from TOML to YAML instead of teaching katalyst TOML.**
  Faster to unblock, but it solves the problem in the wrong layer: a katalyst
  that only reads YAML stays a weaker tool, and the 62-file conversion is pure
  churn with no payoff beyond this repo. Extending the parser (#40) makes
  katalyst usable on real Hugo/Obsidian/Jekyll corpora, which is the point of
  dogfooding.
- **A katalyst check for broken `relref`s.** Duplicates what `make docs-build`
  already enforces in CI. Katalyst should validate what Hugo ignores
  (frontmatter, naming, structure), not re-implement Hugo's resolver.
- **Separate collections per page type (content / section index).** The ideal —
  each contract enforced strictly — but the engine can't express it: one schema
  per collection, recursive unmatched-as-error, and no glob negation mean
  overlapping collections under one root collide. Deferred to #41; the unified
  schema is the strongest contract a single collection can carry today.
- **Enforce on a dirty tree and let CI stay red until cleaned.** Turning on a
  gate that fails on day one trains everyone to ignore it. Clean first
  (#29), then gate.
