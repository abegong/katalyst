# Spec — CheckLibrary

> **Status: planning.** This spec introduces `CheckLibrary` as the single
> abstraction behind every check type, generalizes today's JSON Schema
> validator into the first schema-backed library, and defines the out-of-process
> seam a prose linter (Vale) plugs into. Native check families migrate onto the
> same abstraction as a staged fast-follow.

## Overview

Katalyst runs two kinds of check today, though only one is named. Most check
types (filesystem, plaintext, markdown, and the targeted structured-object
field checks) are **native**: their logic is hand-written. The `object` check
type is different: it delegates to a third-party engine
(`santhosh-tekuri/jsonschema`), compiles a named **schema**, runs an item's
metadata against it, and flattens the engine's error tree into violations. That
delegated-engine pattern has no name, and its three slices are scattered across
`internal/validator`, `internal/config`, and `cmd/engine.go`.

This spec names the abstraction **`CheckLibrary`**: a cohesive bundle of check
types from one provider. Native families and external engines are both
`CheckLibrary`s; the schema-backed ones (JSON Schema today, Vale next)
additionally compile schemas and report their own availability. One registry in
`internal/checks` replaces the implicit split, so a new engine is a package that
satisfies an interface and self-registers, the same ergonomics as adding a check
type today.

## Value

- **Extensibility.** Adding an engine (JSON Schema, Vale, later CUE or Rego)
  becomes a package implementing one interface and self-registering, no new
  special cases threaded through `config` and the engine.
- **One mental model.** Every check type belongs to exactly one `CheckLibrary`.
  Users and contributors get a single answer to "where do checks come from, and
  how are their schemas discovered."
- **Cohesion.** The JSON Schema specifics (compile cache, `--schema`
  precedence, error mapping) leave `cmd/engine.go` and live with the library
  that owns them.
- **A real out-of-process seam.** Modeling availability and invocation lets
  Katalyst delegate to mature external linters (Vale) that only exist as
  binaries, not just in-process Go libraries.

## Current State

The JSON Schema mechanism is one concept spread across three layers, none of
which names it:

| Concern | Where it lives today | What it does |
|---|---|---|
| Schema discovery | `internal/config` (`Config.Schemas map[string]string`, `loadSchemas`, `.katalyst/schemas/`, convention/explicit discovery, yaml/json/both format) | Names and locates schema files |
| Engine adapter | `internal/validator` | `Load`/`LoadYAML` compile; `Validate` runs; `flatten`/`visit` normalize `jsonschema.ValidationError` into a flat `[]Error` |
| Check binding | `internal/checks/structuredobject/object.go` + `cmd/engine.go` | `Object.Run` calls `Schema.Validate(ctx.Meta)`; the engine owns `compile`, the `map[string]*validator.Schema` cache, and the `--schema` / inline / collection resolution precedence |

Specific friction:

- `cmd/engine.go` hardcodes JSON Schema: the `cache` field is typed
  `map[string]*validator.Schema` (`engine.go:25`), `compile` switches on
  `.yaml`/`.json` extensions (`engine.go:46`), and `checksFor` encodes the
  schema-resolution precedence inline (`engine.go:101`). A second engine has
  nowhere to attach.
- The check **registry** (`internal/checks/registry.go`) already models the
  half worth keeping: check types self-register a `Descriptor` and a builder,
  and `cmd/gendocs` + `katalyst check-types list` read it. What is missing is
  the notion that those check types have an **owning provider** with its own
  availability and (sometimes) schema compilation.
- The glossary defines **Schema** as "A JSON Schema document"
  (`docs/content/reference/glossary.md`), conflating the Katalyst concept of a
  collection's schema with one engine's file format.

## Design

### The concept

A **`CheckLibrary`** is a cohesive bundle of check types from one provider.
Every check type belongs to exactly one library:

- **Native libraries** wrap hand-written check types: `filesystem`, `plaintext`,
  `markdownbodytext`, `structuredobject`. They compile no schema and are always
  available.
- **Schema-backed libraries** delegate to an external engine that compiles a
  named **schema** and runs items against it: `json-schema` today, `vale` next.

A **Schema** is elevated to the Katalyst concept it always implicitly was: the
definition of a collection's shape and constraints. JSON Schema and a Vale style
configuration are two *expressions* of a schema, not the concept itself. This is
the vocabulary shift the glossary needs.

**library is orthogonal to family.** AGENTS.md fixes a check type's **family**
as its *source-data kind* (`object` is `structuredObject` because it reads
structured frontmatter). That is unchanged. A library is *who supplies and runs
the engine*; a family is *what data the check reads*. The two are independent,
and a single family spans libraries: the `structuredObject` family holds both
`object` (the `json-schema` library) and `object_required_field` (the
`structuredobject` native library); the `markdownBodyText` family will hold both
the native markdown checks and `vale`'s prose check.

### The interface

The abstraction lives in `internal/checks` (it *is* the checks-provider concept,
not a sibling package). A capability interface keeps native libraries free of
schema machinery:

```go
package checks

// CheckLibrary is a provider of check types. Native libraries implement just
// this; schema-backed libraries also implement SchemaLibrary.
type CheckLibrary interface {
    // Name is the stable id used in diagnostics and docs ("filesystem",
    // "json-schema", "vale"). It never changes once published.
    Name() string

    // Available reports whether the library can run. Native and in-process
    // libraries return nil; an out-of-process library probes for its binary
    // and an acceptable version. A non-nil error fails the run.
    Available() error
}

// SchemaLibrary is a CheckLibrary that compiles named schemas from source
// bytes. The engine caches the result per (library, path).
type SchemaLibrary interface {
    CheckLibrary
    CompileSchema(name string, src []byte) (Schema, error)
}

// Schema is one compiled artifact (a JSON Schema, a resolved Vale config)
// ready to evaluate items. It pulls the slice of the item it needs (Meta,
// body, path) out of the Context itself.
type Schema interface {
    Check(ctx Context) []Violation
}
```

`Context` is already the universal per-item input (`FilePath`,
`CollectionRoot`, `Doc`, `Meta`), so the input-shape difference between engines
is handled *inside* each `Schema` implementation, not in the interface: the JSON
Schema schema reads `ctx.Meta`; the Vale schema reads `ctx.Doc.Body` and
`ctx.FilePath`.

### One registry

`internal/checks` gains library registration alongside the existing check-type
registration. A library `init()` registers itself and its check types; an
`internal/checks/all` aggregator blank-imports each library (it already exists
to wire the families). The registry answers the new question the engine needs:
**given a check type's `kind`, which library owns it?** `cmd/gendocs` and a
future `katalyst libraries list` read the library set for the reference.

### Schema resolution: flat, resolved by kind

`.katalyst/schemas/` stays **flat and unchanged**: a schema is a named resource
(name → absolute path), discovered exactly as today
(`config.loadSchemas`, `Config.Schemas`). There is **no migration** and no
per-library subdirectory. Which library compiles a given schema is resolved at
the **binding site**, from the referencing check type's `kind`:

- `kind: object, schema: book` resolves `book` through the `json-schema` library.
- `kind: prose, schema: house-style` resolves `house-style` through the `vale`
  library.

A `CheckLibrary` therefore owns no directory; `config` needs almost no change
beyond doc language. A schema file in the wrong format for its binding fails at
`CompileSchema`, with the library named in the error.

### Invocation and availability

A library is **in-process** or **out-of-process**, hidden behind the interface,
with two consequences:

- **Availability is a hard error.** The engine calls `Available()` before
  running a library's checks. If a configured out-of-process library's binary is
  missing or too old, the run fails with a non-zero exit and a clear diagnostic.
  This keeps CI authoritative: a misconfigured environment fails loudly rather
  than silently skipping enforcement.
- **Invocation starts per-item.** An out-of-process engine pays process-startup
  cost per run, but the first cut runs **per item** (the simplest correct path),
  consistent with how every per-item check works today. Batching a whole
  collection into one external invocation (mapping findings back via
  `Violation.File`, reusing the collection-scoped check pass) is a performance
  optimization tracked as a follow-up issue, not built here.

### Engine integration

`cmd/engine.go` stops being JSON-Schema-shaped:

- The cache generalizes from `map[string]*validator.Schema` to compiled
  `checks.Schema` keyed by `(library, path)`. `compile` looks up the owning
  library by `kind` and calls its `CompileSchema`, dropping the
  `.yaml`/`.json` extension switch (each library parses its own bytes).
- `--schema <path>` and the inline `schema:` directive stay as **`json-schema`
  sugar** (documented as such), routed through the general path. The resolution
  precedence moves from the engine into the `json-schema` library.

### Worked example: Vale (out-of-process)

A new `internal/checks/vale` package implements `SchemaLibrary`: `Name()` is
`"vale"`, `Available()` runs `vale --version` and checks for a usable binary,
and `CompileSchema` resolves a Vale configuration. Its `Schema.Check` shells out
to `vale --output JSON` over the item body and maps Vale's `{Line, Span, Check,
Message, Severity}` records onto `Violation` (line directly; Vale's
`error`/`warning` onto `checks.Severity`). It provides a check type in the
`markdownBodyText` family.

The check type id (`prose` vs `vale` vs `prose_lint`) is a wire contract decided
when Vale is implemented; this spec uses `prose` as a placeholder. The full Vale
implementation is sequenced after this PR (see Scope); the out-of-process seam is
proven here by a fake out-of-process library in tests.

### Native families as libraries (staged)

The four native families become `CheckLibrary`s (no `SchemaLibrary`, trivial
`Available()`), so the registry is uniform and "every check type has an owning
library" holds without exception. This is mechanical and lands as a fast-follow
PR; until then the native check types keep registering through the existing
`checks.Register(Descriptor, builder)` path and coexist with the library
registry.

## Scope and sequencing

- **This PR.** Introduce `CheckLibrary` / `SchemaLibrary` / `Schema` and library
  registration in `internal/checks`; move `internal/validator` to
  `internal/checks/jsonschema` and relocate `structuredobject/object.go`'s
  binding there as the first `SchemaLibrary`; generalize the engine cache and
  `--schema` handling; prove the out-of-process seam with a fake library in
  tests. Native families and `cmd`/docs continue to pass unchanged.
- **Fast-follow PR.** Migrate the four native families onto `CheckLibrary`.
- **Follow-up issue.** Collection-scoped batching for out-of-process libraries:
  [#68](https://github.com/abegong/katalyst/issues/68).
- **Later.** The full Vale library; any additional engines (CUE, Rego).

## Open Questions

_None._ Remaining choices are deferred wire-contract details (the Vale check
type id) decided when that library is built, noted inline in Design.

## Documentation updates

**Developer docs**

- **`AGENTS.md` (root):** new section on `CheckLibrary`, the one-registry model,
  the `internal/checks/all` aggregator, and the rule that *library is provenance,
  family is source-data kind* (orthogonal). Note that a library registers both
  itself and its check types, and that schema-backed libraries implement
  `SchemaLibrary`.
- **`internal/checks` doc (`checks.go` / a `doc.go` when long):** the
  `CheckLibrary` / `SchemaLibrary` / `Schema` contract, availability as a hard
  error, and the per-item-first invocation note. Where the rejected alternatives
  below graduate when the spec retires.
- **`internal/checks/jsonschema` doc:** reframe the moved `validator` package as
  the JSON Schema library; note that `object` is the check type it provides and
  that `--schema`/inline `schema:` are its sugar.
- **`internal/config` doc/README:** clarify that schemas stay flat and
  library-agnostic on disk; the library is resolved at the binding site.

**User docs**

- **`docs/content/deep-dives/core-concepts.md`:** under **Check**, name the
  `CheckLibrary` concept and the native vs schema-backed distinction,
  backend-agnostic.
- **`docs/content/deep-dives/domain-model.md`:** generalize the **Schema**
  section and the **Resolver** from "compiled JSON Schema" to "a library's
  compiled schema," with JSON Schema as the worked instance.
- **`docs/content/reference/glossary.md`:** add **CheckLibrary** and revise
  **Schema** (the Katalyst concept), **Schema directive**, **Validator**, and
  **Resolver** to point at the general concept.
- **`docs/content/reference/configuration.md`:** note that the schema a check
  references is compiled by the library its `kind` selects.
- **`docs/reference/check-types/` (generated):** regenerate via `make docs-gen`
  when any new check type ships; never hand-edit.

## Rejected alternatives

- **Two registries (a separate library registry beside the check registry).**
  An earlier draft of this spec. Check types and libraries have one home; a
  second registry splits "which check types exist" from "who owns them" and
  forces the engine to consult both. One registry where every check type has an
  owning library keeps the source of truth single.
- **A `DefinitionDir()` per library / namespaced `.katalyst/schemas/<lib>/`.**
  Tidier on paper, but it migrates every existing schema path (including the
  dogfood `schemas/page.json`) for no functional gain: the binding's `kind`
  already names the library, so a flat namespace resolves unambiguously.
- **A generic input-shape parameter on `Schema.Check`.** Declaring "this reads
  Meta / body / path" is redundant: `Context` already carries all per-item data
  and each `Schema` pulls what it needs.
- **In-process-only libraries.** Simpler (no availability or invocation
  concerns), but excludes the mature external linters (Vale) that motivated the
  abstraction.
- **Leaving native check types outside `CheckLibrary`.** Smaller change, but
  permanently keeps two registration mechanisms and the "every check type has an
  owning library" invariant fails for the majority of check types. Staged
  migration gets the uniform model without a giant single diff.

## Test checklist

This spec's build contract for the first PR; scaffold as failing tests first
(TDD, per AGENTS.md). Phasing lives in the plan.

- [ ] Library registration round-trips; a duplicate library name is rejected;
      the registry resolves a check type's `kind` to its owning library.
- [ ] `internal/checks/all` wires in every library.
- [ ] `json-schema` library: `Name()=="json-schema"`, `Available()==nil`;
      `CompileSchema` accepts the existing `.json` and `.yaml` fixtures;
      `Schema.Check` reproduces today's `object` violations byte-for-byte (reuse
      `internal/validator/testdata`).
- [ ] `cmd/engine` cache is keyed by `(library, path)` and compiles each schema
      once per process (preserves the "compile once per absolute path" invariant
      in `internal/checks/README.md`).
- [ ] `--schema` and inline `schema:` precedence is unchanged (existing `cmd`
      tests pass without modification).
- [ ] `config` schema discovery is unchanged; `.katalyst/schemas/` stays flat.
- [ ] A configured but unavailable out-of-process library fails the run with a
      non-zero exit (a fake library whose `Available()` errors).
- [ ] The out-of-process seam works end to end with a fake library (no Vale
      binary needed): `Available()` gates the run and findings map back to the
      right file via `Violation.File`.
