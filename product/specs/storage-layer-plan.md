# Storage layer — plan

> Spec: [Storage layer](./storage-layer-spec.md)
>
> **Status: implementing.** Phase 1 (the seam) landed: `internal/storage`
> exists and `internal/project` delegates to it, with the existing suite green
> unedited. Phases 2–4 pending. Splits the hardcoded filesystem mapping into
> `internal/storage` (StorageType, StorageInstance, CollectionDefinition), then
> moves the config model so a StorageInstance declares its own collections.
> Sequenced **seam first** (a behavior-preserving refactor the existing suite
> guards unchanged), **config change second** (the breaking part), so risk is
> isolated per phase.

## Current State

- **The mapping is hardcoded in `internal/project`.** `project.go` holds the
  forward map `Items(c)` (glob `c.Dir/c.Pattern`, id = filename stem), the
  reverse map `ItemPath(c, id)` (`Join(c.Dir, id+c.Ext())`), `Unmatched(c)`, and
  `ItemAt`. `Item` is `{Collection config.Collection; ID, Path string}`. All four
  assume `os`/`filepath` + a flat directory; there is no interface a second
  backend could implement.
- **Collections are a standalone config kind.** `internal/config/config.go`
  parses `.katalyst/collections/<name>.yaml` (convention) or `collections.defs`
  in `config.yaml` (explicit) into `Config.Collections []Collection`
  (`{Name, Path, Dir, Pattern, Schema, Checks, Query}`), validated for unique
  names and known schema refs. There is **no** notion of *where* a directory
  lives beyond "relative to `Config.Root`."
- **Selectors orchestrate on top.** `selector.go` `Resolve` expands selectors
  into `Item`s and collections-to-scan; `cmd/check.go`, `cmd/engine.go`,
  `cmd/item.go`, `cmd/collection.go`, `cmd/fix.go`, and `cmd/write_validation.go`
  consume `Project` (`p.Items`, `p.ItemAt`, `p.Resolve`, `ItemPath`,
  `p.Collections`). They never touch `os` directly — the seam they go through is
  `Project`, which is what makes the refactor tractable.
- **`init` scaffolds the old shape.** `cmd/init.go` creates `schemas/` and
  `collections/` directories and a commented `config.yaml`; `scaffoldConfig`
  documents the `schemas`/`collections` blocks.
- **Vocabulary.** `docs/content/deep-dives/connectors.md` and the glossary's
  `Connector` row frame all of this as a single "connector," the term the spec
  retires.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | The seam | `internal/storage`: `StorageType`, `CollectionDefinition`, `Granularity`, `Reference`; `Item` moves here; `FilesystemCollectionDefinition` lifts today's `project` bodies; `project` delegates. **No config or fixture change.** |
| 2 | Config model | `internal/config` parses StorageInstances with embedded `collections:` (convention `storage/` + inline `config.yaml`); flatten + global-name validation; `storage` builds one definition per instance; `init` writes `storage/local.yaml`; migrate fixtures. **Breaking.** |
| 3 | Per-collection files | Instance-scoped `storage/<instance>/<collection>.yaml` escape hatch; inline + per-file coexist. Separable into a follow-up PR (seam unchanged). |
| 4 | Docs & graduation | `connectors.md` → `storage.md` reframe, glossary, `configuration.md`, `domain-model.md`, `init`/`cli-spec.md`, `AGENTS.md`, package docs; retire the spec/plan. |

Each phase is **tests-first internally**: write the failing test sub-step, then
the code that makes it pass. A single up-front scaffold doesn't compile in Go,
where a test naming an unbuilt symbol breaks the whole package.

## Phases

### Phase 1 — The seam (`internal/storage`), behavior-preserving

**Goal:** Every path↔item-identity translation passes through one interface, the
filesystem implements it, and the **existing test suite stays green unchanged**.

1. **File:** `internal/storage/storage_test.go` *(new, failing first)* — over a
   `t.TempDir()` corpus, assert a `FilesystemCollectionDefinition` built for a
   `config.Collection`: `Items` returns ids = stems sorted; `Unmatched` lists
   non-`pattern` files; `Reference(c, "dune")` returns `<dir>/dune.md`;
   `Granularity()` is `FileIsItem`. These mirror the assertions in today's
   `internal/project/project_test.go`.
2. **File:** `internal/storage/storage.go` *(new)* — define `StorageType` (a
   string), a registry (`Register`/`Lookup`) with `filesystem` the only entry,
   `Granularity` (`FileIsItem`, `UnitIsCollection`), `Reference string`, and the
   `CollectionDefinition` interface (`Granularity()`, `Collections()`,
   `Items(config.Collection)`, `Unmatched(config.Collection)`,
   `Reference(config.Collection, string)`). `storage` imports `config`, never the
   reverse.
3. **File:** `internal/storage/item.go` *(new)* — **move** `Item`
   (`{Collection config.Collection; ID, Path string}`) here from
   `internal/project/project.go`, resolving the would-be `project`↔`storage`
   cycle (spec: packaging note).
4. **File:** `internal/storage/filesystem.go` *(new)* —
   `FilesystemCollectionDefinition{root string; collections []config.Collection}`
   implementing the interface by **lifting `Items`/`Unmatched`/`ItemPath`
   verbatim** from `project.go` (same `doublestar`, same stat-and-no-op on a
   missing dir). `Reference` is today's `ItemPath`.
5. **File:** `internal/project/project.go` *(edit)* — delete the moved bodies;
   `Item` becomes `type Item = storage.Item` (alias, so `cmd` is untouched).
   `Items`/`Unmatched`/`ItemAt` become thin wrappers that build a
   `FilesystemCollectionDefinition` from `p.cfg` (Root + Collections) and
   delegate. `ItemPath` delegates to `Reference`. `selector.go` `Resolve` is
   unchanged in shape.
6. **Gate:** `go test ./internal/... ./cmd/...` green **with no test or fixture
   edits** — the proof that Phase 1 changed no behavior.

### Phase 2 — Config model: instances declare their collections

**Goal:** A StorageInstance carries connection detail plus an embedded
`collections:` block; the standalone `collections/` kind is gone; `init` writes a
default instance explicitly.

1. **File:** `internal/config/config_test.go` *(edit, failing first)* — assert
   the new loader: a `storage/local.yaml` (`type: filesystem`, `root: .`,
   embedded `collections:`) yields `Config.Storage` with one instance and the
   flattened `Config.Collections`; an inline `config.yaml`
   `storage: {discovery: explicit, defs: {...}}` yields the same; a collection
   name reused across two instances is a load error; an unknown `type` is a load
   error; a collection's `schema` still must reference a known schema; **no**
   `storage/` present is a load error (no implicit synthesis). Update existing
   collection-loading cases to the new shape.
2. **File:** `internal/config/config.go` *(edit)* — add
   `StorageInstance{Name, Type, Root string; Collections []Collection}` and
   `Config.Storage []StorageInstance`. Add `rawStorageKind`/`rawStorageInstance`
   (`type`, `root`, `collections map[string]rawCollection`) and a `loadStorage`
   that reuses `scanKindDir`/`normDiscovery`/`formatExts` for the `storage/`
   directory and the inline `defs`. `buildCollection` gains the instance `root`
   (so `Dir = root + path`) and records `Collection.Storage` (instance name).
   `Config.Collections` is the **flattened** view across instances, validated for
   **project-wide unique names** (selectors carry no instance qualifier). Remove
   `loadCollections`/`collectionsSubdir`. `type` is kept as a plain string here;
   the registry of implementations stays in `internal/storage`, so `config` does
   not import `storage`.
3. **File:** `internal/storage/filesystem.go` *(edit)* — add
   `BuildInstance(config.StorageInstance) (CollectionDefinition, error)` that
   validates `Type` against the registry and constructs the definition from the
   instance `root` + its collections. `internal/project` builds definitions
   per-instance (selecting by `Collection.Storage`) instead of one global one.
4. **File:** `cmd/init.go` *(edit)* — stop creating `collections/`; create
   `storage/` and write `storage/local.yaml` (`type: filesystem`, `root: .`,
   empty `collections:`). Rewrite `scaffoldConfig`'s comments for the `storage`
   block and the two authoring forms. Update `cmd/init_test.go`.
5. **File:** fixtures across `cmd/*_test.go` and `internal/*/testdata`
   *(edit)* — migrate every `.katalyst/collections/<name>.yaml` scaffold to an
   instance with embedded collections (most live in `cmd/fixtures_test.go` /
   `cmd/helpers_test.go`; sweep `check_test`, `item_test`, `collection_test`,
   `fix_test`, `inspect_test`). This is the bulk of the breaking change.
6. **Gate:** `make all` green.

### Phase 3 — Per-collection files for large instances

**Goal:** An instance may externalize a collection into its own file, restoring
one-reviewable-file-per-change for large instances; inline remains the default.

1. **File:** `internal/config/config_test.go` *(edit, failing first)* — assert
   that for instance `local`, a `storage/local/books.yaml` is loaded as the
   `books` collection of `local`; inline and per-file collections **coexist** in
   one instance; a collection name colliding between an instance's inline block
   and its directory is an error.
2. **File:** `internal/config/config.go` *(edit)* — after building an instance's
   inline collections, scan an instance-scoped directory
   `.katalyst/storage/<instance>/` (same `scanKindDir`/format machinery) and fold
   those collections in, erroring on a name collision. The chosen layout mirrors
   today's `collections/` directory (spec settles this over an in-block `$file:`
   key).
3. **Gate:** `make all` green.

*Phase 3 is separable:* the seam and `Project` are unchanged by it, so it can
ship in a follow-up PR if 1–2 land first.

### Phase 4 — Docs & graduation

**Goal:** Retire the term *connector*, document the new model, graduate the spec.

1. **File:** `docs/content/deep-dives/connectors.md` → **rename to `storage.md`**
   (`git mv`); retitle "Storage layer"; reframe the body around StorageType /
   StorageInstance / CollectionDefinition, keeping the GX lineage, the
   granularity principle, the configured/inferred axis, unmatched-is-first-class,
   and the "do better than GX" lessons. Fix the `relref`s in
   `deep-dives/_index.md`, `_index.md`, `domain-model.md` (lines 169–170),
   `contributing/how-we-document.md`, and `how-we-plan.md`.
2. **File:** `docs/content/reference/glossary.md` *(edit)* — remove the
   `Connector` row; add **StorageType**, **StorageInstance**,
   **CollectionDefinition**, **Granularity**.
3. **File:** `docs/content/reference/configuration.md` *(edit)* — document the
   `storage/` kind with embedded `collections:`, `type`/`root`, the convention
   vs. inline forms, the instance-scoped per-collection directory, that `init`
   writes a default `local` instance, and that the standalone `collections/`
   kind is **replaced** (breaking change).
4. **File:** `docs/content/getting-started.md` *(edit)* — move the walkthrough
   from `.katalyst/collections/` to declaring collections inside
   `storage/local.yaml`.
5. **Files:** `internal/storage/doc.go` *(new)* — package doc: the three
   concepts, the two-way contract, granularity, GX provenance + corrections.
   `internal/config/README.md`, `internal/project` package doc *(edit)* — the
   storage kind and the consume-the-seam framing. `AGENTS.md` *(edit)* — record
   "path ⇄ item-identity passes through `internal/storage.CollectionDefinition`;
   don't inline filesystem assumptions elsewhere."
6. **File:** `product/specs/cli-spec.md`, `product/specs/dogfood-docs-spec.md`,
   `product/v0-implementation-plan.md` *(edit)* — update `connectors.md`/
   `collections/` references; fix `cli-spec.md`'s `init` description (default
   storage instance, not empty `collections/`).
7. **Graduation:** set the spec Status to **done**, run the `how-we-plan.md`
   graduation checklist, delete `storage-layer-spec.md` + this plan. (`storage.md`
   is evergreen and stays.)
8. **Gate:** `make all` and `make docs-gen` clean; no stale `connector` /
   `collections/` references (`grep`).

## Key Files

| File | Role |
|---|---|
| `internal/storage/storage.go` | `StorageType` registry, `CollectionDefinition`, `Granularity`, `Reference` (new) |
| `internal/storage/item.go` | `Item`, moved from `project` (new) |
| `internal/storage/filesystem.go` | `FilesystemCollectionDefinition` + `BuildInstance` (new) |
| `internal/storage/*_test.go`, `doc.go` | Seam tests + package doc (new) |
| `internal/config/config.go` | `StorageInstance`, embedded collections, flatten + unique-name validation (edit) |
| `internal/project/project.go` | `Item` alias; `Items`/`Unmatched`/`ItemAt`/`ItemPath` delegate to the seam (edit) |
| `cmd/init.go` | Write `storage/local.yaml`; rewrite scaffold (edit) |
| `cmd/*_test.go`, `internal/*/testdata` | Fixture migration to the instance shape (edit) |
| `docs/.../connectors.md`→`storage.md`, `glossary.md`, `configuration.md`, `domain-model.md`, `getting-started.md` | Vocabulary + config docs (edit/rename) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Seam home | New `internal/storage`, consumed by `project` | The mapping is backend-specific; `project` keeps selectors/orchestration, `storage` owns the backend↔domain map (issue #31's "narrow interface") |
| `Item` location | Move to `internal/storage` | The interface returns items; keeping `Item` in `project` would cycle (`project`↔`storage`). A `type Item = storage.Item` alias keeps `cmd` untouched |
| Type registry vs. config | `config` holds `type` as a string; `internal/storage` owns the registry and validates at build | Prevents a `config`↔`storage` import cycle; `config` stays pure data |
| Collection name scope | Globally unique across instances, validated at load | Selectors are `<collection>/<item>` with no instance qualifier; ambiguity must fail loudly |
| Sequencing | Seam first (no fixture change), config model second | Isolates the behavior-preserving refactor so the unchanged suite proves it; the breaking config change lands on a stable seam |
| No implicit instance | `init` writes `storage/local.yaml`; a missing instance is an error | Spec decision: explicit on disk and reviewable, never a silent runtime default |
| Per-collection files | Instance-scoped `storage/<instance>/` directory | Mirrors today's `collections/` convention; inline stays the default, per-file is the escape hatch for large instances |

## Out of Scope

- **Multi-coordinate templates / the GX two-way port.** Item id stays the stem
  (one coordinate); `Reference` stays `Join(dir, id+ext)`. The bidirectional
  template (`{slug}_{year}.md`) and the 82 GX permutation tests land when a
  richer layout or SQLite arrives (spec).
- **Inferred mode.** Discovering collection names from the store (vs. declaring
  them) belongs to the future `infer`/`profile` path, not `check`.
- **A `doctor`/`explain` command** (GX `self_check`).
- **Any non-filesystem StorageType** and **multiple definitions per instance**.
- **A file mapping into more than one collection.** Invariant #4 (one file, one
  collection) is retained; `Resolve` keeps de-duplicating by path.

## Test checklist

Phase 1 (seam, behavior-preserving):
- [ ] `FilesystemCollectionDefinition.Items` → stems, sorted; missing dir → none
- [ ] `Unmatched` lists non-`pattern` files
- [ ] `Reference(c, id)` → `<dir>/<id><ext>` (reverse map)
- [ ] `Granularity()` → `FileIsItem`
- [ ] existing `internal/project` + `cmd` suites pass **unedited**

Phase 2 (config model):
- [ ] `storage/local.yaml` with embedded `collections:` loads; `Config.Storage`
      + flattened `Config.Collections` correct, `Dir = root + path`
- [ ] inline `config.yaml` `storage` (explicit) loads identically
- [ ] duplicate collection name across instances → load error
- [ ] unknown `type` → load error; unknown schema ref still errors
- [ ] no `storage/` and no inline `storage` → load error (no synthesis)
- [ ] `init` writes `storage/local.yaml`, no `collections/`

Phase 3 (per-collection files):
- [ ] `storage/<instance>/<name>.yaml` loads as that instance's collection
- [ ] inline + per-file collections coexist; name collision → error

Phase 4 (docs):
- [ ] `make docs-gen` clean; no `connector` / `collections/` references remain
