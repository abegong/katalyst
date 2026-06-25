# Plan — listing and predicates
> Spec: [Listing and predicates](./listing-predicate-spec.md)

## Current State

`internal/storage/collection/query` owns three concepts:

- `internal/storage/collection/query/filter.go` defines the metadata predicate
  grammar: `Predicate`, `ParseFilter`, `Predicate.Matches`, and
  `TypeMismatchError`.
- `internal/storage/collection/query/query.go` defines the `item list` pipeline:
  `Record`, `Options`, `Region`, and `Apply`.
- `internal/storage/collection/query/sort.go` defines sort parsing and
  comparison for `item list`: `SortKey`, `ParseSort`, and `less`.

`internal/storage/collection/parse.go` imports `query` to parse variant
`when` discriminators and to define `Collection.Query` as `QuerySettings`.
`cmd/engine.go` evaluates variant discriminators through
`query.Predicate.Matches`. `cmd/item.go` builds `query.Options`, parses
`--filter` and `--sort`, and calls `query.Apply`.

Config uses `query:` at both project and collection scope. The block only
configures listing defaults: `filterTypeMismatch` for `item list --filter` and
`sortMissing` for `item list --sort`.

GitHub issue #76 tracks the wider terminology contradiction. `core-concepts.md`
names Query as a supported operation, `domain-model.md` says Query is out of
scope, and code uses `query` for listing filters and variant predicates. This
plan resolves that issue by naming today's shipped behavior `listing` and
`predicate`, then reserving Query for a future storage operation.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Lock behavior with tests | Add failing or pending coverage for `listing:`, old `query:` rejection, and package split expectations |
| 2 | Split predicate and listing packages | Move filter grammar to `predicate`; move listing pipeline and sort logic to `listing` |
| 3 | Rename config defaults | Rename `QuerySettings`/`RawQuery`/`Collection.Query` to `ListingDefaults`/`RawListingDefaults`/`Collection.ListingDefaults`; switch YAML to `listing:` |
| 4 | Update callers and docs | Update CLI, engine, package docs, deep dives, and reference config |
| 5 | Verify and close terminology loop | Prove old imports are gone, run the test suite, and confirm #76 acceptance is met |

The package split lands before the config rename so call sites can move from
`query` to `predicate`/`listing` independently of YAML behavior.

## Phases

### Phase 1

Goal: Capture the behavior and migration contract before moving code.

1. Add predicate package tests by moving the existing filter tests in place.

   **File:** `internal/storage/collection/predicate/predicate_test.go` (new)

   Copy the current filter grammar coverage from
   `internal/storage/collection/query/filter_test.go`, update expected package
   names to `predicate`, and keep the tests failing until the package exists.

2. Add listing package tests by moving the existing sort and pipeline tests in
   place.

   **File:** `internal/storage/collection/listing/listing_test.go` (new)

   Copy pipeline coverage from `query_test.go`.

   **File:** `internal/storage/collection/listing/sort_test.go` (new)

   Copy sort coverage from `sort_test.go`.

3. Add loader tests for the new config key.

   **File:** `internal/project/loader_test.go`

   Add coverage that project-level `listing:` applies, collection-level
   `listing:` overrides per key, and `query:` fails with an error that names
   `listing:`.

4. Add a CLI precedence test if one does not already cover both flags.

   **File:** `cmd/item_test.go`

   Cover `--on-type-mismatch` and `--sort-missing` overriding
   `ListingDefaults`, using the real Cobra root.

### Phase 2

Goal: Split `query` into predicate and listing packages without changing
behavior.

1. Move the predicate grammar.

   **File:** `internal/storage/collection/predicate/predicate.go` (new)

   Move `Predicate`, `TypeMismatchError`, `ParseFilter`, `Matches`, `match`,
   and the scalar/lookup/compare helpers from
   `internal/storage/collection/query/filter.go`.

   Rename `ParseFilter` to `Parse`. Keep the grammar and error behavior
   unchanged.

2. Move the listing pipeline.

   **File:** `internal/storage/collection/listing/listing.go` (new)

   Move `Region`, `Record`, `Options`, `Apply`, `matchAll`, and `region` from
   `query.go`. Import `internal/storage/collection/predicate` and change
   `Options.Filters` to `[]predicate.Predicate`.

3. Move sort parsing.

   **File:** `internal/storage/collection/listing/sort.go` (new)

   Move `SortKey`, `ParseSort`, `less`, `keyValue`, `compareVals`, and
   `typeRank` from `sort.go`. Use predicate helpers only if they remain
   exported; otherwise keep listing-local comparison helpers so the packages stay
   loosely coupled.

4. Remove the old query package after callers move.

   **File:** `internal/storage/collection/query/*.go`

   Delete the old package once no production or test import refers to it.

### Phase 3

Goal: Rename listing-default config internally and switch YAML from `query:` to
`listing:`.

1. Rename collection config types.

   **File:** `internal/storage/collection/parse.go`

   Rename `QuerySettings` to `ListingDefaults`, `RawQuery` to
   `RawListingDefaults`, and `resolveQuery` to `resolveListingDefaults`.

   Rename `Collection.Query` to `Collection.ListingDefaults`.

2. Parse the new YAML key.

   **File:** `internal/project/loader.go`

   Rename `rawConfig.Query` to `rawConfig.Listing` with `yaml:"listing"`.

   Pass project-level listing defaults into `loadStorage`, `buildInstance`, and
   `collection.Build`.

3. Reject the old YAML key with a targeted error.

   **File:** `internal/project/loader.go`

   Detect `query:` in `.katalyst/config.yaml` and return an error that tells the
   user to use `listing:`.

   **File:** `internal/storage/collection/parse.go`

   Detect collection-level `query:` in raw collection config and return an error
   that tells the user to use `listing:`.

4. Update project aliases.

   **File:** `internal/project/loader.go`

   Replace the `QuerySettings` alias with `ListingDefaults` if callers still
   need the project-level compatibility alias.

### Phase 4

Goal: Update callers and docs to the new names.

1. Update item list.

   **File:** `cmd/item.go`

   Import `internal/storage/collection/listing` and
   `internal/storage/collection/predicate`.

   Rename `queryFlags` only if useful; update `buildQueryOptions` to return
   `listing.Options`, call `predicate.Parse`, call `listing.ParseSort`, and read
   `col.ListingDefaults`.

2. Update variant parsing.

   **File:** `internal/storage/collection/parse.go`

   Store `[]predicate.Predicate` on `CollectionVariant` and parse variant
   `when` expressions with `predicate.Parse`.

3. Update variant routing.

   **File:** `cmd/engine.go`

   Read `c.ListingDefaults.FilterTypeMismatch` when evaluating
   `Predicate.Matches`.

4. Update package guidance.

   **File:** `internal/storage/collection/AGENTS.md`

   Replace `query` guidance with `predicate`, `listing`, and
   `ListingDefaults`.

5. Add package docs.

   **File:** `internal/storage/collection/predicate/doc.go` (new)

   Define metadata predicates and name the two consumers: listing filters and
   variant discriminators.

   **File:** `internal/storage/collection/listing/doc.go` (new)

   Define the in-memory `item list` pipeline.

6. Update user and architecture docs.

   **File:** `docs/content/reference/configuration.md`

   Rename the `query` section to `listing`, show `listing:` examples, and note
   that `query:` has been replaced.

   **File:** `docs/content/deep-dives/domain-model/collections.md`

   Replace references to the query package with the metadata predicate grammar.

   **File:** `docs/content/deep-dives/domain-model/_index.md`

   Replace the "Query" out-of-scope note with the explicit split: listing
   filters and sort keys are shipped for one collection; first-class Query is
   planned.

   **File:** `docs/content/deep-dives/domain-model/_index.md`

   Mark **Query** as planned rather than shipped. Keep listing filters out of the
   operation list unless they are named as part of Listing.

   **File:** `product/specs/domain-model-terminology-matrix.md`

   Update the Query/filter row for the new package names.

7. Update the issue-resolution trail.

   **File:** `product/specs/listing-predicate-spec.md`

   Keep the explicit #76 reference that says this work resolves the contradiction
   by separating listing/predicate from future Query.

### Phase 5

Goal: Verify the rename is complete and behavior stayed stable.

1. Check imports.

   **File:** repository-wide

   Run `rg 'internal/storage/collection/query|\\.Query\\b|QuerySettings|RawQuery|ParseFilter'`.
   Remove or justify every remaining match.

2. Format and test.

   **File:** repository-wide

   Run `gofmt -w` on touched Go files, then `go test ./...`.

3. Update generated snapshots only if output text changes intentionally.

   **File:** `cmd/testdata/snapshots/**`

   Regenerate snapshots only for user-facing text changes caused by
   `query:` to `listing:` diagnostics or docs-driven help text.

4. Verify #76 acceptance.

   **File:** docs and codebase

   Confirm `core-concepts.md`, `domain-model.md`, the config reference, and code
   package names agree: shipped behavior is listing/predicates; Query is planned.
   Close #76 after this lands.

## Key Files

| File | Role |
|---|---|
| `product/specs/listing-predicate-spec.md` | Source spec |
| `product/specs/listing-predicate-plan.md` | This implementation plan |
| `internal/storage/collection/query/filter.go` | Current predicate grammar source |
| `internal/storage/collection/query/query.go` | Current listing pipeline source |
| `internal/storage/collection/query/sort.go` | Current sort parser/comparator source |
| `internal/storage/collection/predicate/predicate.go` | New metadata predicate package |
| `internal/storage/collection/predicate/doc.go` | Predicate package docs |
| `internal/storage/collection/predicate/predicate_test.go` | Predicate grammar tests |
| `internal/storage/collection/listing/listing.go` | New item-list pipeline package |
| `internal/storage/collection/listing/sort.go` | Listing sort parser/comparator |
| `internal/storage/collection/listing/doc.go` | Listing package docs |
| `internal/storage/collection/listing/listing_test.go` | Listing pipeline tests |
| `internal/storage/collection/listing/sort_test.go` | Listing sort tests |
| `internal/storage/collection/parse.go` | Collection variants and listing-default config |
| `internal/project/loader.go` | Project-level `listing:` config parsing |
| `internal/project/loader_test.go` | Project/collection listing-default config tests |
| `cmd/item.go` | `item list` caller |
| `cmd/item_test.go` | CLI listing behavior and precedence tests |
| `cmd/engine.go` | Variant predicate evaluation |
| `internal/storage/collection/AGENTS.md` | Package conventions |
| `docs/content/reference/configuration.md` | User-facing config reference |
| `docs/content/deep-dives/domain-model/collections.md` | Variant terminology |
| `docs/content/deep-dives/domain-model/_index.md` | Query/listing vocabulary |
| `docs/content/deep-dives/domain-model/_index.md` | Query operation vocabulary |
| `product/specs/domain-model-terminology-matrix.md` | Naming matrix |
| GitHub issue #76 | Terminology contradiction this plan resolves |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Predicate home | `internal/storage/collection/predicate` | Predicates are collection-level metadata conditions used by listing and variants. They are not yet a top-level internal concept. |
| Listing home | `internal/storage/collection/listing` | Listing is the in-memory `item list` operation over collection items. It is not a storage query operation. |
| Config key | Rename `query:` to `listing:` | The config block controls listing defaults, not a general query. A targeted error for `query:` keeps the contract explicit. |
| Resolved config type | `ListingDefaults` | The struct holds default policy for listing edge cases. It is not the listing operation itself. |
| CLI flags | Keep `--filter`, `--sort`, `--on-type-mismatch`, `--sort-missing` | The CLI terms are already precise for users. The internal package names change to clarify ownership. |
| Query term | Reserve for a future storage operation | A future query should mean asking storage for matching items directly, not filtering an in-memory list. This resolves #76 by naming today's behavior listing/predicate. |

## Documentation Updates

Documentation ships in Phase 4.

- `internal/storage/collection/AGENTS.md`: update package layout and conventions.
- `internal/storage/collection/predicate/doc.go`: add predicate package docs.
- `internal/storage/collection/listing/doc.go`: add listing package docs.
- `docs/content/reference/configuration.md`: rename the `query` section to
  `listing` and document the migration error.
- `docs/content/deep-dives/domain-model/collections.md`: describe variants as using metadata
  predicates.
- `docs/content/deep-dives/domain-model/_index.md`: distinguish shipped listing filters
  from planned Query.
- `docs/content/deep-dives/domain-model/_index.md`: mark Query as planned.
- `product/specs/domain-model-terminology-matrix.md`: update the Query/filter
  row.

## Out of Scope

- Do not add a first-class storage query operation.
- Do not rename user-facing `item list --filter` to `--predicate`.
- Do not move `predicate` to a top-level `internal/predicate` package.
- Do not change filter grammar, sort behavior, grep behavior, skip/limit
  behavior, or variant routing semantics.
- Do not keep `query:` as a compatibility alias after the targeted migration
  error lands.
