# Dogfood katalyst on its own docs

> **Status: planning.** Configure a katalyst project over `docs/content/` and
> enforce it in CI, so the Hugo docs corpus is validated by the tool it
> documents (#28). Doing so requires the docs to first be *consistent* — the
> reorganizations (Explanation → Deep dives, the inspect series) left orphans,
> broken cross-links, and overlaps — so the cleanup pass (#29) is the
> precondition, not a separate effort. This spec covers both as one change.

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

### Frontmatter format: standardize the docs on YAML

Resolving the blocking gap (Current State) two ways:

1. **Convert `docs/content/` frontmatter from TOML `+++` to YAML `---`.** Hugo
   supports YAML frontmatter natively, so this is a mechanical, lossless
   per-file change with no rendering impact. It makes katalyst's *native*
   format the docs' format, which also unlocks `fix --check` (canonical
   frontmatter formatting) on the corpus.
2. **Add TOML frontmatter support to katalyst.** Broader long-term value, but
   it is a parser+formatter feature with its own round-trip `fix` semantics —
   a separate spec, and the wrong thing to block dogfooding on.

**Decision: (1) now, (2) as a follow-up gap issue.** Standardizing the docs on
YAML unblocks the whole effort immediately and keeps this spec about *config
and cleanup*, not a parser feature. Multi-format reading is the right
long-term answer for katalyst-the-tool (Obsidian/Jekyll emit TOML/JSON), so it
is filed as a gap — but the docs themselves have no reason to stay on TOML.
See Open Question 1.

### `.katalyst/` layout over the docs

The project root is `docs/content/`, so `.katalyst/` lives at
`docs/content/.katalyst/` and `make docs-build`/Hugo ignore the dotted dir.
Collections model the page *types*, because their frontmatter contracts differ
and a collection carries exactly one schema + check list:

- **`pages`** — ordinary content pages. `pattern` selects `**/*.md` excluding
  `_index.md`. Schema requires `title` (string) and `weight` (integer).
- **`sections`** — section landing pages (`pattern: **/_index.md`). Schema
  requires `title`; `bookCollapseSection` boolean when present. The root
  `_index.md` (no `weight`) is the one exception the schema must tolerate.
- Filesystem checks shared by both: `filesystem_name_case` (kebab),
  `filesystem_extension_in` (`md`).

`internal/project` enumerates with `doublestar` (`project.go:61`), so `**`
recursive patterns work and **files not matching a collection's pattern are
reported as errors** (`project.go:79`). That unmatched-as-error rule is the
sharp edge: every `.md` under a collection's directory must be claimed by some
pattern, or `check` fails. Modeling page types as overlapping `**` patterns
under one root needs care — see Open Question 2.

The generated check-type pages (`reference/check-types/`, written by
`make docs-gen`) carry `aliases` and a "GENERATED" banner. They are valid
content pages and the `pages` schema should accept them (extra keys allowed);
katalyst does not need a dedicated collection for them. See Open Question 3.

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

1. **TOML vs. YAML frontmatter.** Recommendation above is to convert
   `docs/content/` to YAML now and file TOML support as a katalyst gap. Confirm
   before the 62-file conversion lands, since it is a large (if mechanical)
   diff and the reverse churn if we later prefer TOML is real. Blocks
   everything else.
2. **Modeling heterogeneous page types under one root given
   unmatched-as-error.** Do `pages` (`**/*.md` minus `_index.md`) and
   `sections` (`**/_index.md`) cleanly partition the tree with no file left
   unclaimed, given `doublestar` semantics and per-directory collection roots?
   If overlapping `**` collections under a single root don't compose, fall
   back to one collection per top-level section (`how-to/`, `reference/`,
   `deep-dives/`, `contributing/`). Resolve by trial against the real tree.
3. **Do generated check-type pages get checked?** Folding them into `pages`
   (extra keys allowed) is simplest; the alternative is excluding
   `reference/check-types/**` so the generator stays the sole authority.
   Leaning toward checking them — drift in generated output should fail too.
4. **One spec or split at implementation.** This spec covers #28 and #29
   together because cleanup gates enforcement. If the cleanup grows, split the
   plan into a cleanup phase and an enforcement phase rather than two specs.

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

## Gaps to file as follow-up issues (per #28)

- **TOML (and JSON) frontmatter support** in `internal/frontmatter` — the
  blocking gap, deferred so dogfooding ships on YAML. Its own spec.
- **Hugo-shortcode-aware link checking** — only if we ever want katalyst (not
  Hugo) to own `relref` integrity; today the Hugo build covers it.
- **`fix --check` gate on the docs** — second CI gate, once the corpus is
  confirmed canonical.
- Any check-set limitations the first real `check` run surfaces (e.g. an
  "exactly one of these keys" or conditional-required-field shape the docs need
  but the 18 checks can't express).

## Rejected alternatives

- **Add TOML frontmatter support first, then dogfood.** Correct long-term, but
  it front-loads a parser feature to unblock a config-and-cleanup task. YAML
  conversion gets the same dogfooding today; TOML support ships on its own
  merits later.
- **A katalyst check for broken `relref`s.** Duplicates what `make docs-build`
  already enforces in CI. Katalyst should validate what Hugo ignores
  (frontmatter, naming, structure), not re-implement Hugo's resolver.
- **One catch-all collection over `docs/content/` with one schema.** The page
  types (content / section index / template) have genuinely different
  frontmatter contracts; a single permissive schema would assert almost
  nothing. Separate collections keep each contract honest.
- **Enforce on a dirty tree and let CI stay red until cleaned.** Turning on a
  gate that fails on day one trains everyone to ignore it. Clean first
  (#29), then gate.
