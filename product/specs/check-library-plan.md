# CheckLibrary — plan

> Spec: [CheckLibrary](./check-library-spec.md)
>
> **Status: planning.** Implements the spec's **first PR**: introduce
> `CheckLibrary` / `SchemaLibrary` / `Schema` in `internal/checks`, migrate the
> JSON Schema validator onto it as the first schema-backed library, and prove
> the out-of-process seam with a fake library. Native-family migration and the
> Vale library are staged follow-ups (see Sequencing). No user-visible behavior
> changes: `.katalyst/schemas/` stays flat and every existing `cmd` test passes
> unchanged.

## Current State

- **The validator is a standalone wrapper.** `internal/validator/validator.go`
  exposes `Schema` (wrapping `*jsonschema.Schema`), `Load(name, r)` /
  `LoadYAML(name, r)`, `Validate(instance) Result`, and `flatten`/`visit`/
  `normalize`. It imports `santhosh-tekuri/jsonschema/v6` and `yaml.v3` and knows
  nothing about `internal/checks`.
- **The `object` check binds the validator to an item.**
  `internal/checks/structuredobject/object.go` defines `Object{Schema
  *validator.Schema}`; `Run` calls `Schema.Validate(ctx.Meta)` and maps each
  `validator.Error` to a `checks.Violation` via `checks.LookupLine`. Its `init()`
  registers the `object` Descriptor under `Family: "structuredObject"` with **no
  builder** (the engine builds it specially).
- **The engine owns compilation, caching, and precedence.** `cmd/engine.go`:
  `engine.cache map[string]*validator.Schema` (`engine.go:25`); `compile(path)`
  opens the file and switches on `.yaml`/`.json` to call `validator.LoadYAML` /
  `validator.Load` (`engine.go:46`); `checksFor` encodes the object-schema
  precedence inline — `--schema` forced path, then inline `schema:` key, then the
  collection's `object` checks (`engine.go:101`) — each branch appending a
  `structuredobject.Object{Schema}`.
- **The check registry is the single source of truth for check types.**
  `internal/checks/registry.go`: `Descriptor` (with json wire-contract tags),
  `Register(desc, build, buildColl)` (panics on duplicate kind, `byKind` index),
  `Descriptors()`, `Build`/`BuildCollection`, `Families()`. `registry_test.go`
  enforces kind↔descriptor parity against `config.normalizeCheck`.
- **Wiring is by blank-import.** `internal/checks/all/all.go` blank-imports the
  four family packages so their `init()`s populate the registry; the engine,
  `gendocs`, the `check-types` command, and `registry_test` import it.
- **Test helpers reach the validator directly.**
  `internal/checks/checktest/checktest.go` `MustLoadSchema` calls
  `validator.Load`; `internal/checks/structuredobject/structuredobject_test.go`
  builds `Object{Schema}` from it; `internal/validator/validator_test.go` and
  `internal/validator/testdata/` hold the schema fixtures.
- **Schemas are discovered flat.** `internal/config` `loadSchemas` populates
  `Config.Schemas map[string]string` (name → absolute path) from
  `.katalyst/schemas/`. This is **unchanged** by this work; the owning library is
  resolved at the binding site from the check's `kind`.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Core abstraction | `CheckLibrary`/`SchemaLibrary`/`Schema`, library registry, `Descriptor.Library`, kind→library lookup, availability gate — all additive |
| 2 | JSON Schema library | Move `internal/validator` → `internal/checks/jsonschema`; implement `SchemaLibrary`; relocate the `object` check + error mapping; register; retire the old package |
| 3 | Engine cutover | Cache keyed `(library, path)`; compile + availability via the library; object-schema precedence resolved by the library |
| 4 | Out-of-process seam | A fake `SchemaLibrary` proves availability is a hard error and `Violation.File` maps findings back |
| 5 | Docs (this PR's share) | Package docs, `AGENTS.md`, glossary/deep-dive vocabulary; regenerate the check-types reference |

Phases 1 is additive (new code beside the old). Phase 2 is the atomic move of
the validator; Phase 3 flips the engine onto it; together they leave no
half-migrated state. Each phase is **tests-first internally**: write the failing
test sub-step, then the code that passes it. After every phase, `make all` is
green.

**Staged follow-ups (separate PRs, not this plan):**

- **Native families onto `CheckLibrary`.** Each of `filesystem`, `plaintext`,
  `markdownbodytext`, `structuredobject` registers as a (non-schema) library and
  sets `Descriptor.Library`; the kind→library map then covers every check type.
- **The Vale library.** `internal/checks/vale`, the `prose` check type, the real
  `vale --version` / `vale --output JSON` integration.
- **Collection-scoped batching for out-of-process libraries.** A GitHub issue
  opened on this PR's completion (per spec).

## Phases

### Phase 1 — Core abstraction

**Goal:** The `CheckLibrary` vocabulary and registry in `internal/checks`, with
nothing consuming it yet. Core stays free of the jsonschema dependency.

1. **File:** `internal/checks/library_test.go` *(new, failing first)* —
   `package checks_test`. Register a fake `SchemaLibrary` via
   `checks.RegisterLibrary`; assert `Libraries()` returns it, a duplicate name
   panics, and `LibraryFor(kind)` resolves a check type to its owning library via
   the Descriptor's `Library` field (and returns `(nil, false)` for a kind with
   an empty `Library`). Assert a `Schema` returned by `CompileSchema` runs
   through `Schema.Check(ctx)`.
2. **File:** `internal/checks/library.go` *(new)* — define the three interfaces
   (`CheckLibrary{Name, Available}`, `SchemaLibrary{CheckLibrary, CompileSchema(name
   string, src []byte) (Schema, error)}`, `Schema{Check(Context) []Violation}`),
   plus `RegisterLibrary(CheckLibrary)` / `Libraries()` (sorted, duplicate-name
   panic mirroring `Register`) and `LibraryFor(config.CheckType) (CheckLibrary,
   bool)`. `Schema` is abstract, so core still imports no engine library.
3. **File:** `internal/checks/registry.go` — add `Library string` (json:
   `"library,omitempty"`) to `Descriptor`; back `LibraryFor` with `byKind` →
   `desc.Library` → a `byLibrary` name index. Document that the field is the
   owning library's `Name()`, empty until a check type is migrated.
4. **File:** `internal/checks/registry_test.go` — extend parity: every non-empty
   `Descriptor.Library` must name a registered library. Empty stays legal during
   the staged migration.
5. **Gate:** `go test ./internal/checks/...` green; no other package changed.

### Phase 2 — JSON Schema library

**Goal:** `internal/validator` becomes `internal/checks/jsonschema`, the first
`SchemaLibrary`, owning the `object` check and the error mapping. The old
package and the scattered `object` binding are gone.

1. **File:** `internal/checks/jsonschema/jsonschema_test.go` *(new, failing
   first)* — `package jsonschema_test`. Port `internal/validator/validator_test.go`:
   assert the library `Name() == "json-schema"`, `Available() == nil`, and
   `CompileSchema` accepts both the JSON and YAML book fixtures (one entry point,
   no extension switch). Assert the compiled `Schema.Check` over a
   `checks.Context` reproduces today's `object` violations **byte-for-byte**
   (path + message + line), reusing the moved `testdata`.
2. **File:** `internal/checks/jsonschema/jsonschema.go` *(new)* — move
   `validator.go`'s `Schema`/`Load`/`LoadYAML`/`flatten`/`visit`/`normalize` here
   as the unexported engine. Add `Library` implementing `SchemaLibrary`:
   `CompileSchema(name, src)` normalizes through the YAML path (`LoadYAML` accepts
   JSON too, since JSON is valid YAML), eliminating the engine's `.yaml`/`.json`
   switch. The compiled type implements `checks.Schema.Check` by calling
   `Validate(ctx.Meta)` and mapping `Error` → `Violation` with
   `checks.LookupLine(ctx.Doc.Lines, …)` — the logic lifted from
   `structuredobject/object.go`.
3. **File:** `internal/checks/jsonschema/object.go` *(new)* — the `object` check
   type: a per-item check holding a `checks.Schema`, whose `Run` delegates to
   `schema.Check(ctx)`; an exported constructor the engine uses; and the `init()`
   that calls `checks.RegisterLibrary(Library{})` and `checks.Register` for the
   `object` Descriptor (moved verbatim from `structuredobject/object.go`, now with
   `Family: "structuredObject"` unchanged **plus** `Library: "json-schema"`, still
   nil builders).
4. **File:** move `internal/validator/testdata/` → `internal/checks/jsonschema/testdata/`;
   port `testdata/README.md`. Update `internal/checks/checktest/checktest.go`
   `MustLoadSchema` to return a `checks.Schema` via the jsonschema library's
   compile entry point.
5. **File:** delete `internal/checks/structuredobject/object.go`; move its
   `object` test case out of `structuredobject_test.go` into the jsonschema suite.
   `structuredobject` keeps only its field check types (`object_required_field`,
   …). Delete `internal/validator/` entirely.
6. **File:** `internal/checks/all/all.go` — add
   `_ "github.com/abegong/katalyst/internal/checks/jsonschema"`.
7. **Gate:** `go test ./internal/checks/...` green; `grep -r internal/validator`
   finds nothing.

### Phase 3 — Engine cutover

**Goal:** `cmd/engine.go` compiles and gates through the library; the
object-schema precedence policy lives in the jsonschema library.

1. **File:** `internal/checks/jsonschema/resolve_test.go` *(new, failing first)*
   — assert the library's schema-source resolution honors the spec precedence:
   forced `--schema` path wins, then an inline `schema:` name (looked up in
   `config.Schemas`), then the collection's `object` check schemas; an unknown
   inline/collection name returns a clear error naming the schema.
2. **File:** `internal/checks/jsonschema/resolve.go` *(new)* — a `Resolve` helper
   (given forced path, inline name, the collection's effective checks, and
   `*config.Config`) returning the ordered `[]` of (name, absolute path) to
   compile. This moves `checksFor`'s object branch out of the engine; the package
   already imports `config`.
3. **File:** `cmd/engine_test.go` *(extend)* — assert the cache compiles a schema
   once per `(library, path)` (the "compile once per absolute path" invariant in
   `internal/checks/README.md`), and that a configured library whose `Available()`
   errors fails the run with a non-zero exit before any item is checked.
4. **File:** `cmd/engine.go` — retype `cache` to `map[libPathKey]checks.Schema`;
   rewrite `compile` to resolve the owning library via `checks.LibraryFor("object")`,
   read the file bytes, and call `SchemaLibrary.CompileSchema`. Replace the three
   inline precedence branches in `checksFor` with a call to `jsonschema.Resolve`
   over the effective checks, then build the jsonschema `object` check per
   resolved schema. Call `lib.Available()` before compiling and surface a non-nil
   error as a run failure. Drop the `structuredobject` and `validator` imports;
   add `jsonschema`.
5. **Gate:** `make all` green; existing `cmd` tests (including `--schema` and
   inline-`schema:` coverage) pass unchanged.

### Phase 4 — Out-of-process seam

**Goal:** Prove the abstraction carries an out-of-process library without
shipping Vale: availability is a hard error, and findings map back by file.

1. **File:** `internal/checks/library_oop_test.go` *(new, failing first)* — a
   fake `SchemaLibrary` modeling an out-of-process tool: `Available()` returns an
   error when a test flag says the "binary" is absent, and its `Schema.Check`
   returns `Violation`s with `File` set (as a batched external run would). Assert
   the engine/registry path: (a) a configured library whose `Available()` errors
   aborts with a non-zero exit and the library name in the message; (b) when
   available, violations preserve `Violation.File` so a finding is attributed to
   the right item. No `vale` binary involved.
2. **Gate:** `go test ./...` green.

### Phase 5 — Docs (this PR's share)

**Goal:** Lock the vocabulary and package rationale that ships with the
abstraction; defer the rest to the follow-up PRs that add libraries.

1. **File:** `internal/checks/library.go` doc comments (or a `doc.go` if long) —
   the `CheckLibrary`/`SchemaLibrary`/`Schema` contract, the one-registry model,
   availability as a hard error, and the per-item-first invocation note. Absorbs
   the spec's rejected alternatives.
2. **File:** `internal/checks/jsonschema/jsonschema.go` package doc — reframe the
   moved wrapper as "the JSON Schema library"; note `object` is the check type it
   provides and that `--schema` / inline `schema:` are its sugar.
3. **File:** `AGENTS.md` (root) — a `CheckLibrary` section: the one registry, the
   `internal/checks/all` aggregator now wiring libraries too, *library is
   provenance, family is source-data kind* (orthogonal), and that a schema-backed
   library implements `SchemaLibrary` while natives will be plain libraries.
4. **File:** `docs/content/reference/glossary.md` — add **CheckLibrary**; revise
   **Schema** (the Katalyst concept, expressed as JSON Schema today), **Schema
   directive**, **Validator**, and **Resolver** toward the general concept.
5. **File:** `docs/content/deep-dives/core-concepts.md` and
   `domain-model.md` — name the `CheckLibrary` concept under **Check**, and
   generalize the **Schema**/**Resolver** prose from "compiled JSON Schema" to "a
   library's compiled schema," JSON Schema as the worked instance.
6. **File:** regenerate `docs/reference/check-types/` with `make docs-gen` —
   `Descriptor` gained a `library` json field, so `check-types list --json`
   output is additively different; confirm `docs-gen-check` is clean.
7. **Gate:** `make all` and `make docs-gen` clean.

## Key Files

| File | Role |
|---|---|
| `internal/checks/library.go` | `CheckLibrary`/`SchemaLibrary`/`Schema`, `RegisterLibrary`/`Libraries`/`LibraryFor` (new) |
| `internal/checks/library_test.go`, `library_oop_test.go` | registry + availability + file-mapping, via fakes (new) |
| `internal/checks/registry.go` | `Descriptor.Library`, `byLibrary` index (edited) |
| `internal/checks/jsonschema/jsonschema.go` | moved validator + `SchemaLibrary` impl + `Schema.Check` mapping (new, from `internal/validator`) |
| `internal/checks/jsonschema/object.go` | `object` check type, library + descriptor registration (new) |
| `internal/checks/jsonschema/resolve.go` | object-schema precedence resolver (new) |
| `internal/checks/jsonschema/testdata/` | schema fixtures (moved) |
| `internal/checks/structuredobject/object.go` | the old `object` binding (deleted) |
| `internal/validator/` | the standalone wrapper (deleted) |
| `internal/checks/all/all.go` | blank-import jsonschema (edited) |
| `internal/checks/checktest/checktest.go` | `MustLoadSchema` → jsonschema entry point (edited) |
| `cmd/engine.go` | cache `(library, path)`, compile + availability via library, precedence via `jsonschema.Resolve` (edited) |
| `AGENTS.md`, `docs/.../glossary.md`, `core-concepts.md`, `domain-model.md` | vocabulary + rationale (edited) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| One abstraction | `CheckLibrary` for every check-type provider; `SchemaLibrary` capability for schema-backed ones | Spec: every check type has an owning library; native libraries need no schema/availability machinery, so a capability interface keeps them clean |
| One registry | Library registration beside check-type registration in `internal/checks`; `Descriptor.Library` links them | Avoids a second registry that splits "which check types exist" from "who owns them"; reuses the proven `Register`/`byKind` machinery |
| Schemas stay flat | `.katalyst/schemas/` and `config.Schemas` unchanged; library resolved from the binding's `kind` | Spec Q1: no migration; the `kind` already names the engine, so a flat namespace is unambiguous |
| Compilation in the library | `CompileSchema(name, src)` normalizes JSON and YAML through one path | Removes the engine's extension switch; each library parses its own bytes |
| Availability is a hard error | Engine calls `Available()` before running a library's checks; non-nil fails the run | Spec: keeps CI authoritative, a missing out-of-process binary fails loudly, never silently skips |
| Precedence in the library | `jsonschema.Resolve` owns the `--schema` / inline / collection order | Spec: the policy is a json-schema/`object` concern, not an engine concern; isolates it where the next engine won't inherit it |
| Out-of-process proven by fake | A test-only `SchemaLibrary` exercises availability + `Violation.File` | Validates the seam now without depending on the `vale` binary or shipping the Vale library this PR |
| Native migration staged | Natives keep `checks.Register` until a fast-follow gives each a library | Spec: the uniform model without one giant diff; `Descriptor.Library` empty is legal in the interim |

## Out of Scope

- **Native families as libraries.** A mechanical fast-follow PR; until then
  `Descriptor.Library` is empty for native check types and the parity test allows
  it.
- **The Vale library.** `internal/checks/vale`, the `prose` check type id, and
  the real `vale` subprocess integration are a later PR; this plan only proves
  the seam.
- **Collection-scoped batching for out-of-process libraries.** Per-item is the
  first cut; batching is a follow-up issue opened on this PR's completion (spec).
- **Schema-discovery changes.** `.katalyst/schemas/` layout, discovery modes,
  and formats are untouched.
- **Renaming the `object` kind or moving its family.** `object` stays
  `structuredObject` family; only its owning package and `Library` tag change.

## Test checklist

The spec's [Test checklist](./check-library-spec.md) is the contract. The
pending tests scaffold across phases: library registry + lookup (1), the JSON
Schema library reproducing `object` byte-for-byte (2), the engine cache /
availability / precedence cutover (3), and the out-of-process seam via a fake
library (4). `make all` green at every phase boundary.
