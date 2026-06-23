# Spec — external check libraries

> **Status: planning.** This spec formalizes "external check library" as a
> first-class concept, generalizes today's JSON Schema validator into one
> instance of it, and defines the abstraction a second library (Vale) plugs
> into.

## Overview

Katalyst runs two kinds of check today, though only one is named. Most check
types (filesystem, plaintext, markdown, and the targeted structured-object
field checks) are **native**: their logic is hand-written in Katalyst. The
`object` check type is different. It delegates to a third-party engine
(`santhosh-tekuri/jsonschema`), loads named **definitions** (schemas) from
`.katalyst/schemas/`, compiles and caches them, and flattens the engine's
error tree into Katalyst violations. That delegated-engine pattern is an
**external check library**, but the codebase treats it as a one-off: the three
slices of it are scattered across `internal/validator`, `internal/config`, and
`cmd/engine.go`.

This spec names the concept, gives it a registry and a stable interface, and
reorganizes the JSON Schema pieces under it, so a second external library can
be added by dropping in a package rather than by threading new special cases
through config and the engine. Vale (prose linting over the markdown body) is
the concrete second library that proves the abstraction.

## Value

- **Extensibility.** Adding an engine (CUE, Rego, a prose linter) becomes a
  package that satisfies an interface and self-registers, the same ergonomics
  as adding a native check type today (`internal/checks/all`).
- **One mental model.** Users and contributors get a single answer to "where do
  delegated checks come from and how are their definitions discovered," instead
  of JSON-Schema-shaped assumptions baked into config and the engine.
- **Cohesion.** The JSON Schema logic stops leaking into `cmd/engine.go` (the
  compile cache, the `--schema` precedence). Each library owns its own loading,
  caching, and error mapping.

## Current State

The JSON Schema mechanism is one concept spread across three layers, none of
which names it:

| Concern | Where it lives today | What it does |
|---|---|---|
| Definition discovery | `internal/config` (`Config.Schemas map[string]string`, `loadSchemas`, `.katalyst/schemas/`, convention/explicit discovery, yaml/json/both format) | Names and locates schema files |
| Engine adapter | `internal/validator` | `Load`/`LoadYAML` compile; `Validate` runs; `flatten`/`visit` normalize `jsonschema.ValidationError` into a flat `[]Error` |
| Check binding | `internal/checks/structuredobject/object.go` + `cmd/engine.go` | `Object.Run` calls `Schema.Validate(ctx.Meta)`; the engine owns `compile`, the `map[string]*validator.Schema` cache, and the `--schema` / inline / collection resolution precedence |

Specific friction this creates:

- `cmd/engine.go` hardcodes JSON Schema: the `cache` field is typed
  `map[string]*validator.Schema`, `compile` switches on `.yaml`/`.json`
  extensions, and `checksFor` encodes the schema-resolution precedence inline
  (`engine.go:46`, `engine.go:101`). A second library has nowhere to attach.
- `internal/config` hardcodes the vocabulary: `Schemas` is a fixed field,
  `schemasSubdir = "schemas"` is a fixed constant, and `loadSchemas` is the
  only definition loader. There is no general notion of "a library's
  definitions."
- The glossary advises "prefer **schema** for what users author and
  **validator** only for the runtime check itself"
  (`docs/content/reference/glossary.md`), language that only makes sense for
  the JSON Schema case and does not generalize.

The check **registry** (`internal/checks/registry.go`) already does the half of
this we want to keep: check types self-register a `Descriptor` and a builder,
and `cmd/gendocs` + `katalyst check-types list` read it. The `object` check
type registers there today. What is missing is a peer registry for the
**libraries** themselves: their definition discovery, availability, and
lifecycle.

## Design

### The concept

An **external check library** is a check engine Katalyst does not implement
itself. It compiles a named **definition** (a JSON Schema, a Vale style
configuration, a Rego policy) loaded from files under `.katalyst/`, runs an
item's data against that definition, and returns findings Katalyst flattens
into `[]checks.Violation`. It contrasts with **native check types**, whose
logic lives entirely in Katalyst (`filesystem`, `plaintext`, `markdownbodytext`,
and the targeted `structuredobject` field checks).

"External check library" is a **provenance** axis, orthogonal to the existing
**family** axis. AGENTS.md fixes a check type's family as its *source-data kind*
(`object` is filed under `structuredObject` because it reads structured
frontmatter). That does not change. The JSON Schema library *provides* the
`object` check type, which *stays in* the `structuredObject` family. Vale
provides a prose check type that lives in the `markdownBodyText` family because
it reads the body. A library is "who supplies and runs the engine"; a family is
"what data the check reads." The two are independent and a spec reader should
not conflate them.

### The interface

A new core package `internal/checklib` owns the abstraction and a registry. It
depends on `internal/checks` (for `Context` and `Violation`) and nothing in the
library implementations; the implementations depend on it, mirroring how the
check families depend on `internal/checks` and never the reverse.

```go
package checklib

// Library is an engine Katalyst delegates checks to. It owns discovery and
// compilation of its definitions and reports its own availability. Each
// library also registers one or more check types in the checks registry;
// those check types bind a compiled Definition to an item at run time.
type Library interface {
    // Name is the stable id used in config, docs, and diagnostics
    // ("json-schema", "vale"). It never changes once published.
    Name() string

    // DefinitionDir is the subdirectory under .katalyst/ holding this
    // library's definitions ("schemas" for JSON Schema, "vale" for Vale).
    DefinitionDir() string

    // Available reports whether the library can run. An in-process library
    // returns nil unconditionally; an out-of-process one probes for its
    // binary and an acceptable version.
    Available() error

    // Compile loads and compiles one named definition.
    Compile(name string, src []byte) (Definition, error)
}

// Definition is one compiled artifact (a schema, a resolved Vale config)
// ready to evaluate items. It already knows how to pull the slice of the
// item it cares about (Meta, body, file path) out of the Context.
type Definition interface {
    Check(ctx checks.Context) []checks.Violation
}
```

`checks.Context` is already the universal per-item input (`FilePath`,
`CollectionRoot`, `Doc`, `Meta`), so the input-shape difference between
libraries is handled *inside* each `Definition`, not in the interface: the JSON
Schema definition reads `ctx.Meta`; the Vale definition reads `ctx.Doc.Body` and
`ctx.FilePath`. The interface does not need an input-shape parameter.

### The library registry

`internal/checklib` exposes `Register(Library)` and `Libraries()`, the same
shape as `checks.Register`/`Descriptors()`. Library packages self-register from
an `init()`, and a `internal/checklib/all` aggregator blank-imports each one,
paralleling `internal/checks/all`. `cmd/gendocs` and a future `katalyst
libraries list` read `Libraries()` for the reference page.

The two registries connect at the check type: a library's package both
`checklib.Register(...)`s the library and `checks.Register(...)`s the check
type(s) it provides. The `object` check type keeps its existing descriptor; it
gains nothing except a documented owner.

### Invocation models

A library is **in-process** or **out-of-process**; the distinction lives behind
the interface, with two consequences the core must account for:

- **Availability.** In-process libraries return `nil` from `Available()`.
  Out-of-process libraries probe their binary and version. The engine calls
  `Available()` before running a library's checks and surfaces a clear
  diagnostic when it fails (see Open Question 2 for the policy).
- **Batching.** An out-of-process engine pays process-startup cost per
  invocation, so running it once per item is wasteful. Vale can lint a whole
  directory in one run. This maps cleanly onto the **collection-scoped check**
  mechanism that already exists (`checks.CollectionCheck`,
  `engine.collectionChecksFor`, the second whole-collection pass in
  `cmd/check.go`): a batched library registers a collection-scoped check that
  gathers every item, runs the engine once, and maps findings back to files via
  `Violation.File`. In-process libraries stay per-item. See Open Question 5.

### Definition discovery in config

`internal/config` generalizes from one hardcoded `Schemas` field to per-library
definitions, while keeping `schemas:` working unchanged for JSON Schema:

- Each registered library contributes its `DefinitionDir()` to discovery.
  `loadSchemas` generalizes to a loop over registered libraries that scans
  `.katalyst/<dir>/` with the existing convention/explicit + format machinery
  (`scanKindDir`, `normDiscovery`, `formatExts`).
- `Config.Schemas map[string]string` is retained as the JSON Schema library's
  definition map (and the canonical name for back-compat), with a general
  accessor (`Config.Definitions(library) map[string]string`) layered over it.
- This keeps `config` from importing `checklib` (which would cycle, `checklib`
  imports `checks` imports `config`'s `CheckType`): config consults a
  parse-time allowlist of `(library name -> definition dir)`, the same pattern
  it already uses for `knownStorageTypes` (config validates storage `type`
  against an allowlist it owns, without importing `internal/storage`). The
  allowlist grows when a library is added.

### Engine integration

`cmd/engine.go` stops being JSON-Schema-shaped:

- The cache generalizes from `map[string]*validator.Schema` to a cache of
  compiled `checklib.Definition` keyed by `(library, definitionPath)`. `compile`
  delegates to the owning library's `Compile`, dropping the `.yaml`/`.json`
  extension switch (each library decides how to parse its own files).
- The `--schema` flag and the inline `schema:` frontmatter directive stay as
  JSON-Schema sugar (documented as such), routed through the general path. The
  resolution precedence in `checksFor` becomes "ask the JSON Schema library to
  resolve a definition," isolating the precedence policy to that library rather
  than to the engine. Other libraries bind definitions through their check
  instance's config fields only.

### Worked examples

**JSON Schema (in-process, structured-object, per-item).** `internal/validator`
moves to `internal/checklib/jsonschema`. It implements `Library` (`Name()`
returns `"json-schema"`, `DefinitionDir()` returns `"schemas"`, `Available()`
returns `nil`, `Compile` is today's `Load`/`LoadYAML`). The compiled schema
implements `Definition.Check` by calling today's `Validate` and adapting
`[]validator.Error` into `[]checks.Violation` with `checks.LookupLine`, the
adaptation that lives in `structuredobject/object.go:Run` today. The `object`
check type stays in `internal/checks/structuredobject`, in the
`structuredObject` family, now documented as "provided by the JSON Schema
library."

**Vale (out-of-process, body-text, batched).** A new
`internal/checklib/vale` implements `Library` with `Name() == "vale"`,
`DefinitionDir() == "vale"`, and an `Available()` that runs `vale --version`
and checks for a usable binary. Its `Definition` shells out to `vale --output
JSON` over the item body (or the collection's files, batched) and maps Vale's
`{Line, Span, Check, Message, Severity}` records onto `checks.Violation`
(`Line` directly; Vale's `error`/`warning` severity onto `checks.Severity`). It
provides a new `prose` (working name) check type in the `markdownBodyText`
family. Vale's "definition" is a resolved configuration bundle rather than many
per-item-named files, so it exercises the "a library may have one definition,
not a named set" path the interface allows (see Open Question 3).

## Open Questions

1. **Definition-directory layout per library.**
   **Context.** Today JSON Schema definitions live in `.katalyst/schemas/`.
   Each new library needs a home for its definitions. **Choices & tradeoffs.**
   (a) Each library owns a top-level subdir it names via `DefinitionDir()`
   (`.katalyst/schemas/`, `.katalyst/vale/`): flat, matches the existing
   `schemas/` and `storage/` siblings, but grows the top level of `.katalyst/`
   with every library. (b) A single `.katalyst/libraries/<name>/` namespace:
   tidier top level, but breaks the existing `schemas/` path and forces a
   migration. **Recommendation.** (a), keep `schemas/` where it is (no
   migration, back-compat) and let each library claim a sibling subdir. Revisit
   only if the top level gets crowded. Your call.

2. **Failure policy when an out-of-process library is unavailable.**
   **Context.** If the `vale` binary is missing or too old, `Available()`
   fails. Native checks never had this state. **Choices & tradeoffs.** (a)
   **Hard error** (non-zero exit): reproducible CI, but a contributor without
   Vale installed cannot run *any* check locally. (b) **Skip with a warning**:
   local ergonomics, but a misconfigured CI silently stops enforcing prose
   rules, exactly the drift Katalyst exists to prevent. (c) **Configurable per
   library** (`onMissing: error|skip|warn`, default `error`): flexible, more
   config surface. **Recommendation.** (c) defaulting to `error`, so CI is
   safe by default and a contributor can opt into `skip` locally. Open to (a)
   for simplicity if you would rather not add the knob now.

3. **Named definitions vs. a single definition.**
   **Context.** JSON Schema has many named definitions selected per item
   (`schema: book`). Vale has essentially one resolved configuration. The
   interface allows both, but config and the check instance need to express
   "this library has no per-item name." **Choices & tradeoffs.** (a) Treat the
   single-definition case as a library that exposes exactly one definition with
   a fixed name (e.g. `default`): uniform model, slightly artificial for Vale.
   (b) Make `DefinitionDir()` optional (empty), and a library with no dir
   resolves its one definition from a fixed config file (`.katalyst/vale.ini`
   or a referenced `.vale.ini`): more honest for Vale, a second code path.
   **Recommendation.** (b), it matches how Vale is actually configured and the
   interface already tolerates an empty `DefinitionDir()`. Worth confirming
   before it hardens.

4. **Naming.** **Context.** The package is proposed as `internal/checklib`
   with the concept "external check library"; the Vale check type is `prose`
   (working name). **Choices & tradeoffs.** `checklib` is terse and parallels
   `checks`; alternatives are `engine`/`checker`/`provider` (each already
   overloaded in this codebase: `cmd/engine.go`, the "check engine"). The
   concept term "external check library" is precise but long; "check library"
   or "check engine" are shorter but blur into existing usage.
   **Recommendation.** Keep `internal/checklib` and "external check library" in
   prose, shortened to "check library" after first use on a page. Decide the
   Vale check type id (`prose` vs `vale` vs `prose_lint`) when that library is
   built; it is a wire contract once shipped.

5. **Per-item exec vs. collection-scoped batch for out-of-process libraries.**
   **Context.** Out-of-process engines amortize startup by running once over
   many files. The collection-scoped check pass already exists for exactly
   "run once over the whole collection." **Choices & tradeoffs.** (a) Vale
   registers as a **collection-scoped** check (`CollectionCheck`): one Vale run
   per collection, findings mapped back via `Violation.File`, reuses existing
   machinery, but a single-item selector (`katalyst check notes/foo`) still
   triggers a whole-collection Vale run (the documented behavior of
   collection-scoped checks today). (b) Vale registers **per-item** and the
   library internally caches a batched run keyed by collection: precise
   single-item runs, but new caching logic and a lifecycle the engine does not
   model today. **Recommendation.** (a), reuse the collection-scoped pass and
   accept the whole-collection run under a narrow selector (it already holds for
   `filesystem_unique_filename`). Only build (b) if single-item latency becomes
   a real complaint.

## Documentation updates

**Developer docs**

- **`AGENTS.md` (root):** new section on external check libraries, the
  `internal/checklib` core + per-library packages + `internal/checklib/all`
  aggregator pattern, and the rule that *library is provenance, family is
  source-data kind* (they are orthogonal). Note that a library both
  `checklib.Register`s itself and `checks.Register`s its check type(s).
- **`internal/checklib/doc.go`:** package doc establishing the `Library` /
  `Definition` contract, the two registries' relationship, and the in-process
  vs out-of-process distinction (availability + batching). This is where the
  rejected alternatives below graduate when the spec retires.
- **`internal/checklib/jsonschema` doc + `internal/checks/structuredobject`:**
  reframe the moved `validator` package as the JSON Schema library; note that
  `object` is the check type it provides.
- **`internal/config` doc/README:** generalize the "schemas" language to
  "per-library definitions," documenting the allowlist pattern (mirrors
  `knownStorageTypes`).
- **`.cursor/skills/`:** add an "adding an external check library" note if an
  authoring skill exists for check types; otherwise fold into AGENTS.md.

**User docs**

- **`docs/content/deep-dives/core-concepts.md`:** under **Check**, distinguish
  native check types from external check libraries (the delegated-engine
  case), keeping it backend-agnostic.
- **`docs/content/deep-dives/domain-model.md`:** generalize the **Schema**
  section and the resolver discussion from "compiled JSON Schema" to "a
  library's compiled definition," with JSON Schema as the worked instance.
- **`docs/content/reference/glossary.md`:** add **External check library**,
  **Definition**, and **Native check type**; revise the **Schema**,
  **Validator**, and **Resolver** entries to point at the general concept.
- **`docs/content/reference/configuration.md`:** document per-library
  definition discovery (`.katalyst/<library>/`) alongside `schemas:`.
- **`docs/reference/check-types/` (generated):** regenerate via `make docs-gen`
  once any new check type ships; do not hand-edit.
- **A `katalyst libraries list` reference** if that command is added, parallel
  to `check-types list`.

## Rejected alternatives

- **Leave it as-is and only document the concept.** Names the idea but leaves
  the JSON Schema assumptions in `cmd/engine.go` and `internal/config`, so the
  second library still threads special cases through both. Rejected because the
  stated goal is to make a second library a drop-in.
- **One registry for everything (fold libraries into the check registry).**
  Check types and libraries have different lifecycles (a library owns
  definition discovery and availability; a check type owns a per-item or
  per-collection run). Overloading one registry blurs them. A peer registry
  keeps each concern's source of truth single, the same reason storage types
  and check types are separate today.
- **A generic input-shape parameter on `Definition.Check`.** Pass the library a
  declared "this reads Meta / body / path" descriptor. Unnecessary:
  `checks.Context` already carries all per-item data, and each `Definition`
  pulls what it needs. The descriptor would be redundant indirection.
- **In-process-only libraries.** Simpler (no availability or batching
  concerns), but excludes the entire class of mature external linters (Vale,
  language tools) that only exist as binaries, which is the concrete motivation
  here.

## Test checklist

This spec's build contract; scaffold as failing tests first (TDD, per
AGENTS.md). Detailed phasing lives in the plan.

- [ ] `checklib.Register` / `Libraries()` round-trip; duplicate-name
      registration is rejected.
- [ ] `internal/checklib/all` blank-import wires in every library, parallel to
      `internal/checks/all`.
- [ ] JSON Schema library: `Name()=="json-schema"`, `DefinitionDir()=="schemas"`,
      `Available()==nil`; `Compile` accepts the existing `.json` and `.yaml`
      fixtures; `Definition.Check` reproduces today's `object` violations
      byte-for-byte (reuse `internal/validator/testdata`).
- [ ] `cmd/engine` cache is keyed by `(library, path)` and compiles each
      definition once per process (preserves the "compile once per absolute
      path" invariant in `internal/checks/README.md`).
- [ ] `--schema` and inline `schema:` precedence is unchanged for the JSON
      Schema library (existing `cmd` tests pass without modification).
- [ ] `config` discovers definitions for every registered library's
      `DefinitionDir()`; `schemas:` continues to populate `Config.Schemas`.
- [ ] An unavailable out-of-process library follows the Open Question 2 policy
      (a fake library whose `Available()` errors).
- [ ] A batched (collection-scoped) library maps findings back to the right
      file via `Violation.File` (a fake library, no Vale binary needed).
