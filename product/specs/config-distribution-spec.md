# Spec — config distribution (objects own their config)

> **Status: planning.** The companion to `collection-reorg-spec.md` (Spec 1).
> Dissolves the centralized `config` package so each object type validates and
> instantiates itself from its own config, leaving only a thin generic loader.
> This is the change that **removes the `config → query` interleaving Spec 1
> tolerates**, and retires the `normalizeCheck`/registry parity duplication.

## Overview

`internal/project/config` is katalyst's "block config" layer: a single
1,200-line package that defines a typed struct for every object's config and
holds all the parse/validate logic for all of them. This spec inverts that —
each check type, storage type, collection, and schema owns its own config shape
and validation, the way Great Expectations' fluent datasources own theirs — and
shrinks `config` to a generic loader plus a `kind`→owner dispatch. It changes
*who parses config*, not the on-disk `.katalyst/` format.

## Value

Today, adding or changing a check means editing it in **two** places that must
agree: the check's own file (its `Descriptor`, runtime, `Builder`) and the
central `config.normalizeCheck` switch (its args and validation). A test
(`registry_test.go`) exists only to police that they stay in sync. The same
double-bookkeeping governs storage types (`knownStorageTypes`) and collections
(`buildCollection`). Distributing config makes each object the single source of
truth for its own config: one place to add a check, no parity test, no
1,200-line god package, and — per Spec 1 — a clean `storage/collection/` layering
instead of a tolerated cross-tree edge.

## Current State

`config` centralizes four object families' config:

- **Checks.** `config.normalizeCheck` is a ~200-line `switch` over every check
  `kind`, validating each one's arguments inline (`object_required_field`
  requires `field`; `filesystem_name_case` requires a valid `style`; …). The
  parsed result is `config.CheckInstance` — a **union of every check's fields**
  (`Field`, `Schema`, `Values`, `Min`, `Max`, `MinLength`, `Style`, `Target`,
  `Transform`, `Prefix`, `Suffix`, …), most nil for any given check. `CheckType`
  constants re-enumerate the registry's kinds. The blast radius is wide: **every
  check family package imports `config` solely to read its fields off this
  union** — `filesystem`, `structuredobject`, `markdownbodytext`, `plaintext`,
  and `jsonschema` all appear in `config`'s importer list for no reason but
  `CheckInstance`. Distributing args removes `config` from ~7 packages at once,
  which is why checks-first (below) is where most of the coupling lives.
- **Storage.** `knownStorageTypes` allowlists backend kinds; `buildInstance`
  validates and constructs `StorageInstance`.
- **Collections.** `buildCollection`/`buildVariants`/`resolveQuery` parse and
  validate the collection block, including variant `when` predicates.
- **Schemas.** `loadSchemas`/`scanKindDir` discover and resolve schema files.

The telling part is that **half the distributed pattern already exists.** The
checks registry has each check type self-register its `Descriptor` and a
`Builder func(config.CheckInstance) Check` (`checks/registry.go`). The check type
already owns its metadata and its constructor — it just doesn't own its
*parsing*. `config.normalizeCheck` parses the raw args into the union struct, and
the `Builder` reads them back out. The registry is a working `_type_lookup`; it
is fed pre-digested config instead of raw config.

This is precisely the shape Great Expectations moved **away from**: central
marshmallow/`DataContextConfig` schemas → per-object Pydantic models where each
datasource/asset owns its fields and validators and a `_type_lookup` resolves
subtypes. Their result was ~85% less config boilerplate and "instantiation is the
config." Katalyst is one registry-call short of the same design.

## Design

The principle: **an object owns its config.** There is no central typed-config
library — only generic decoding (YAML/JSON → raw nodes) and a `kind`/`type`
dispatch to the owning type, which decodes its own typed args, validates them,
and constructs itself.

### Target shape

- **A thin loader** (the residue of `config`): discover `.katalyst/`, read the
  files into raw `yaml.Node`s, and *assemble* — dispatch each block to its
  owner's parser and wire the results together. This is katalyst's `DataContext`.
- **Per-object parsers**, registered the way `Descriptor`s already are. Each
  check type registers a parser that decodes its **own** args struct from a raw
  node and returns the runnable `Check` (validation and construction fused,
  GX-style). The same for each storage type, the collection, and schemas.
- **`config.CheckInstance` dissolves** into per-check arg structs, each beside its
  check (e.g. `structuredobject.RequiredField{Field string}`), unmarshalled and
  validated by that check. The union struct and the `CheckType` constant list go
  away; the registry is the only enumeration.

Concretely, the registry call changes from consuming a pre-parsed instance to
owning the parse:

```
// today: config parses, Builder reads the union back out
Register(desc, func(ci config.CheckInstance) Check { ... }, ...)

// target: the check type owns its args and validation
Register(desc, func(raw yaml.Node) (Check, error) {
    var a requiredFieldArgs            // this check's own shape
    if err := raw.Decode(&a); err != nil { return nil, err }
    if a.Field == "" { return nil, errors.New(`object_required_field requires "field"`) }
    return RequiredField{Field: a.Field}, nil
}, ...)
```

`yaml.v3`'s deferred `Node.Decode` is the Go stand-in for Pydantic's per-subtype
validation: the loader holds the block as a `Node` and hands it to the owner,
which decodes into its own type.

### The dependency design (and how it pays off Spec 1)

Distributing config *seems* to risk a `collection ↔ checks` cycle: collections
contain checks, and collection-scoped checks reference collections. Implementing
Spec 1 showed the codebase **already avoids this, and the resolution is just to
preserve the existing boundary.** Two facts from the code:

- `config.Collection.Checks` is `[]CheckInstance` — a collection holds its checks
  as **config data, never built `Check` values**.
- Building is lazy, at the top layer: `cmd/engine.go:176` calls `checks.Build(ch)`
  (and `:234` `BuildCollection`) at run time, not at load time.

So nothing holds a `Collection`-that-contains-`Check`-values; the `Collection`→
`Check` relationship is data, and the *engine* is the assembler. The distributed
version keeps exactly that shape: a `Collection` carries its checks as **raw
check config** (the deferred `yaml.Node`s), and the registry builds them lazily
at the engine boundary. `checks → collection` (for the `CollectionCheck` target
type) stays one-directional; `collection` never imports `checks`. No cycle.

**The anti-pattern to avoid** is eager building: if the loader built
`[]checks.Check` onto `Collection`, and `Collection` lives in `storage/collection`,
you would get `storage/collection → checks → storage/collection`. Lazy building
is therefore load-bearing, not incidental — the spec commits to it.

This is also exactly what **removes Spec 1's interleaving**. The variant `when`
predicate grammar (`query`) is used only while parsing a collection's variant
config — which, once the collection owns that parsing, lives under
`storage/collection/`. So `collection → query` becomes *intra-`collection`*, and
the central `config → query` edge that made `project/` and `storage/` reference
into each other simply ceases to exist. Spec 1's tolerated compromise is retired
here, by construction.

### What stays centralized (the irreducible bits)

- **Generic decoding and `.katalyst/` discovery** — the loader.
- **The dispatch registry** — the `kind`→parser table. Small, and we already have
  it for checks and storage.
- **`Descriptor`s stay** as the documentation source; `cmd/gendocs` and
  `katalyst check-types` are unaffected (metadata didn't move, only parsing did).

### Phasing

The migration is incremental and stays green throughout, because the loader can
dispatch already-migrated kinds to their owners while the rest run the legacy
path:

0. **Relocate the types** — a mechanical, Spec-1-style pre-step: move the
   `Collection` type to `storage/collection` and `StorageInstance` to `storage`
   (storage root is already config-free post-Spec-1), leaving the parsing behind
   in `config` for now. This sets the homes before redistributing parse logic.
1. **Checks first** — the registry already exists; convert `Builder` to own its
   parse, one family at a time (`structuredobject`, `markdownbodytext`,
   `filesystem`, `plaintext`), deleting `normalizeCheck` cases and `CheckInstance`
   fields as each family lands. Drop the parity test when the switch is gone.
2. **Storage types** — `knownStorageTypes`/`buildInstance` → the storage registry
   owns instance config.
3. **Collections and schemas** — `buildCollection`/`buildVariants`/`loadSchemas`
   → owned by the collection package and the schema handling; this is the step
   that dissolves `config → query`.
4. **Collapse** what remains of `config` into the thin loader; rename/rehome it.

## Resolved by implementing Spec 1

These two were open before Spec 1 landed; the hands-on move settled them.

- **Where the types and loader live.** The concept *types* redistribute to their
  concept packages; the *loader* is the assembler on top. No `config` package
  survives.

  | Today (central `config`) | Distributed home |
  |---|---|
  | `CheckInstance` + `normalizeCheck` | per-check arg structs, in each check family |
  | `Collection` | `storage/collection` (the readers already consume it) |
  | `StorageInstance` | `storage` (backend config; root is config-free post-Spec-1) |
  | `.katalyst/` discovery + YAML + assembly | `project` — the DataContext/loader |

- **No shared "model" package is needed.** The earlier worry — a neutral
  `Check`-interface package to break `collection ↔ checks` — is moot given the
  lazy-build boundary above: `Collection` holds raw check config, not `Check`
  values, so `collection` never imports `checks`. Keep the assembler in the
  engine; add no shared-interface package.

## Open Questions

1. **Validation-message consistency.** Central `normalizeCheck` gives uniform
   error phrasing; per-object parsers must not drift in tone. A tiny shared helper
   set (required-field, enum-of) keeps them consistent without recentralizing.
2. **Does the on-disk schema (`.katalyst/schemas/page.json`-style validation of
   the config files themselves) change?** No intent to; confirm the dogfood
   `pages` collection still validates after the loader is rehomed.

## Documentation updates

- **`docs/content/deep-dives/`** — a "configuration architecture" note (or a
  section in `collections.md`/`domain-model.md`) on the object-owns-its-config
  design and the GX-fluent precedent; this is the durable *why*.
- **`internal/.../AGENTS.md`** — each object family's `AGENTS.md` gains the "this
  type owns its config parsing" convention; the (old) `config` `AGENTS.md`
  collapses into the loader's.
- **`.cursor/skills/add-katalyst-check-type/SKILL.md`** — rewrite: adding a check
  type is now one file (Descriptor + args + parser + runtime), no
  `config.normalizeCheck` step. This skill getting *shorter* is the headline proof
  the change worked.
- **`docs/content/reference/configuration.md`** — confirm the key-by-key surface
  still matches once parsing is distributed (format unchanged).
- **`product/specs/domain-model-terminology-matrix.md`** — update the Config row;
  "centralized typed config" is no longer accurate.
- **Generated reference** — `make docs-gen-check` stays byte-identical
  (`Descriptor`s unchanged).

## Rejected alternatives

- **Keep config centralized; just split the file.** Rejected: the 1,200 lines are
  a symptom, not the disease. The disease is that config knows every object's
  shape — splitting files keeps the double-bookkeeping and the parity test.
- **Generate `CheckInstance`/`normalizeCheck` from the registry.** Rejected:
  codegen to keep two representations in sync is a workaround for them being two
  representations. The fix is one representation, owned by the object.
- **A JSON-Schema for the whole `.katalyst/` config, validated centrally.**
  Rejected as the *primary* mechanism: it recentralizes the shape knowledge in a
  schema file and still can't construct the objects. (A generated schema *derived*
  from the per-object parsers, for editor support, is a fine future add-on.)
- **Do this before Spec 1.** Rejected: Spec 1 is a mechanical relocation that
  verifies trivially; this is a semantic change touching every object type.
  Sequencing the safe move first keeps it from being held hostage — and Spec 1
  already names this spec as what retires its one compromise.
