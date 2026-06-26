# Domain model cleanup — plan

> Spec: [Domain model cleanup](./domain-model-cleanup-spec.md)

The work is a documentation and terminology pass. No Go identifier changes: the
code already matches the settled terms (`frontmatter.Document`, `field`,
`Source*`, the storage vocabulary), so every edit below lands in `docs/`,
`cmd/AGENTS.md`, and one relocated `cmd/` markdown file.

## Current State

- `docs/content/reference/glossary.md` — meant to be canonical but missing
  *attribute*, *field*, *operation*, *aggregate*, *validation result*, and a
  uses "Raw-source layer." Family is intentionally not a standalone term.
- `docs/content/deep-dives/domain-model/_index.md` (170 lines) — encyclopedic: a "Data
  interface" concept, a structured/unstructured "Goal," full definitions of
  item/collection/attribute/check/inspector, and an "Implications" section that
  states the operations thesis. Weight `20`.
- `docs/content/deep-dives/progressive-operations.md` — the tiered model; lede
  "How data interfaces evolve." Weight `30`.
- `docs/content/deep-dives/domain-model/_index.md` (112 lines) — the katalyst hub #73
  built; indexes the subsystem pages. Uses "Markdown document."
- `docs/content/deep-dives/domain-model/collections.md`, `inspectors.md` — new in #73; own the
  detail. `inspectors.md` uses "raw-source layer."
- `docs/content/deep-dives/domain-model/storage.md` — calls itself the realization of "the
  data interface concept."
- `docs/content/deep-dives/command-organization.md` ("How the core commands are
  organized") — referenced only from `cmd/AGENTS.md:9` and
  `deep-dives/_index.md:14`.
- `docs/content/deep-dives/{checks,collections,inspectors}.md` titled "How X
  work."
- "raw-source" appears in `internal/inspect/registry.go` (generates the inspector
  reference), `cmd/gendocs/main.go`, `cmd/inspect.go`, `cmd/inspectors.go`, and
  many comments. The Go *type* prefix is the short `Source*`. This plan keeps
  "raw-source" as the prose term, so none of that code is touched (see
  Architecture Decisions).

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Glossary canonical | Resolve conflicts, add missing entries, record the general/specific rule. |
| 2 | Slim core-concepts | Cut the dichotomy, relocate the operations thesis, reorder the ToC. |
| 3 | Align hub + subsystem pages | item/document, data-interface→storage, raw-source consistency. |
| 4 | Retitle + relocate command-organization | Plain titles; move the page into `cmd/`. |
| 5 | Verify | Dogfood `katalyst check` on docs, `make test`, build, confirm no generated-doc drift. |

The glossary leads because every later phase links to it for definitions. Phases
2–4 are independent of each other once the glossary is fixed and could land in
any order or one PR.

### Phase 1 — Glossary canonical

**Goal:** `glossary.md` defines every concept once, with the general/specific
pairs and their relationship explicit.

1. **File:** `docs/content/reference/glossary.md` — add rows: **Attribute** (a
   named characteristic of an item: a frontmatter key, but also filename/path),
   **Field** (a key in an item's structured object/frontmatter; *a field is an
   attribute, a filename is not*), **Operation**, **Aggregate**, **Validation
   result**. Do NOT add a standalone Family row: family is an organizing axis for check types, kept in the CheckLibrary row and checks.md. Alphabetize the table.
2. **File:** `docs/content/reference/glossary.md` — extend the **Document** and
   **Item** rows to state the specialization: a document is the markdown
   file-form of an item.
3. **File:** `docs/content/reference/glossary.md` — add a usage note recording
   the general/specific rule: default to the general term (*item*, *attribute*);
   use the specific term (*document*, *field*) only where the markdown-file or
   structured-object form is the subject.
4. **File:** `docs/content/reference/glossary.md` — keep **Raw-source layer** as
   is; no StorageType/data-interface change needed here (the glossary already
   uses the storage vocabulary).

### Phase 2 — Slim core-concepts, relocate the thesis

**Goal:** core-concepts becomes a general-altitude hub mirroring domain-model;
the operations thesis moves to where it is demonstrated.

1. **File:** `docs/content/deep-dives/domain-model/_index.md` — delete the "Goal"
   structured/unstructured dichotomy (duplicates `vision.md`); keep one sentence
   motivating a shared cross-backend vocabulary.
2. **File:** `docs/content/deep-dives/domain-model/_index.md` — collapse each concept
   (item, collection, attribute, operation, check, inspector) to a one-line
   intro that links to the glossary (definition) and to its discussion page;
   rename the "Data interface" concept to **Storage** and drop "data interface"
   from the prose and the examples table header.
3. **File:** `docs/content/deep-dives/domain-model/_index.md` — fix the attribute/field
   synonym line (currently "a named characteristic or field") to state the
   specialization instead; remove the "Implications" section, leaving a one-line
   pointer to progressive-operations.
4. **File:** `docs/content/deep-dives/progressive-operations.md` — add the
   relocated operations-thesis sentence (structuredness = which operations are
   supported; checks are the means) as the page's opening thesis; reword the
   "data interfaces evolve" lede to "storage backends."
5. **File:** `docs/content/deep-dives/progressive-operations.md` — set
   `weight = 20`. **File:** `docs/content/deep-dives/domain-model/_index.md` — set
   `weight = 30`. Order becomes vision → progressive operations → core concepts.

### Phase 3 — Align the hub and subsystem pages

**Goal:** the katalyst-altitude pages use the settled terms consistently.

1. **File:** `docs/content/deep-dives/domain-model/_index.md` — sharpen the one-line
   statement of how it differs from core-concepts (specific map vs general map);
   apply item/document usage; confirm "raw-source" wording matches the glossary.
2. **File:** `docs/content/deep-dives/domain-model/storage.md` — reword "the data interface
   concept" to name the deprecation explicitly or drop it; the storage
   vocabulary stands on its own.
3. **File:** `docs/content/deep-dives/domain-model/collections.md`,
   `docs/content/deep-dives/domain-model/inspectors.md` — apply item/document where the form
   is the subject; keep "raw-source layer" (consistency check only).

### Phase 4 — Retitle pages, relocate command-organization

**Goal:** plain key-term titles, and CLI-org rationale lives next to `cmd/`.

1. **File:** `docs/content/deep-dives/domain-model/collections.md` → title "Collections";
   `docs/content/deep-dives/domain-model/checks.md` → "Checks";
   `docs/content/deep-dives/domain-model/inspectors.md` → "Inspectors". Title-only;
   filenames and `relref` links are unchanged.
2. **File:** `cmd/organization.md` (new) — move the body of
   `command-organization.md` here as plain markdown: strip the Hugo `+++`
   frontmatter, convert `{{< relref >}}` shortcodes to relative links (or plain
   references), keep the two-grammar rationale and placement rule.
3. **File:** `docs/content/deep-dives/command-organization.md` — delete.
4. **File:** `cmd/AGENTS.md` — repoint the line 9 link from the deep-dive path to
   `./organization.md`.
5. **File:** `docs/content/deep-dives/_index.md` — drop the "how the core commands
   are organized" clause from the intro paragraph.

### Phase 5 — Verify

**Goal:** the docs still build, pass katalyst's own checks, and the generated
reference is unchanged.

1. Run `make docs-gen` and confirm **no diff** (registry strings untouched, so
   the check-types/inspectors reference must not move).
2. Run `make test` (help snapshots are unaffected because no help text changed)
   and `go build ./...`.
3. Run `katalyst check` over `docs/` (dogfooding) so the prose passes
   `markdown_writing_tells` and the em-dash rubric.

## Key Files

| File | Role |
|---|---|
| `docs/content/reference/glossary.md` | Canonical definitions; gains attribute/field/operation/aggregate/validation-result/family + the general/specific rule. |
| `docs/content/deep-dives/domain-model/_index.md` | Slimmed general hub; loses Goal + Implications; reweighted `30`. |
| `docs/content/deep-dives/progressive-operations.md` | Gains the operations thesis; reweighted `20`. |
| `docs/content/deep-dives/domain-model/_index.md` | Katalyst hub; terminology-aligned. |
| `docs/content/deep-dives/domain-model/storage.md` | Drops the "data interface" framing. |
| `docs/content/deep-dives/domain-model/collections.md` | Retitled "Collections"; item/document aligned. |
| `docs/content/deep-dives/domain-model/checks.md` | Retitled "Checks". |
| `docs/content/deep-dives/domain-model/inspectors.md` | Retitled "Inspectors"; item/document aligned. |
| `docs/content/deep-dives/_index.md` | Drops the command-organization clause. |
| `cmd/organization.md` (new) | CLI command-grammar rationale, moved from the deep-dive. |
| `docs/content/deep-dives/command-organization.md` | Deleted. |
| `cmd/AGENTS.md` | Link repointed to `./organization.md`. |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Scope of renames | Docs + `cmd/AGENTS.md` only; no Go identifier changes | Code already matches the settled terms (`Document`, `field`, `Source*`, storage); renaming identifiers would be churn for no clarity gain. |
| "source" vs "raw-source" | **Keep "raw-source" as the prose term**; `Source*` stays the short code identifier | Reverses the spec's earlier "source wins": "raw-source" is the dominant, clearer prose term and is baked into `registry.go`-generated reference text; flipping it would mean editing code + regenerating docs for no user benefit. Update the spec's Settled-naming line to match. |
| data interface vs storage | Storage wins everywhere, including the general level in core-concepts | The shipped code and `storage-layer-spec.md` use the storage vocabulary; one name for the backend. |
| command-organization home | `cmd/organization.md` linked from `cmd/AGENTS.md`, not inlined | `cmd/AGENTS.md` is already 118 lines and loads as agent context; a sibling file keeps the conventions lean, mirroring the prior AGENTS-points-to-page pattern. |
| Query / engine | Deferred to GitHub issues #76 (query) and #77 (engine) | Each needs separate treatment; tracked outside this branch. |

## Documentation updates

This plan *is* the documentation change; every phase is a doc edit. No
user-facing behavior changes, so there is no getting-started/how-to update beyond
the terminology sweep already covered by Phases 1–4. The generated check-types
and inspectors reference must stay byte-identical (Phase 5, step 1).

## Out of Scope

- **Query** (#76) and **engine** (#77) terminology — tracked in their own issues.
- **#39** internal/ package realignment — deferred entirely.
- **Go identifier renames** — `frontmatter.Document`, `Source*`, etc. stay.
- **Regenerating the reference with new labels** — only runs to *confirm no
  drift*, not to change output.
- **Vale / out-of-process CheckLibrary** docs — unrelated in-flight work.
