# Filesystem checks: expanded & revised library — plan

> Spec: [Filesystem checks: expanded & revised library](./filesystem-checks-spec.md)
>
> **Status: done.** All five phases implemented in one PR; `make all` green.
> Tiers 1–2 (revised + new per-item checks) and Tier 3 (collection-scoped via a
> new `CollectionCheck` interface and a second engine pass) all shipped, with
> the rule reference regenerated and the durable content graduated into
> `AGENTS.md`, the domain model, and the glossary.
>
> **Deviations from the plan as written:**
> - Integer-bounded checks (`name_length`, `path_depth`) reuse the shared
>   `min`/`max` yaml keys but store dedicated `*int` fields (`MinInt`/`MaxInt`)
>   on `Check`, converted from the yaml `*float64` — keeping them off
>   `object_number_range`'s float `Min`/`Max`.
> - `Violation` gained a `File` field so collection-scoped violations can name
>   the offending path (the per-item reporter already knows the file).
> - The empty-checks guard now also accepts a collection that configures *only*
>   collection-scoped checks (`Collection.HasCollectionChecks`).
> - Hand-authored docs targeted `deep-dives/domain-model.md` (the page that
>   references kinds today); the `explanation/` vs `deep-dives/` question is
>   issue #17. Spec/plan kept (not deleted) pending merge.

## Current State

See the spec's Current State for the full picture. The facts that drive the
phasing:

- Checks are **configured per collection, evaluated per item** (`cmd/check.go`
  loops `res.Items`, each calling `engine.checksFor` + `checks.RunAll`). No pass
  holds a whole collection at once.
- The four folded kinds (`filename_kebab_case`, `no_spaces_in_path`,
  `filename_prefix`, `filename_matches_slug`) are referenced across exactly:
  `internal/config/config.go` (constants + `normalizeCheck`),
  `internal/checks/filesystem.go` (structs), `cmd/engine.go` (dispatch),
  `internal/checks/registry.go` (Descriptors), plus tests
  (`internal/checks/checks_test.go`, `internal/config/config_test.go`) and the
  **generated** reference pages under `docs/content/reference/rules/filesystem/`.
  Hand-authored docs that name them: `docs/content/deep-dives/domain-model.md`,
  `docs/content/how-to/configure-rules.md`,
  `docs/content/reference/configuration.md`.
- Pre-v0: the folded kinds are **removed outright**, not aliased.
- `make all` = `vet test build`; `make docs-gen` regenerates the rules reference
  from `registry.go` Descriptors (parity enforced by `registry_test.go`).
- New config-validation messages follow the error grammar now durable in
  `cmd/AGENTS.md` (lowercase, no trailing period, `%q` around kind/field
  values); check violations keep the `path:line: /pointer: message` diagnostic
  format (exempt). Use the `cmd/usage.go` helpers where a command-level error
  is raised.
- The **`rules`** command (added on main, then split into `rules list` /
  `rules show <kind>`) is a read-only view over `checks.Descriptors()` /
  `checks.Families()`. New kinds surface automatically; its tests are dynamic
  (count via `len(Descriptors())` / `familyKinds`, JSON unmarshals a field
  subset), so adding/removing kinds won't break them. The Phase 4
  `Descriptor.Scope` field flows into its output and should show in
  `rules show`.
- Docs-location caveat: main carries **both** `docs/content/explanation/` and
  `docs/content/deep-dives/`. The `deep-dives/domain-model.md` page is the one
  that references check kinds today; the docs convention treats `deep-dives/` as
  canonical. Phases 3/5 target the page that actually references kinds and
  confirm authority at graduation rather than guessing.

## Sequencing

| Phase | Focus | Shipment |
|---|---|---|
| 1 | Target×rule foundation + Tier 1 revised checks (fold 4 → `name_case`, `name_matches_field`, `name_affix`, `path_charset`) | A |
| 2 | Tier 2 per-item checks (`name_regex`, `name_length`, `path_depth`, `parent_dir_matches_field`, `referenced_files_exist`) | A |
| 3 | Docs & graduation for Tiers 1–2 | A |
| 4 | Tier 3 collection-scoped: `CollectionCheck` interface, engine second pass, `unique_filename` / `unique_field` / `index_file_required` | B |
| 5 | Docs & graduation for Tier 3 | B |

Each phase is **tests-first internally**: write the failing sub-step, then the
implementation. A single up-front "scaffold all tests" phase doesn't work in Go
— tests referencing unbuilt symbols break package compilation. Shipment A
(Phases 1–3) stands alone and can merge/release before B. **Phase 4 is gated on
resolving spec Q1** (adopt `CollectionCheck`); do not start it until that's
locked.

`referenced_files_exist` lives in Phase 2, not 4: per the spec it needs no
siblings and ships on the per-item `Check` contract, so it belongs with Shipment
A.

## Phases

### Phase 1 — Target×rule foundation + Tier 1 revised checks

**Goal:** Replace the four folded kinds with `name_case`, `name_matches_field`,
`name_affix`, `path_charset`, all built on a shared `target` resolver. Keep
`extension_in` and `parent_dir_in` unchanged.

1. **File:** `internal/checks/checks_test.go` *(edit, failing first)* — replace
   the `FilesystemFilenameKebabCase` / `FilenameMatchesSlug` /
   `FilesystemNoSpacesInPath` / `FilesystemFilenamePrefix` cases with table
   tests for the new structs:
   - `NameCase{Style, Target}` — one row per style (`kebab`, `snake`,
     `screaming-snake`, `camel`, `pascal`, `point`, `lower`) against a basename;
     a `target: parent-dir` row; a `target: path-segments` row asserting a bad
     mid-path segment **and** a bad basename both flag (inclusive).
   - `NameMatchesField{Field, Transform, Target}` — `transform: none` equals old
     matches-slug; `transform: slugify` matches a slugified `title`.
   - `NameAffix{Prefix, Suffix, Target}` — prefix-only == old behavior;
     suffix-only; both.
   - `PathCharset{Allow, Deny}` — `deny: [" "]` == old no-spaces; `allow` set
     whitelist.
2. **File:** `internal/checks/filesystem.go` — add `resolveTarget(ctx Context,
   target string) []string` returning the slice(s) a rule tests:
   `filename` → `[basename]`, `filename-ext` → `[base+ext]`, `parent-dir` →
   `[parent]`, `path-segments` → every collection-relative dir segment + the
   basename. Implement `NameCase` (per-style anchored patterns + a `lower`
   special case), `NameMatchesField` (with a `slugify` helper), `NameAffix`,
   `PathCharset`. **Delete** `FilenameMatchesSlug`,
   `FilesystemFilenameKebabCase`, `FilesystemNoSpacesInPath`,
   `FilesystemFilenamePrefix`. Keep `FilesystemExtensionIn`,
   `FilesystemParentDirIn`.
   - *Collection-relative resolution:* `resolveTarget` needs the collection
     root to compute `path-segments`. The per-item `Context` has only
     `FilePath`. Add `CollectionRoot string` to `checks.Context` (populated by
     `engine.checksFor`'s caller from `item.Collection.Path`); segments are
     `FilePath` made relative to it. Document this as the minimal Context
     addition Tier 1 forces.
3. **File:** `internal/config/config.go` — retire the four `CheckKind` constants;
   add `CheckFilesystemNameCase`, `CheckFilesystemNameMatchesField`,
   `CheckFilesystemNameAffix`, `CheckFilesystemPathCharset`. Extend `rawCheck`
   and `Check` with `Style`, `Target`, `Transform`, `Prefix`, `Suffix`, `Allow`,
   `Deny` fields. Add `normalizeCheck` cases with validation: `name_case`
   requires a known `style`; `target` (when present) is one of the four values;
   `name_affix` requires at least one of `prefix`/`suffix`; `path_charset`
   rejects setting both `allow` and `deny` (and requires one). Remove the four
   old `case`s.
4. **File:** `internal/config/config_test.go` — update fixtures/assertions that
   referenced the old kinds (the `got[12]` slug index assertion and the
   unknown-kind test). Add load-error cases for the new validation above.
5. **File:** `cmd/engine.go` — replace the four dispatch `case`s with the new
   kinds, constructing the new structs; pass `CollectionRoot` into the
   `checks.Context` built in `cmd/check.go`.
6. **File:** `internal/checks/registry.go` — replace the four Descriptors with
   the new ones (fields, `ConfigExample`s). Keep `extension_in`/`parent_dir_in`.
7. **Gate:** `go test ./internal/checks/... ./internal/config/... ./cmd/...`
   green; `registry_test.go` parity green.

### Phase 2 — Tier 2 per-item checks

**Goal:** Add `name_regex`, `name_length`, `path_depth`,
`parent_dir_matches_field`, `referenced_files_exist` — all on the per-item
`Check` contract.

1. **File:** `internal/checks/checks_test.go` *(edit, failing first)* — add:
   - `NameRegex{Pattern, Target}` — pattern is anchored `^…$`; respects `target`.
   - `NameLength{Min, Max, Target}` — enforces bounds; (config-level) requires ≥1.
   - `PathDepth{Min, Max}` — depth relative to collection root; `max: 0` ⇒ flat.
   - `ParentDirMatchesField{Field}` — parent dir == frontmatter field.
   - `ReferencedFilesExist{Fields}` — flags a frontmatter path that resolves
     nowhere; resolves **relative to the item's directory**; accepts both a
     string and a list value.
2. **File:** `internal/checks/filesystem.go` — implement the five structs.
   `ReferencedFilesExist` reads each named field from `ctx.Meta`, coerces
   string-or-`[]string`, joins each against `filepath.Dir(ctx.FilePath)`, and
   `os.Stat`s. `PathDepth` counts separators in the collection-relative path.
3. **File:** `internal/config/config.go` — add the five `CheckKind` constants;
   extend `rawCheck`/`Check` with `Pattern`, `Min`, `Max` *(reuse the existing
   `Min`/`Max` `*float64`? — `path_depth`/`name_length` want ints; add
   `MinInt`/`MaxInt` or reuse with int parsing — decide in implementation)*, and
   `Fields []string`. Add `normalizeCheck` cases: `name_regex` requires
   `pattern` (and compiles it to validate); `name_length`/`path_depth` require
   ≥1 of min/max; `parent_dir_matches_field` requires `field`;
   `referenced_files_exist` requires non-empty `fields`.
4. **File:** `cmd/engine.go` — dispatch the five new kinds.
5. **File:** `internal/checks/registry.go` — Descriptors for the five.
6. **File:** `internal/config/config_test.go` — load-error cases for the new
   validation.
7. **Gate:** `go test ./...` green; `registry_test.go` parity green.

### Phase 3 — Docs & graduation (Shipment A)

**Goal:** Reconcile durable docs for Tiers 1–2 and regenerate the reference.

1. **File:** `make docs-gen` — regenerate
   `docs/content/reference/rules/filesystem/`. The four old generated pages
   disappear; new pages appear. Verify the family `_index.md` reflects the new
   set.
2. **Files:** `docs/content/deep-dives/domain-model.md` (the page referencing
   check kinds today — confirm vs. `deep-dives/core-concepts.md` at graduation),
   `docs/content/how-to/configure-rules.md`,
   `docs/content/reference/configuration.md` — replace references to the four
   folded kinds with the `target × rule` model and the new kind names; add a
   short note on the `target` key.
3. **File:** `docs/content/reference/glossary.md` — add "target."
4. **File:** `AGENTS.md` — record the `target × rule` convention and the
   `resolveTarget` helper / `Context.CollectionRoot` addition as a gotcha.
5. **File:** `product/specs/filesystem-checks-spec.md` — tick the Tier 1/2 test
   checklist items; leave Tier 3 unticked.
6. **Gate:** `make all` green; repo grep finds no stale references to the four
   folded kinds outside historical spec notes.

### Phase 4 — Tier 3 collection-scoped checks (Shipment B)

**Gated on spec Q1.** Do not start until the `CollectionCheck` interface is
approved.

**Goal:** A second, collection-scoped check pass and the three cross-item
checks.

1. **File:** `internal/checks/collection_test.go` (new, failing first) — table
   tests over a synthetic `CollectionContext`:
   - `UniqueFilename` — two items sharing a basename flag, message names both
     paths.
   - `UniqueField{Field}` — duplicate `field` values across items flag.
   - `IndexFileRequired{Name}` — a subdirectory of items missing `_index.md`
     flags; default `name` is `_index.md`.
2. **File:** `internal/checks/collection.go` (new) — `CollectionCheck` interface
   (`RunCollection(CollectionContext) []Violation`), `CollectionContext{Root
   string; Items []ItemContext}` (`ItemContext{FilePath string; Meta
   map[string]any}`), and the three implementations.
3. **File:** `internal/config/config.go` — add the three `CheckKind` constants
   and `Name` field; `normalizeCheck` cases (`unique_field` requires `field`;
   `index_file_required` defaults `name` to `_index.md`).
4. **File:** `cmd/engine.go` + `cmd/check.go` — after the per-item loop, for
   each collection in the selection build a `CollectionContext` by **re-scanning
   the full collection** via `project.Items(collection)` (independent of the
   selector narrowing), then run its configured `CollectionCheck`s. Split the
   collection's configured checks into item vs collection at compile time
   (a `kind → scope` lookup). Emit violations through the same reporter.
5. **File:** `internal/checks/registry.go` — Descriptors for the three; add a
   `Scope` field (`item` | `collection`) to `Descriptor`, defaulting `item` for
   all existing entries.
6. **File:** `cmd/rules.go` — show `Scope` in `runRulesDetail` (the
   `rules show <kind>` readout). `rules list` and `--json` are registry-driven
   and need no change beyond the new field flowing through; add a
   `rules_test.go` assertion that `rules show` for a collection-scoped kind
   names its scope.
7. **File:** `cmd/check_test.go` (or equivalent) — assert: a single-item
   selector still re-scans the whole collection for the collection pass
   (uniqueness verdict unaffected); per-item checks still honor the selector.
7. **Gate:** `go test ./...` green; parity green.

### Phase 5 — Docs & graduation (Shipment B)

1. **File:** `make docs-gen` — regenerate the reference; generated pages note
   `Scope: collection` for the three.
2. **Files:** the authoritative domain/deep-dive page (confirm
   `deep-dives/domain-model.md` vs. `deep-dives/`) — document the
   collection-scoped check tier and the full-collection re-scan behavior (a
   single-item selector still reads siblings when a collection-scoped check is
   configured).
3. **File:** `docs/content/reference/glossary.md` — add "collection-scoped check."
4. **File:** `AGENTS.md` — record the `CollectionCheck` interface and the
   re-scan rule.
5. **File:** `product/specs/filesystem-checks-spec.md` — tick the Tier 3
   checklist; set Status **done**; delete spec + plan per `how-we-plan.md` once
   both shipments have landed (rationale graduates into the deep-dives page).
6. **Gate:** `make all` green.

## Key Files

| File | Role |
|---|---|
| `internal/checks/filesystem.go` | New per-item structs + `resolveTarget`; delete 4 folded structs (Phases 1–2) |
| `internal/checks/checks.go` | Add `Context.CollectionRoot` (Phase 1) |
| `internal/checks/collection.go` | `CollectionCheck`, `CollectionContext`, 3 cross-item checks (Phase 4, new) |
| `internal/checks/registry.go` | Replace/add Descriptors; add `Scope` field (Phases 1, 2, 4) |
| `internal/config/config.go` | Retire 4 constants; add new kinds + `rawCheck`/`Check` fields + validation (Phases 1, 2, 4) |
| `cmd/engine.go` | Dispatch new kinds; pass `CollectionRoot`; split item vs collection checks (Phases 1, 2, 4) |
| `cmd/check.go` | Build per-item `Context`; run collection pass with full re-scan (Phases 1, 4) |
| `cmd/rules.go`, `cmd/rules_test.go` | Registry-driven; show `Scope` in detail readout (Phase 4) |
| `internal/checks/checks_test.go`, `internal/checks/collection_test.go`, `internal/config/config_test.go`, `cmd/check_test.go` | Coverage (tests-first each phase) |
| `docs/content/reference/rules/filesystem/*` | Regenerated by `make docs-gen` (Phases 3, 5) |
| `docs/content/deep-dives/domain-model.md` (vs. `deep-dives/`), `how-to/configure-rules.md`, `reference/configuration.md`, `glossary.md`, `AGENTS.md` | Graduation targets (Phases 3, 5) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| `target` resolution home | One `resolveTarget(ctx, target) []string` helper | Every name-shape check shares it; adding a target is one switch arm, not per-check code |
| Collection root in Context | Add `Context.CollectionRoot` | `path-segments` and `path_depth` need it; minimal addition vs. a larger refactor |
| `referenced_files_exist` placement | Phase 2 (per-item), resolved relative to item dir | Needs no siblings; ships on the existing `Check` contract |
| Item vs collection split | A `kind → Scope` lookup; two engine passes | Keeps the per-item hot path unchanged; collection pass is additive |
| Collection pass + selectors | Re-scan the full collection regardless of selector | A uniqueness verdict is only correct against all items (see spec) |
| Folded kinds | Removed outright, no alias | Pre-v0; the project-layout spec set the no-back-compat precedent |
| Shipment split | A (Tiers 1–2) before B (Tier 3) | A is a contained refactor; B introduces a new interface gated on Q1 |
| Error/violation messages | Follow `cmd/AGENTS.md` grammar; violations keep the diagnostic format | Match the standard now durable on main rather than inventing phrasing |

## Out of Scope

- **`any_of` / rule composition** — its own spec (spec Q2).
- **Body markdown link integrity** — a future `markdown_relative_links_resolve`
  check, not this family (spec Q4).
- **Auto-fixing filesystem violations** — renames are link-breaking; `fix` stays
  a formatter (spec "`fix` interaction").
- **L1/L2 (heuristic/judgment) checks** — this plan is L0 only; see the research
  brief for the broader landscape.
- **Migrating configs that use the folded kinds** — no backward compatibility.
- **`inspect` suggesting filesystem checks** — the new inspector layer profiles
  a directory into a draft schema (object-shaped); teaching it to propose
  filesystem/collection checks is a separate effort.
