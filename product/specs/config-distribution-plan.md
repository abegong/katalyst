# Plan — config distribution

> Spec: [Config distribution](./config-distribution-spec.md)
> **Status: planning.**

## Current State

- `internal/project/config/config.go` (1,198 lines) is the central typed-config
  layer: `Load` reads `.katalyst/`, and `normalizeCheck` (a ~200-line `switch`
  over every `kind`) validates each check's args into `config.CheckInstance` — a
  union of every check's fields. `CheckType` constants re-enumerate the kinds;
  `buildInstance` builds `StorageInstance`; `buildCollection`/`buildVariants`/
  `resolveQuery` build `Collection` (whose `Checks` is `[]CheckInstance`).
- `internal/checks/registry.go`: each check self-registers a `Descriptor` plus
  `Builder func(config.CheckInstance) Check` and `CollectionBuilder
  func(config.CheckInstance) CollectionCheck` (`Register(desc, build, buildColl)`,
  either builder may be nil). `checks.Build`/`BuildCollection` read the union.
- `cmd/engine.go:176,234` is the build site: it iterates a collection's
  `[]CheckInstance` and calls `checks.Build`/`BuildCollection` at run time —
  building is already lazy, the boundary the spec preserves.
- Every check family package (`structuredobject`, `markdownbodytext`,
  `filesystem`, `plaintext`, `jsonschema`) imports `config` only to read fields
  off `CheckInstance` in its builder.
- `internal/checks/registry_test.go` (`dispatchedKinds`) parses `config.go`'s
  `normalizeCheck` switch to police kind-parity — a test that exists only because
  the two enumerations must agree.
- `config_test.go` golden-asserts validation error strings (`unknown style`,
  `must be none or slugify`, `requires "prefix" or "suffix"`).

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Parse/build infra | registry parse+build registration, `argcheck` helpers, unified raw-node→check dispatch with legacy fallback, loader carries raw check nodes |
| 2 | Convert check families | `structuredobject`, `markdownbodytext`, `filesystem`, `plaintext`, `jsonschema` own their parse+build; drop each `normalizeCheck` case |
| 3 | Remove the legacy path | delete `normalizeCheck`, `CheckInstance`, `CheckType` consts, the legacy `Builder` union arg, the parity test |
| 4 | Storage types | distribute `StorageInstance` config; move the `StorageInstance` type to `storage` |
| 5 | Collections & schemas | distribute collection/variant/schema parsing; move `Collection` to `storage/collection`; dissolve `config → query` |
| 6 | Collapse loader + docs | fold the `config` remnant into the `project` loader, delete the `config` package; docs/AGENTS/skill sweep |

Type relocation folds into the phase that distributes that object's parsing
(`StorageInstance` in 4, `Collection` in 5), not a separate mechanical pre-step —
moving a type with its parser touches it once instead of twice. The check work
splits across phases 1–3 because it is the largest and the registry already
exists, so it both proves the pattern and earns the most decoupling first.

## Phases

### Phase 1 — Parse/build infrastructure

**Goal:** the registry can build a check from a raw `yaml.Node` via a check-owned
parse + build, falling back to the legacy union path for unconverted kinds, so
the tree stays green while families migrate one at a time.

1. **File:** `internal/checks/argcheck/argcheck.go` (new). Generic validation
   helpers with no per-kind knowledge: `RequireString(kind, field, v)`,
   `OneOf(kind, field, v, allowed)`, `RequireAny(kind, fields...)`, emitting the
   canonical phrasing `config_test.go` asserts. Unit-tested in
   `argcheck_test.go` (new).
2. **File:** `internal/checks/registry.go`. Add a parse-then-build registration
   alongside the legacy one: `RegisterParsed(desc, parse func(*yaml.Node)
   (any, error), build func(any) Check, buildColl func(any) CollectionCheck)`.
   Add `BuildFromConfig(kind string, node *yaml.Node) (Check, CollectionCheck,
   error)` that prefers a registered parser and falls back to the legacy
   `normalizeCheck`+`Builder` for kinds not yet converted.
3. **File:** `internal/project/config/config.go`. Have the loader retain each
   check block's raw `yaml.Node` (alongside today's decode) on the collection, so
   `BuildFromConfig` has the node for converted kinds and the legacy struct for
   the rest.
4. **File:** `cmd/engine.go`. Route check building through `BuildFromConfig`
   instead of `checks.Build`/`BuildCollection` directly. Existing engine tests
   stay green (behavior identical).

### Phase 2 — Convert the check families

**Goal:** each check owns its argument struct, parse, and build; its
`normalizeCheck` case and `CheckInstance` field reads disappear.

Per family (`structuredobject` → `markdownbodytext` → `filesystem` →
`plaintext` → `jsonschema`), repeat:

1. **File:** `internal/checks/<family>/<check>.go`. Add the check's own args
   struct, a parse func (decode + `argcheck` validation), and build func(s);
   register via `RegisterParsed`. Drop the `config.CheckInstance` read.
2. **File:** `internal/project/config/config.go`. Delete that kind's
   `normalizeCheck` case and its now-dead `CheckInstance` fields.
3. **File:** `internal/checks/<family>/<check>_test.go`. Add an arg-parse test
   (valid + each validation error); the family's existing behavior tests stay
   green.

### Phase 3 — Remove the legacy check path

**Goal:** one representation per check; the parity test is unnecessary.

1. **File:** `internal/project/config/config.go`. Delete `normalizeCheck`, the
   `CheckInstance` struct, the `CheckType` constants, and the raw-check legacy
   decode.
2. **File:** `internal/checks/registry.go`. Drop the legacy `Builder`/
   `CollectionBuilder` union-arg signatures and the fallback in
   `BuildFromConfig`.
3. **File:** `internal/checks/registry_test.go`. Delete `dispatchedKinds` and the
   parity test — the registry is now the only enumeration.

### Phase 4 — Storage types own their config

**Goal:** `StorageType` config is parsed/validated by the storage layer, not
`config`.

1. **File:** `internal/storage/storage.go`. Move the `StorageInstance` type here
   and add per-type instance parsing/validation (filesystem today), registered
   on the existing `StorageType` registry (`knownStorageTypes` becomes the
   registry's own enumeration).
2. **File:** `internal/project/config/config.go`. Delete `buildInstance` and
   `knownStorageTypes`; the loader dispatches storage blocks to the storage
   registry.
3. **File:** `internal/storage/storage_test.go`. Cover instance parsing + its
   validation errors.

### Phase 5 — Collections and schemas own their config

**Goal:** the collection parses its own block (including variant `when`
predicates and query settings) and schemas resolve themselves; the central
`config → query` edge dissolves.

1. **File:** `internal/storage/collection/collection.go`. Move the `Collection`,
   `CollectionVariant`, and `QuerySettings` types here and add the collection's
   own parse (block → `Collection`, holding raw check nodes), using the sibling
   `storage/collection/query` for `when` predicates — so the dependency is
   intra-`collection`, not cross-tree.
2. **File:** `internal/checks/jsonschema/` (or a `schema` owner). Move schema
   discovery/resolution (`loadSchemas`/`scanKindDir`) to the schema handling.
3. **File:** `internal/project/config/config.go`. Delete `buildCollection`,
   `buildVariants`, `resolveQuery`, `loadSchemas`, and the `query` import — the
   interleaving is gone.
4. **File:** `internal/storage/collection/collection_test.go`,
   `.../query`/dogfood. Verify collection + variant parsing and that `katalyst
   check` over `docs/` is unchanged.

### Phase 6 — Collapse the loader and sweep docs

**Goal:** no `config` package; `project` is the assembler; docs match.

1. **File:** `internal/project/loader.go` (new, or fold into `project.go`).
   The remaining `.katalyst/` discovery + YAML read + assembly moves here; the
   `internal/project/config` package is deleted. Update all importers.
2. **File:** `AGENTS.md` (root). Rewrite the layout tree (`config` gone; loader
   in `project`) and the dependency prose; the `config → query` caveat is
   removed.
3. **File:** `.cursor/skills/add-katalyst-check-type/SKILL.md`. Rewrite: adding a
   check is one file (args + parse + build + Descriptor + runtime), no
   `normalizeCheck` step. Its getting shorter is the proof the change worked.
4. **File:** `docs/content/deep-dives/` (`collections.md` + a config-architecture
   note). Document object-owns-its-config and the GX-fluent precedent.
5. **File:** `product/specs/domain-model-terminology-matrix.md`. Update the
   Config row; "centralized typed config" is no longer accurate.
6. Run `make all` + `make docs-gen-check` (must stay byte-identical — no
   `Descriptor` labels change).

## Key Files

| File | Role |
|---|---|
| `internal/checks/argcheck/argcheck.go` (new) | generic validation helpers (uniform error phrasing) |
| `internal/checks/registry.go` | parse+build registration; `BuildFromConfig` dispatch |
| `internal/checks/registry_test.go` | parity test deleted in Phase 3 |
| `internal/checks/<family>/*.go` | per-check args + parse + build |
| `cmd/engine.go` | build site; routes through `BuildFromConfig` |
| `internal/project/config/config.go` | source of everything being distributed; deleted by Phase 6 |
| `internal/project/config/config_test.go` | golden error strings; the parity guard through phases 1–3 |
| `internal/storage/storage.go` | new home for `StorageInstance` + instance parsing |
| `internal/storage/collection/collection.go` | new home for `Collection`/`Variant`/`QuerySettings` + parsing |
| `internal/project/loader.go` (new) | the thin loader/DataContext |
| `.cursor/skills/add-katalyst-check-type/SKILL.md` | shortened; the proof check |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Parser return shape | parse → validated args; separate `build`/`buildColl` | one decode feeds both the item and collection-scoped builders; preserves the dual registration (spec OQ2) |
| Error consistency | generic `argcheck` helpers | uniform, test-stable phrasing with no per-kind knowledge, not a recentralized switch (spec OQ1) |
| Build timing | lazy — `Collection` holds raw check config, engine builds | the existing boundary (`cmd/engine.go`); keeps `collection ⊥ checks`, avoiding the eager-build cycle |
| Migration safety | `BuildFromConfig` with legacy fallback | converted and unconverted kinds coexist, so each family lands green |
| Type homes | `Collection`→`storage/collection`, `StorageInstance`→`storage`, loader→`project` | types live with their concept; the loader assembles on top (spec, resolved) |

## Documentation updates

Carried from the spec, landing in Phase 6 except where a convention changes
earlier: the `add-katalyst-check-type` skill (Phase 6), root `AGENTS.md` layout +
prose (Phase 6), per-package `AGENTS.md` for the new owners (with each family/
storage/collection phase), the `collections.md` + config-architecture deep-dive
note (Phase 6), and the terminology matrix Config row (Phase 6). `make
docs-gen-check` stays byte-identical throughout (no `Descriptor` labels move).

## Out of Scope

- The `query` (#76) and `engine` (#77) terminology questions — separate issues.
- Any change to the on-disk `.katalyst/` format, or a JSON-Schema *for* the
  config files (a generated editor schema is a possible later add-on, not this).
- Inspector configuration — inspectors are not config-instantiated the same way;
  confirm no `config` coupling rather than refactor them.
- Spec 1's module moves (already shipped).
