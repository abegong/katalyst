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
| 4 | Storage types | validate the declared `type` via the `storage` registry (`storage.Known`); drop config's duplicate allowlist |
| 5 | Collections | move `Collection`/`CollectionVariant`/`QuerySettings` + their parse to `storage/collection`; dissolve `config → query` (flip the edge to `config → collection`) |
| 6 | Collapse loader + docs | fold the `config` remnant into the `project` loader, delete the `config` package; docs/AGENTS/skill sweep |

Type relocation folds into the phase that distributes that object's parsing
(`Collection` in 5), not a separate mechanical pre-step — moving a type with its
parser touches it once instead of twice. `StorageInstance` is the exception: it
embeds `[]Collection` and `collection` imports `storage`, so it can live neither
in the `storage` root nor in `config`; as an assembly type it lands in the
`project` loader in Phase 6. The check work
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

### Phase 3 — Decouple `checks` from `config`; validate at Load (Path A)

**Goal:** one representation per check, validated at Load; `checks` no longer
imports `config`, so the loader can parse checks through the registry.

1. **File:** `internal/checks/kinds.go` (new). Move the `CheckType` type and its
   `Check*` constants here from `config`.
2. **File:** `internal/checks/*/**.go` (the ~35 check files + `jsonschema/object.go`).
   `config.CheckX` → `checks.CheckX` in each Descriptor; drop the now-unused
   `config` import.
3. **File:** `internal/checks/registry.go`. `Descriptor.CheckType` and `byKind`
   become `checks.CheckType`; change the entry point to `Parse(kind, node)
   (args, error)` + `Build(kind, args)` / `BuildCollection(kind, args)`; drop the
   legacy `Builder`/`Build`/`BuildCollection`(`CheckInstance`) and the `config`
   import. Move `CollectionScoped(kind)` here (answered by `Descriptor.Scope`).
4. **File:** `internal/checks/jsonschema/object.go`. Register the object check via
   a Descriptor-only `RegisterDescriptor` (no builder); the engine still builds it
   from the compiled schema.
5. **File:** `internal/project/config/config.go`. `config` now imports `checks`.
   Replace `CheckInstance` with the carrier `[]checks.ConfiguredCheck{Kind, Args}`;
   `buildChecks` calls `checks.Parse(kind, node)` at Load (errors fail
   `config.Load`, preserving Load-time validation) and stores the validated args.
   Delete `normalizeCheck`, the union fields, the filesystem/text validators
   (now in the family packages), and the default consts.
6. **File:** `cmd/engine.go`. Iterate `[]checks.ConfiguredCheck`, build via
   `checks.Build(kind, args)` / `BuildCollection`; the object check stays special.
7. **File:** `internal/checks/registry_test.go`. Delete `dispatchedKinds` and the
   parity test — the registry is the only enumeration.
8. **File:** `internal/project/config/config_test.go`. Stays green (Load still
   validates); adjust only error strings where `argcheck` phrasing differs.

### Phase 4 — Storage types own their config

**Goal:** the storage layer owns the enumeration of valid backend kinds;
`config` validates the declared `type` against it rather than a duplicate
allowlist.

1. **File:** `internal/storage/storage.go`. Already the single source of truth
   for backend kinds: `StorageType`, the `Filesystem` constant, the `registered`
   set, and `Known`. No change needed this phase.
2. **File:** `internal/project/config/config.go`. Delete the duplicate
   `knownStorageTypes`/`storageTypeFilesystem`; `buildInstance` defaults the type
   to `storage.Filesystem` and validates via `storage.Known`. `config` may import
   the `storage` root because it is a config-free leaf.

**Deferred to Phase 5:** moving the `StorageInstance` type. It embeds
`[]Collection`, and `collection` imports `storage` for `Granularity`/`Reference`,
so `StorageInstance` cannot live in the `storage` root (`storage → collection →
storage` cycles) nor in `config` once `Collection` moves. It is an assembly type:
it lands in the `project` loader in Phase 6, alongside the rest of the loader.

### Phase 5 — Collections own their config (DONE)

**Goal:** the collection parses its own block (including variant `when`
predicates and query settings); the central `config → query` edge dissolves.

1. **File:** `internal/storage/collection/parse.go` (new). The `Collection`,
   `CollectionVariant`, and `QuerySettings` types live here, with the collection's
   own build (`Build(BuildInput)`: raw block → `Collection`, holding raw check
   nodes), using the sibling `storage/collection/query` for `when` predicates — so
   the `when`/query dependency is intra-`collection`, not cross-tree. The raw YAML
   mirrors (`RawCollection`/`RawVariant`/`RawWhen`/`RawCheck`/`RawQuery`) move here
   too, exported so the loader can unmarshal a storage instance's collections.
   Schema validation is injected as a `SchemaKnown func(string) bool`, so the
   collection never reaches into the loader's name→path map.
2. **File:** `internal/storage/collection/collection.go`. `Item` and
   `CollectionDefinition` now name the local `Collection`, dropping the
   `collection → config` import: the edge flips to `config → collection`.
3. **File:** `internal/project/config/config.go`. `buildCollection`/`buildChecks`/
   `buildVariants`/`resolveQuery` and the raw collection mirrors are gone;
   `buildInstance` calls `collection.Build`. `config` re-exports `Collection`/
   `CollectionVariant`/`QuerySettings` as aliases so its ~14 call sites are
   untouched, and drops its direct `query` import.
4. **Tests:** `internal/project/config/config_test.go` still exercises collection
   + variant + query parsing through `config.Load` (the loader delegates), and the
   dogfood `katalyst check` over `docs/` is unchanged. (A focused
   `collection`-package parse test can follow; `Build` is covered end-to-end
   today.)

**Schema discovery stays in the loader.** The plan floated moving
`loadSchemas`/`scanKindDir` to a schema owner. Schema *resolution* is already
config-free (`jsonschema.Resolve` takes a `schemaPath` func); what remains is
directory *discovery*, which is a project-filesystem-layout concern that belongs
with the loader, not an "object owns its config" win. It folds into the `project`
loader in Phase 6 rather than moving twice.

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
| `internal/storage/storage.go` | the backend-kind registry (`Known`); Phase 5 also homes `StorageInstance` |
| `internal/storage/collection/parse.go` (new) | home for `Collection`/`Variant`/`QuerySettings` + their parse (`Build`) |
| `internal/storage/collection/collection.go` | `Item`/`CollectionDefinition` now name the local `Collection` (edge flipped) |
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
