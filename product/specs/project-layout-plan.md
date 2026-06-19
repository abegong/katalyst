# Project layout & init — plan

> Spec: [Project layout & init](./project-layout-spec.md)
>
> **Status: done.** All five phases implemented; `make all` green.
>
> **Deviations from the plan as written:**
> - Phase 1 used two concrete settings structs (`rawSchemaKind`,
>   `rawCollectionKind`) instead of a generic `rawKind[T]`, avoiding any
>   reliance on generics in yaml.v3 unmarshalling.
> - Phase 2 added `validator.LoadYAML(name, io.Reader)` (decode YAML → marshal
>   JSON → existing `Load`) rather than `validator.Compile(name, any)`. The
>   re-marshal route reuses the proven JSON compile path exactly, avoiding any
>   question of whether the compiler accepts YAML-native number types.
> - Phase 4 dropped the whole-config `cmd/testdata/configs/*.yaml` fixtures
>   entirely; collection bodies are small enough to inline per test, and schema
>   fixtures stayed JSON with `schemas: {format: json}` in each test project.

## Current State

- **`init` writes three files.** `cmd/init.go` holds `scaffoldConfig`,
  `scaffoldSchema`, `scaffoldExample` and writes `katalyst.yaml`,
  `schemas/book.json`, `notes/example.md`, refusing to overwrite any. Tests in
  `cmd/init_test.go` assert all three exist, that `fix --check` and `check` pass
  on the scaffold (`TestInit_scaffoldIsCanonical`, `TestInit_scaffoldChecksCleanly`).
- **Config discovery.** `internal/config/config.go`: `Filename = "katalyst.yaml"`,
  `find()` ascends for the nearest ancestor containing that file. `rawConfig` is
  `{Schemas map[string]string, Collections map[string]rawCollection}`. `Load`
  resolves schema paths, then for each collection builds checks, validates
  (`no checks configured`, unknown schema), and appends in name order.
- **Schema compilation is JSON-only.** `cmd/engine.go` `compile(path)` opens the
  file and calls `validator.Load(path, f)`; `internal/validator/validator.go`
  `Load` runs `jsonschema.UnmarshalJSON(r)`. `Validate` already normalizes
  YAML-native instance values, but the *schema document* must be JSON.
- **Callers.** `cmd/schema.go` reads `cfg.Schemas`/`SchemaPath`; its strings and
  `loadConfigFromCWD`'s error name `katalyst.yaml`. Test helpers
  (`cmd/helpers_test.go`) build a project via `writeConfigDir` (writes
  `katalyst.yaml` + `schemas/*.json`) and `setupScaffoldRepo` (runs `init`).
  Fixtures live in `cmd/testdata/{configs,schemas}` and
  `internal/validator/testdata/schemas`.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Config layout & discovery | `.katalyst/` marker; per-kind `discovery`/`format`/`defs`; convention scan + explicit map; rewrite `config_test.go` |
| 2 | Schema format | YAML/JSON schema compilation in `validator` + `engine` |
| 3 | `init` rewrite | scaffold `.katalyst/` dirs + commented `config.yaml`; drop examples |
| 4 | Callers, helpers, fixtures | migrate test helpers/fixtures; repoint `schema.go` strings; `make all` green |
| 5 | Docs & graduation | rewrite D1, domain model, `cli-spec.md` config, user docs; retire spec |

Each phase is **tests-first internally**: write the failing test sub-step, then
the implementation that makes it pass. A single up-front "scaffold all tests"
phase doesn't work in Go — tests referencing unbuilt symbols break package
compilation, blocking the rest of the suite. Phases 1–3 are the behavior change;
4 makes the existing suite green again; 5 is graduation.

## Phases

### Phase 1 — Config layout & discovery

**Goal:** Load a project from `.katalyst/`, with discovery (convention | explicit)
and format (yaml | json | both) settable per kind in `config.yaml`.

1. **File:** `internal/config/config_test.go` *(rewrite, failing first)* — replace
   the `katalyst.yaml` fixtures with the new layout scaffolded into `t.TempDir()`:
   `.katalyst/schemas/book.yaml`, `.katalyst/collections/notes.yaml`. Assert:
   project root is the ancestor of `.katalyst/`; `book` is discovered with
   `SchemaPath` pointing at the file; `notes` collection loads with `path`/`pattern`
   defaults; a collection with neither `schema` nor `checks` errors; `ErrNotFound`
   when no `.katalyst/` in any ancestor. Add cases for `discovery: explicit` (reads
   `defs`, ignores the dir), explicit with empty `defs` (error), `format: json`,
   `format: both`, per-kind independence, and a project with no `config.yaml`
   loading via defaults.
2. **File:** `internal/config/config.go` — replace `Filename` with `Dir =
   ".katalyst"`. Rewrite `find()` to ascend for an ancestor where
   `filepath.Join(dir, Dir)` is a directory; keep the `EvalSymlinks` root
   resolution. Update `ErrNotFound`'s message to name `.katalyst/`.
3. **File:** `internal/config/config.go` — restructure `rawConfig` into two
   per-kind settings blocks:
   ```go
   type rawConfig struct {
       Schemas     rawKind[string]        `yaml:"schemas"`
       Collections rawKind[rawCollection] `yaml:"collections"`
   }
   type rawKind[T any] struct {
       Discovery string       `yaml:"discovery"` // "" → convention
       Format    string       `yaml:"format"`    // "" → yaml
       Defs      map[string]T `yaml:"defs"`
   }
   ```
   Read `.katalyst/config.yaml` if present; absent → a zero `rawConfig` (all
   defaults). Validate `discovery`/`format` against the allowed sets.
4. **File:** `internal/config/config.go` — schema loading. Convention: scan
   `<root>/.katalyst/schemas/` for files matching `format` (`*.yaml`/`*.yml`,
   `*.json`, or both); `Schemas[stem] = abspath`. Explicit: require non-empty
   `Defs`; `Schemas[name] = resolve(root, path)`. A `both`-mode name collision
   (e.g. `book.yaml` and `book.json`) is a load error.
5. **File:** `internal/config/config.go` — collection loading. Extract today's
   per-collection build+validate (lines ~171–218: path/pattern defaults, object
   check from `schema`, `normalizeCheck` loop, `no checks` guard,
   `Collection.Schema` mirror) into `buildCollection(name, rawCollection)`.
   Convention: scan `<root>/.katalyst/collections/`, unmarshal each file into
   `rawCollection`, call `buildCollection(stem, …)`. Explicit: iterate `Defs`.
   Append in name order either way.
6. **Gate:** `go test ./internal/config/...` green. (The `cmd` suite is red until
   Phase 4 — expected.)

### Phase 2 — Schema format (YAML/JSON)

**Goal:** Compile schemas authored in YAML or JSON.

1. **File:** `internal/validator/validator_test.go` *(failing first)* — add a test
   that compiles a YAML-authored schema via the new entrypoint and validates the
   same instance the JSON schema accepts/rejects, asserting identical `Result`.
2. **File:** `internal/validator/validator.go` — add
   `Compile(name string, doc any) (*Schema, error)` doing `AddResource` + `Compile`
   on an already-decoded document. Refactor `Load` to `UnmarshalJSON` then delegate
   to `Compile`, so existing JSON callers and the normalize/flatten paths are
   untouched.
3. **File:** `cmd/engine.go` — in `compile(path)`, read the file bytes and decode
   by extension: `.yaml`/`.yml` → `yaml.Unmarshal` into `any`, `.json` →
   `jsonschema.UnmarshalJSON` (via the existing `validator.Load`). Pass the decoded
   doc to `validator.Compile`. Cache key stays `path`.
4. **Gate:** `go test ./internal/validator/...` green; engine schema-compile path
   exercised by Phase 1/4 fixtures.

### Phase 3 — `init` rewrite

**Goal:** `init` prepares the directory and writes no example content.

1. **File:** `cmd/init_test.go` *(rewrite, failing first)* — assert `init` creates
   `.katalyst/`, `.katalyst/schemas/`, `.katalyst/collections/` and a
   `.katalyst/config.yaml`; that no example schema/collection/document is written;
   that it refuses (exit 2, writes nothing) when `.katalyst/` already exists; that
   the scaffolded `config.yaml` is in `fix --check` canonical form; and that
   `check` on a fresh project exits 0. Delete `TestInit_scaffoldChecksCleanly`.
2. **File:** `cmd/init.go` — remove `scaffoldSchema` and `scaffoldExample`. Replace
   `scaffoldConfig` with a commented-template `config.yaml` (the default
   `schemas`/`collections` blocks shown commented out). Change the writer to create
   the three directories and the single `config.yaml`; switch the refuse-overwrite
   guard to stat `<target>/.katalyst`. Update the command `Short`.
3. **Gate:** `go test ./cmd -run TestInit` green.

### Phase 4 — Callers, helpers, fixtures

**Goal:** Bring the whole suite back to green at the new layout.

1. **File:** `cmd/helpers_test.go` — rewrite `writeConfigDir` to emit the new
   layout: `.katalyst/schemas/book.yaml` + `person.yaml` and
   `.katalyst/collections/*.yaml` for the book-and-person fixture (drop the root
   `katalyst.yaml`). Audit `setupScaffoldRepo` callers: any that relied on the old
   example `notes` collection must now scaffold their own collection explicitly.
2. **File:** `cmd/fixtures_test.go`, `cmd/testdata/{configs,schemas}` — convert the
   embedded schema fixtures to YAML (or keep `.json` and set `format: both`); split
   `testdata/configs/*.yaml` (whole-config) into per-collection files matching the
   `.katalyst/collections/` shape. Update `cmd/testdata/README.md`.
3. **File:** `cmd/schema.go` — replace `katalyst.yaml` in `Short` strings and the
   `loadConfigFromCWD` error with "the config" / "no `.katalyst/` found in this
   directory or any ancestor (run `katalyst init`)".
4. **File:** `internal/validator/testdata/` — add a YAML schema fixture mirroring
   `book.json`; note it in that `testdata/README.md`.
5. **Gate:** `make all` green.

### Phase 5 — Docs & graduation

**Goal:** Reconcile durable docs and retire the spec.

1. **File:** `product/decisions.md` — rewrite **D1**: marker is the `.katalyst/`
   directory; `config.yaml` carries per-kind `discovery`/`format` options
   (default convention + YAML); schemas/collections are file-per-definition under
   plural dirs. Note it supersedes the old root-`katalyst.yaml` decision.
2. **File:** `product/domain-model.md` (+ `domain-model-mapping.md`) — redefine
   **Project** as "the directory containing `.katalyst/`"; record the
   name-is-filename-stem convention for schemas and collections.
3. **File:** `product/specs/cli-spec.md` — replace the "Config (v0)" block
   (`collections:`/`schemas:` maps) with the `.katalyst/` layout; update the
   `init` command description (no longer "Unchanged from today").
4. **File:** `docs/configuration.md`, `README.md`, root `AGENTS.md` (the
   `internal/config` layout line) — point at `.katalyst/` and the new options.
5. **Graduation:** set the spec Status to **done** and delete spec + plan per
   `how-we-plan.md` (rationale now lives in D1 and the domain model).
6. **Gate:** `make all` green; repo grep finds no stale `katalyst.yaml` references
   outside the rewritten D1's historical note.

## Key Files

| File | Role |
|---|---|
| `internal/config/config.go` | `.katalyst/` discovery; per-kind settings; convention/explicit loading (edited) |
| `internal/config/config_test.go` | New-layout + options coverage (rewritten) |
| `internal/validator/validator.go` | `Compile(name, doc any)` entrypoint (edited) |
| `cmd/engine.go` | Decode schema by extension before compiling (edited) |
| `cmd/init.go` | Scaffold `.katalyst/` dirs + commented `config.yaml` (rewritten) |
| `cmd/init_test.go` | New init behavior (rewritten) |
| `cmd/schema.go` | Repointed strings/messages (edited) |
| `cmd/helpers_test.go`, `cmd/fixtures_test.go`, `cmd/testdata/*` | Fixtures at new layout (edited) |
| `internal/validator/testdata/*` | YAML schema fixture (added) |
| `product/decisions.md`, `product/domain-model*.md`, `product/specs/cli-spec.md`, `docs/configuration.md`, `README.md`, `AGENTS.md` | Graduation targets (edited) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Discovery/format home | Branch inside `config.Load`, no new package | One loader stays the single entry; the branch is small and table-driven |
| Explicit map location | Under each kind's block as `defs` | Avoids colliding the settings block name (`schemas:`) with a bare name→path map |
| YAML schema support | Decode in `engine`, add `validator.Compile(name, any)` | The validator already tolerates YAML-native *instances*; this extends the same tolerance to the schema document without a second compiler path |
| `format: both` collisions | `book.yaml` + `book.json` → load error | Two files claiming one name is ambiguous; fail loudly rather than pick |
| Tests-first granularity | Per-phase, not one scaffolding phase | Go won't compile a package whose tests reference unbuilt symbols |
| `config.yaml` optional | Absent → all defaults | Keeps the directory the marker; a no-settings project needs no file |

Mirror D1 in `product/decisions.md` during Phase 5 (the rewrite *is* that mirror).

## Out of Scope

- **New check kinds or selector-grammar changes** — `cli-spec.md` owns those.
- **Migrating an existing `katalyst.yaml`** — no backward compatibility (pre-v0).
- **`schema show` rendering YAML** — it pretty-prints JSON and already falls back
  to raw bytes for non-JSON, so a YAML schema prints verbatim; a YAML-aware
  pretty-printer is deferred.
- **`config.yaml` settings beyond `discovery`/`format`/`defs`** — no other
  project-level settings in v0.
- **Per-kind format for collection *checks*** — collection files follow the same
  `format` scan, but the check vocabulary itself is unchanged.
