# Inspector layers — raw-source and collection inspectors over the storage seam

> **Status: planning.** Splits inspectors into two layers with **distinct
> data-referencing machinery**: a **raw-source** layer that measures a backend
> store directly (a filesystem, a file database, …) addressed by backend-native
> locator, and a **collection** layer that measures *configured* collections,
> addressed by domain identity and probing items through **checks as a common
> substrate**. Builds on the storage seam from
> [storage-layer-spec.md](./storage-layer-spec.md) (#31) and evolves the
> single-scope inspector model in [inspect-spec.md](./inspect-spec.md). Sets the
> direction that the inspector cull (#25) executes against. Plan: TBD.

## Overview

[inspect-spec.md](./inspect-spec.md) introduced inspectors as the descriptive
dual of checks: a check asserts a predicate, an inspector reports the
distribution that predicate would be tested against. It shipped 11 inspectors,
each implemented as a pure function of a single `inspect.Corpus` — a flat walk
of `*.md` files under a path:

```go
type Inspector interface {
    Name() string
    Inspect(Corpus) Evidence
}
```

That model has one input shape and one addressing scheme (a file's path
relative to the scope root). It predates the storage layer. Now that #31 has
landed a real seam between a backend store and the domain model
(`StorageType` / `StorageInstance` / `CollectionDefinition`), inspectors can —
and should — be organized along the same seam.

Two facts the corpus model conflates:

1. **Some inspectors describe a raw store** — "what is physically present, how
   does it parse, what are its filename conventions" — *before* any collection
   configuration is meaningful. These belong to the backend, addressed by its
   native locator.
2. **Some inspectors describe a configured collection** — "across the items of
   `notes/books`, how often is `rating` present, what values does `status`
   take" — which only has meaning once config has mapped the store into
   collections and items.

These are not the same operation at different scopes; they **reference the data
differently**. A raw-source inspector reaches bytes by `storage.Reference` (a
file path today, a table name or object key later). A collection inspector
reaches them by domain identity (`config.Collection` + `storage.Item.ID`),
*through* the `CollectionDefinition`. One `Corpus` type cannot serve both,
because the two layers don't address the data the same way.

This spec defines the two layers, the boundary between them, and the use of
checks as the substrate the collection layer probes through.

## Value

- **Inspectors generalize past the filesystem.** Expressing the raw layer
  against the storage abstraction (not "`*.md` under a path") means the same
  inspector model works when a `sqlite` or `mongodb` backend lands. `walk_parse`
  becomes "enumerate the store's units and report access/parse success,"
  independent of whether a unit is a file or a row.
- **The collection layer reuses the engine instead of re-deriving it.** Today
  the `object_field_*` inspectors re-walk the corpus and re-extract frontmatter,
  parallel to what the check engine already does. Making checks the substrate —
  an inspector measures the distribution a check would assert against — removes
  that duplication and keeps inspector and check semantics in lockstep.
- **It gives the cull (#25) a principled basis.** "Which inspectors are noisy?"
  becomes "which layer does each inspector belong to, and does it address the
  data correctly for that layer?" Several object-field inspectors collapse into
  one collection-layer probe rather than being culled one by one for noise.
- **Clear separation of when each runs.** Raw-source inspectors run on an
  *unconfigured* store (the onboarding case: "I have a directory, what's in
  it?"). Collection inspectors run on a *configured* project ("describe the
  shape of this collection's items"). The split makes the `inspect` command's
  scope argument meaningful instead of always "a directory."

## Current state

- **One inspector interface, one input.** All 11 inspectors take `Corpus` and
  return `Evidence`. `Corpus.Load(root)` walks `*.md`, parses each via
  `internal/frontmatter`, and stores `File{Rel, Doc, ParseErr}`. Addressing is
  always `File.Rel` (a path), and the only granularity is one-file-one-unit.
- **The storage seam exists but inspectors don't use it.** `internal/storage`
  defines `CollectionDefinition` with forward discovery (`Collections()`,
  `Items()`, `Unmatched()`) and reverse resolution (`Reference()`), plus
  `Granularity` (`FileIsItem` / `UnitIsCollection`) and the opaque
  `Reference` locator. Inspectors bypass all of it and walk the filesystem
  directly.
- **`inspect <path>` only ever means "a directory."** The command builds a
  `Corpus` from a path; it has no notion of "inspect a configured collection"
  vs "inspect a raw store," even though the spec's own scope definition was "a
  directory, a collection, or a single item."
- **Checks and the object inspectors duplicate field extraction.**
  `internal/checks` walks items and reads frontmatter to assert predicates; the
  `object_field_*` inspectors walk the same data to histogram it. There is no
  shared probe primitive.

## Design

### The two layers

**Layer 1 — raw-source inspectors.** Measure a backend store directly, before
collection configuration is meaningful. Their input is the store's own
structure as exposed by the storage layer; their addressing is the
backend-native `storage.Reference`. They answer *"what is physically in this
store, and how is it shaped at the storage level?"*

A raw-source inspector is expressed against the storage abstraction, not the
filesystem, so it generalizes across `StorageType`s. Its unit of measurement is
whatever the backend's `Granularity` makes it: a file for `FileIsItem`, a table
or row for a tabular backend. Some raw inspectors are inherently
backend-specific (filename casing only means something for a filesystem); those
declare the `StorageType`(s) they apply to and are simply absent for stores
where they don't apply.

**Layer 2 — collection inspectors.** Measure a *resolved* collection: the items
a `CollectionDefinition` maps out of a store. Their input is a collection view
(`config.Collection` plus its `[]storage.Item`); their addressing is domain
identity — `Item.ID` (collection-relative) and the collection's fields/schema —
reached *through* the definition, never through raw paths. They answer *"what is
the shape of this configured collection's items?"*

The boundary test, mirroring inspect-spec's determinism dividing line:

> **Does the measurement depend on collection configuration?**
> No → raw-source (Layer 1). Yes → collection (Layer 2).

`walk_parse` doesn't need config (it counts and parses store units) → Layer 1.
`object_field_frequency` is meaningless without "which items, which fields" →
Layer 2.

### The referencing machinery differs by layer (load-bearing)

This is the reason the two layers are distinct *types*, not one type at two
scopes. They reach the data through different machinery:

| | Raw-source (Layer 1) | Collection (Layer 2) |
|---|---|---|
| Input | the store's units (via the storage layer) | a resolved collection + its items |
| Address by | `storage.Reference` (path/table/key) | `config.Collection` + `storage.Item.ID` |
| Reached through | the backend directly / forward discovery | the `CollectionDefinition` |
| Probe primitive | parse + storage-native attributes | **checks** (see substrate below) |
| Needs config? | no | yes |
| Generalizes across backends? | the abstraction does; some inspectors are backend-specific | yes |

Because the address spaces differ, the input shape differs, and so the
interfaces differ. Concretely, two interfaces replace the single one:

```go
// Layer 1: measures a raw store, addressed by backend-native locator.
type SourceInspector interface {
    Name() string
    AppliesTo(storage.StorageType) bool   // backend-specific inspectors opt in
    Inspect(SourceView) Evidence          // SourceView wraps the storage layer
}

// Layer 2: measures a configured collection, addressed by domain identity.
type CollectionInspector interface {
    Name() string
    Inspect(CollectionView) Evidence      // CollectionView resolves items via
                                          // the CollectionDefinition + a probe API
}
```

`SourceView` and `CollectionView` are the two addressing surfaces. `SourceView`
exposes the store's units and their `Reference`s plus parse access.
`CollectionView` exposes `config.Collection`, its resolved `[]storage.Item`, and
the check-substrate probe API below — it never hands out raw paths.

`Evidence` itself is unchanged (`{inspector, scope, n, evidence}`); what gains
meaning is `scope`: a `Reference`/path for Layer 1, a collection name for
Layer 2.

### Checks as the common substrate (collection layer)

A collection inspector measures *the distribution a check would assert against*.
Rather than re-walking items and re-extracting fields, it runs a check-shaped
**probe** across the collection's items and aggregates the per-item result into
evidence instead of a pass/fail verdict.

- A check answers, per item, "is `status` one of {a,b,c}?" → violation or not.
- The dual probe answers, across items, "what is the distribution of `status`?"
  → `{read: 80, reading: 12, to-read: 50}`.

The substrate is the shared field-access and item-iteration machinery
`internal/checks` already owns. The collection layer borrows it so that:

- Inspector and check semantics stay identical (the same field extraction, the
  same type coercion, the same collection-scope iteration).
- Field-level inspectors stop duplicating the engine. The five `object_field_*`
  inspectors become **probes over one shared substrate**, not five independent
  corpus walks.

This substrate applies only to Layer 2. Layer 1 has no checks to dual against —
it predates configuration, which is where checks live — so it probes with
parse + storage-native attributes directly.

### Re-classifying the current 11

| Inspector | Layer | Notes |
|---|---|---|
| `walk_parse` | **Raw-source** | Generalize to "enumerate store units, report access/parse success." Backend-agnostic. |
| `frontmatter_shape` | **Raw-source** | Pre-configuration by definition — it clusters raw units into *candidate* collections, the thing that proposes collections. Stays deterministic-grouping only (inspect-spec's resolved question). |
| `filesystem_naming` | **Raw-source (filesystem-specific)** | `AppliesTo(Filesystem)`. Filename/path conventions only mean something for a filesystem backend. |
| `object_field_frequency` | **Collection** | Probe: per-key presence over the collection's items. |
| `object_field_types` | **Collection** | Probe: per-key type histogram. (Type-overlap with `frontmatter_shape` resolved below.) |
| `object_field_values` | **Collection** | Probe: per-key cardinality + enum set. |
| `object_field_numeric_range` | **Collection** | Probe over numeric fields. |
| `object_field_string_length` | **Collection** | Probe over string fields. |
| `markdown_heading_shape` | **Collection** | Probe over item bodies. |
| `markdown_sections` | **Collection** | Probe over item bodies; recurring-section signal. |
| `markdown_code_fences` | **Collection** | Probe over item bodies. |

Two structural consequences fall out of the re-classification, independent of
the cull:

1. **Type-overlap resolution.** `frontmatter_shape` (raw) reports a per-key type
   *set* as part of clustering; `object_field_types` (collection) reports a
   per-key type *histogram*. With the layers split, ownership is clean:
   `frontmatter_shape` keeps only the key-set fingerprint it needs to cluster;
   per-key types are owned by the collection layer, where they're a probe with
   counts. The duplication the cull analysis flagged dissolves at the layer
   boundary.
2. **Object-field consolidation is natural.** Because the five `object_field_*`
   inspectors become probes over one substrate, merging them into a single
   `object_fields` collection inspector emitting one rich record per key
   (`{present, types, cardinality, values?, numeric_range?, string_length?}`)
   is a layer-aligned refactor, not a feature change. Whether to consolidate vs.
   keep them as separate probes is left to the cull (#25).

### What this means for the cull (#25)

The cull stops being a per-inspector noise judgment and becomes two
layer-aligned decisions:

- **Layer 1:** keep the three (`walk_parse`, `frontmatter_shape`,
  `filesystem_naming`); they're the raw-store description and don't overlap.
- **Layer 2:** decide the shape of the collection-field probe(s) — one
  consolidated `object_fields` record per key vs. several narrow probes — and
  which body probes (`markdown_*`) earn their place. The low-signal outliers the
  analysis named (`object_field_string_length`, `object_field_numeric_range`,
  `markdown_code_fences`) become *fields to drop from the consolidated record*
  or *probes not to ship*, rather than whole inspectors to delete.

The cull decision (small/medium/large/consolidate) is still the maintainer's;
this spec re-frames the menu so the choice is made per layer.

### Command and registry surface

- **`inspect` gains a notion of what it's inspecting.** Against an unconfigured
  path it runs the raw-source layer (onboarding). Against a configured project /
  collection selector it runs the collection layer. Exact grammar
  (`inspect <path>` vs `inspect <collection-selector>`, or an explicit flag) is
  an open question below.
- **The registry splits by layer.** `inspect.Descriptors()` grows a `Layer`
  field (or splits into two registries) so `inspectors list` can group by layer
  the way it groups by family today, and `gendocs` renders the two layers as
  distinct sections. Registry parity (`registry_test.go`) extends to both
  interfaces.
- **`inspectors list|show`** (from #24) keeps working; it gains the layer as a
  grouping dimension alongside family. No grammar change to the noun command.

### Domain-model impact

- **New concept: inspector layer.** Add to
  [core-concepts.md](../../docs/content/deep-dives/core-concepts.md) and the
  [glossary](../../docs/content/reference/glossary.md): inspectors come in a
  raw-source layer (over a `StorageType`) and a collection layer (over a
  configured collection, probing through checks).
- **Cross-references.** `inspect-spec.md` gets a banner pointing here;
  `storage-layer-spec.md` is referenced as the foundation; `core-concepts.md`'s
  operation model notes that the descriptive operation now has two layers.

## Open questions

- **`inspect` grammar for the two layers.** Does layer selection come from the
  argument type (a raw path → Layer 1; a collection selector in a configured
  project → Layer 2), or an explicit flag? Leaning toward argument type, to keep
  the onboarding case (`inspect ./wiki`) flag-free.
- **Consolidate or keep separate object-field probes.** Resolve with the #25
  cull decision. The layering supports either; the substrate makes consolidation
  cheap.
- **`SourceView` / `CollectionView` concrete shapes.** Exact methods are a
  plan-level detail; this spec fixes only that they are *two distinct addressing
  surfaces* and that `CollectionView` exposes the check-substrate probe API, not
  raw paths.
- **How much of `internal/checks` is exported as the substrate.** The probe API
  needs the engine's field-access/iteration without its verdict logic. Whether
  that's a new exported helper in `internal/checks` or a small shared package is
  a plan-level decision.
- **Backend-specific raw inspectors and `AppliesTo`.** Confirm the opt-in shape
  (a `StorageType` predicate) vs. registering raw inspectors per `StorageType`.

## Rejected alternatives

- **Keep one `Corpus`-based interface and just add a "collection scope."** The
  two layers reference data through different machinery (`Reference` vs
  `Collection`+`ID` via the definition); a single input type would force one
  addressing scheme onto both and re-hardcode the filesystem assumption the
  storage seam exists to remove.
- **Put everything in the collection layer and treat "raw" as a collection with
  no schema.** Raw-source description must work *before* any collection exists
  (onboarding) — that's its entire point. Forcing a configured-collection shape
  onto an unconfigured store inverts the dependency.
- **Re-implement field extraction inside the collection inspectors (status
  quo).** Duplicates the check engine and lets inspector and check semantics
  drift. Checks-as-substrate keeps them identical by construction.
- **Cull inspectors first, then layer.** Culling without the layer model is the
  per-inspector noise judgment #25 started as; it would delete inspectors that
  the layering instead *reclassifies or consolidates*, throwing away signal the
  collection layer would keep.

## Test checklist (what the pending tests will assert)

Layering:
- [ ] a raw-source inspector runs against an unconfigured store and addresses by
      `Reference`, never by collection identity
- [ ] a collection inspector runs against a resolved collection and addresses by
      `Collection` + `Item.ID`, never by raw path
- [ ] `walk_parse` reports access/parse success over store units independent of
      `StorageType`
- [ ] a filesystem-specific raw inspector is absent for a non-filesystem
      `StorageType` (`AppliesTo` honored)

Substrate:
- [ ] a collection field-probe and the corresponding check extract the same
      fields with the same type coercion (no drift)
- [ ] per-key types are reported by the collection layer; `frontmatter_shape`
      carries only its clustering fingerprint (no duplicate type reporting)

Registry / command:
- [ ] every inspector declares a layer; registry parity holds for both
      interfaces
- [ ] `inspectors list` groups by layer; `gendocs` renders both layers
- [ ] `inspect` selects the raw layer for an unconfigured path and the
      collection layer for a configured selector
