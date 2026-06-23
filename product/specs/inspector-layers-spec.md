# Inspector layers — raw-source and collection inspectors over the storage seam

> **Status: planning.** Reorganizes inspectors into two layers with **distinct
> data-referencing machinery**, and collapses today's 11 inspectors into **a few
> reusable measurement primitives** applied across those layers. A
> **raw-source** layer measures a backend store directly (a filesystem, a file
> database, …), addressed by backend-native locator; a **collection** layer
> measures *configured* collections, addressed by domain identity and probing
> items through **checks as a common substrate**. Builds on the storage seam
> from [storage-layer-spec.md](./storage-layer-spec.md) (#31) and supersedes the
> single-scope inspector model in [inspect-spec.md](./inspect-spec.md). Sets the
> direction the inspector cull (#25) executes against. Plan: TBD.

## Overview

[inspect-spec.md](./inspect-spec.md) introduced inspectors as the descriptive
dual of checks: a check asserts a predicate, an inspector reports the
distribution that predicate would be tested against. It shipped 11 inspectors,
each a pure function of a single `inspect.Corpus` — a flat walk of `*.md` files
under a path:

```go
type Inspector interface {
    Name() string
    Inspect(Corpus) Evidence
}
```

That model has one input shape and one addressing scheme (a file's path relative
to the scope root). It predates the storage layer, and its 11 inspectors are
narrower than they need to be: five of them (`object_field_*`) are columns of
one table, three (`markdown_*`) are facets of one body walk, and two
(`walk_parse`, `filesystem_naming`) are the same tree walk at two depths.

Now that #31 has landed a real seam between a backend store and the domain model
(`StorageType` / `StorageInstance` / `CollectionDefinition`), inspectors can be
reorganized along two axes at once:

1. **By layer** — does the measurement depend on collection configuration?
   *Raw-source* inspectors describe a store before any config (the onboarding
   case); *collection* inspectors describe configured collections. The two
   **reference data through different machinery** (a backend `Reference` vs. a
   `Collection` + `Item.ID` resolved through the definition), so they are
   distinct types, not one type at two scopes.
2. **By primitive** — most inspectors reduce to one of three reusable
   measurement engines (`object_fields`, `markdown_body`, file-metadata),
   pointed at different inputs. Consolidating around the primitives is what
   turns 11 inspectors into ~5.

This spec defines the layers, the primitives, the inspectors built from them,
and the shared summarizer that keeps per-directory output small.

## Value

- **Far fewer moving parts.** 11 inspectors → 5, over 3 reused primitives. The
  five `object_field_*` become columns of one data dictionary; the three
  `markdown_*` become facets of one body inspector; the two filesystem walkers
  become one shallow + one deep variant.
- **Inspectors generalize past the filesystem.** Expressing the raw layer
  against the storage abstraction (not "`*.md` under a path") means the same
  model works when `sqlite`/`mongodb` land.
- **The collection layer reuses the engine.** Making checks the substrate — an
  inspector measures the distribution a check would assert against — removes the
  field-extraction the `object_field_*` inspectors duplicate today and keeps
  inspector and check semantics in lockstep.
- **Candidate-collection discovery gets stronger.** `document_shape` clusters
  files on a *composite* fingerprint (frontmatter keys + body structure + file
  metadata), a far better "these belong together" signal than frontmatter alone.
- **A principled cull (#25).** "Which inspectors are noisy?" becomes "which
  primitive owns this measurement, and is it a column worth keeping?" Low-signal
  measurements become *columns to drop*, not whole inspectors to delete.

## Current state

- **One interface, one input.** All 11 inspectors take `Corpus`, return
  `Evidence`. `Corpus.Load(root)` walks `*.md`, parses each via
  `internal/frontmatter`, stores `File{Rel, Doc, ParseErr}`. Addressing is
  always `File.Rel`.
- **The storage seam exists but inspectors ignore it.** `internal/storage`
  defines `CollectionDefinition` (`Collections`, `Items`, `Unmatched`,
  `Reference`), `Granularity` (`FileIsItem`/`UnitIsCollection`), and the opaque
  `Reference`. Inspectors walk the filesystem directly instead.
- **Heavy internal duplication.** The five `object_field_*` inspectors each
  re-walk the corpus to histogram one aspect of the same fields;
  `frontmatter_shape` re-extracts the same key-sets; `walk_parse` and
  `filesystem_naming` walk the same tree. Checks walk items to assert
  predicates over the same fields, with no shared probe.
- **`inspect <path>` only ever means "a directory."** No notion of "inspect a
  configured collection" vs. "inspect a raw store," and inspectors are
  parameterless by an explicit v1 decision in inspect-spec.

## Design

### The two layers and their referencing machinery (load-bearing)

The layers are distinct **types** because they reach data through different
machinery — not one input type at two scopes:

| | Raw-source (Layer 1) | Collection (Layer 2) |
|---|---|---|
| Input | the store's units (via the storage layer) | a resolved collection + its items |
| Address by | `storage.Reference` (path/table/key) | `config.Collection` + `storage.Item.ID` |
| Reached through | the backend directly / forward discovery | the `CollectionDefinition` |
| Probe primitive | parse + storage-native attributes | **checks** (shared substrate) |
| Needs config? | no | yes |
| When it runs | onboarding: "what's in this store?" | "describe this collection's items" |

The boundary test:

> **Does the measurement depend on collection configuration?**
> No → raw-source. Yes → collection.

Two interfaces replace the single `Inspector`:

```go
// Layer 1: measures a raw store, addressed by backend-native locator.
type SourceInspector interface {
    Name() string
    AppliesTo(storage.StorageType) bool   // backend-specific inspectors opt in
    Inspect(SourceView, Params) Evidence  // SourceView wraps the storage layer
}

// Layer 2: measures a configured collection, addressed by domain identity.
type CollectionInspector interface {
    Name() string
    Inspect(CollectionView, Params) Evidence  // resolves items via the
                                              // CollectionDefinition + probe API
}
```

`SourceView` exposes the store's units and their `Reference`s plus parse access;
`CollectionView` exposes `config.Collection`, its resolved `[]storage.Item`, and
the check-substrate probe API — it never hands out raw paths. `Params` is the
new descriptor-parameter channel (see *Parameters* below). `Evidence` is
unchanged (`{inspector, scope, n, evidence}`); `scope` gains meaning — a
`Reference`/path for Layer 1, a collection name for Layer 2.

### Measurement primitives (the reuse layer)

Three layer-agnostic engines do the actual measuring. Inspectors are thin
wrappers that point a primitive at an input.

- **`object_fields`** — a **data dictionary** over a set of objects (maps). Per
  field, it reports: presence/frequency over `n`, observed type histogram, value
  cardinality, and the most common values (the enum signal, with counts). The
  five `object_field_*` inspectors are exactly the columns of this one table.
- **`markdown_body`** — a structure profile over a set of markdown bodies, with
  facets for heading shape, recurring sections, and code fences. The three
  `markdown_*` inspectors are facets of this one walk.
- **file-metadata** — path-level attributes over a set of references: type /
  extension, naming convention (kebab/snake/other), depth, counts. No file is
  opened.

Because they're layer-agnostic, the same primitive serves both layers: a
collection inspector runs `object_fields` over a collection's items, while the
raw layer runs the *same* `object_fields` over the frontmatter of loose markdown
files in a directory.

### Dictionary vs. clustering (why both, and how they compose)

A data dictionary reports per-field **marginals** (columns); clustering groups
**files into populations** (rows). They answer different questions and neither
substitutes for the other. For `notes/`:

```
a.md {title, author, rating}      Dictionary (marginal):     Clustering (populations):
b.md {title, author, rating}        title  5/5 string          P1 ×3 {title,author,rating}
c.md {title, author, rating}        author 3/5 string          P2 ×2 {title,date,tags}
d.md {title, date, tags}            rating 3/5 integer
e.md {title, date, tags}            date   2/5 date
                                    tags   2/5 array
```

The dictionary alone misleads: `author 3/5` reads as "optional," when it is
**required in P1, absent in P2**. So the two compose — **cluster to find the
populations, then run the dictionary inside each cluster**: `author` becomes
`3/3 in P1, 0/2 in P2`, the actionable form. Clustering is `document_shape`'s
job; the dictionary is `object_fields`'.

### Raw-source inspectors

- **`file_tree`** (`FileTreeInspector`) — **shallow, cheap.** Walks the store
  tree and reports a **per-directory** file-metadata profile: file types /
  extensions, naming conventions, depth, counts. Opens nothing. Filesystem-
  specific (`AppliesTo(Filesystem)`); a tabular backend supplies its own.
- **`file_tree_content`** (`FileTreeDeepContentInspector`) — **deep,
  expensive.** Same walk, but opens and parses files of *known-parseable* types
  (markdown today): per-directory parse success, frontmatter presence, body
  presence, and `object_fields` / `markdown_body` summaries of what's inside.
  The agent runs `file_tree` first, this only where it matters.
- **`document_shape`** — clusters files into candidate collections on a
  **composite fingerprint** assembled from the primitives:

  ```
  fingerprint(file) = {
    frontmatter: key-set            (object_fields dimension)
    body:        section skeleton   (markdown_body dimension)
    file:        type + naming      (file-metadata dimension)
  }
  ```

  Files with a matching fingerprint form a class — a candidate collection that
  agrees on metadata schema **and** body structure **and** file convention.
  This is the renamed, widened `frontmatter_shape`: same deterministic-grouping
  role (drawing/naming the final boundary stays the agent's judgment, per
  inspect-spec), now on a richer signal. Clustering identity is the composite
  fingerprint; the per-cluster dictionary ships as adjacent evidence.

### Collection inspectors

- **`object_fields`** over a configured collection's items — the data
  dictionary, addressed by domain identity, probing through the check substrate.
- **`markdown_body`** over a configured collection's items — the body structure
  profile, same addressing and substrate.

### Checks as the common substrate (collection layer)

A collection inspector measures *the distribution a check would assert against*.
Instead of re-walking items and re-extracting fields, it runs a check-shaped
**probe** across the collection's items and aggregates the per-item result into
evidence rather than a verdict. The substrate is the field-access and
item-iteration machinery `internal/checks` already owns; borrowing it keeps
inspector and check semantics identical and removes the duplication above. Layer
1 has no checks to dual against (it predates configuration), so it probes with
parse + storage-native attributes directly.

### Per-directory profiles, kept small (the shared summarizer)

`file_tree*` and `document_shape` both emit per-directory / per-cluster output,
which would explode on a large tree. They share one mechanism, which is the same
clustering idea applied to directories:

> **Output is proportional to the number of distinct profiles, not the number of
> directories.**

1. Compute a profile per directory (or per file, for `document_shape`).
2. **Dedupe identical profiles into named classes** (P1, P2, …).
3. Render the tree as `directory → class`, plus the (small) class definitions,
   plus a short **outlier/diff list** for directories that differ from their
   siblings or parent. Each directory may be rendered as a **delta from its
   parent**, so depth doesn't multiply output either.

A 200-directory wiki where 190 dirs match one profile collapses to "190 = P1,
7 = P2, 3 outliers (here's how each differs)." The symmetry is deliberate:
clustering **files** → candidate **collections**; clustering **directories** →
candidate **storage layout**; one "classes + outliers" renderer serves both.

### Parameters (first inspector parameter)

inspect-spec resolved "initial inspectors are parameterless in v1." This spec
supersedes that: the summarizer needs a tunable **collapse tolerance** — how
similar two profiles (directory profiles, or file fingerprints) must be to share
a class. Higher tolerance merges more into fewer classes (smaller output, big
buckets + outliers); lower tolerance keeps finer distinctions. It is passed to
the inspector via the `Params` channel and surfaced as a command flag (e.g.
`--tolerance`/`--collapse`). Exact name and scale are plan-level; the design
commitment is that inspectors now take parameters and this is the first.

### Re-classification: 11 → 5

| Today (11) | Becomes | Layer |
|---|---|---|
| `walk_parse` | `file_tree_content` (deep variant) | Raw-source |
| `filesystem_naming` | `file_tree` (shallow variant) | Raw-source |
| `frontmatter_shape` | `document_shape` (composite fingerprint) | Raw-source |
| `object_field_frequency` | column of `object_fields` | Collection (primitive reused raw) |
| `object_field_types` | column of `object_fields` | Collection |
| `object_field_values` | column of `object_fields` | Collection |
| `object_field_numeric_range` | column of `object_fields` (optional) | Collection |
| `object_field_string_length` | column of `object_fields` (optional) | Collection |
| `markdown_heading_shape` | facet of `markdown_body` | Collection |
| `markdown_sections` | facet of `markdown_body` | Collection |
| `markdown_code_fences` | facet of `markdown_body` (optional) | Collection |

The `frontmatter_shape` / `object_field_types` type-overlap dissolves: per-key
types are an `object_fields` column; `document_shape` carries only the
fingerprint it clusters on.

### The cull (#25) as a consequence

With the primitives in place, the cull is no longer "delete inspectors" but
"choose columns/facets":

- **`object_fields` columns** — keep presence, types, cardinality, common
  values; treat `numeric_range` and `string_length` as optional columns to drop
  (they were the noisiest standalone inspectors).
- **`markdown_body` facets** — keep heading shape and recurring sections; treat
  code fences as an optional facet.
- **Raw-source** — keep all three; they don't overlap once split by depth and
  role.

The maintainer still picks how lean to go; the model makes each choice a column,
not an amputation.

### Command and registry surface

- **`inspect` gains a layer notion.** An unconfigured path runs the raw-source
  layer (onboarding); a configured collection selector runs the collection
  layer. Grammar (argument type vs. explicit flag) is an open question below.
- **Registry grows a `Layer` (and primitive) dimension.** `inspectors list`
  groups by layer alongside family; `gendocs` renders the layers as distinct
  sections; registry parity (`registry_test.go`) extends to both interfaces.
  The `inspectors list|show` noun command from #24 keeps working unchanged.
- **Parameters** surface as flags on `inspect` and are documented per inspector.

### Domain-model impact

- **New concepts: inspector layer and measurement primitive.** Add to
  [core-concepts.md](../../docs/content/deep-dives/core-concepts.md) and the
  [glossary](../../docs/content/reference/glossary.md): inspectors come in a
  raw-source layer (over a `StorageType`) and a collection layer (over a
  configured collection, probing through checks), built from a small set of
  reusable primitives.
- **Cross-references.** `inspect-spec.md` carries a supersession banner pointing
  here; `storage-layer-spec.md` is the foundation; `core-concepts.md`'s
  operation model notes the descriptive operation now has two layers.

## Open questions

- **`inspect` grammar for layer selection** — argument type (raw path → Layer 1;
  collection selector in a configured project → Layer 2) vs. an explicit flag.
  Leaning argument-type to keep onboarding (`inspect ./wiki`) flag-free.
- **Tolerance parameter shape** — name, scale (0–1 proportion? discrete
  levels?), and whether `file_tree*` and `document_shape` share one knob or take
  separate ones.
- **`SourceView` / `CollectionView` concrete shapes** — plan-level; the spec
  fixes only that they are two distinct addressing surfaces and that
  `CollectionView` exposes the probe API, not raw paths.
- **How much of `internal/checks` is exported as the substrate** — a new
  exported helper vs. a small shared package.
- **`object_fields` over nested/array values** — how deep the dictionary
  characterizes non-scalar fields (today: scalars only enter the value set).

## Rejected alternatives

- **Keep one `Corpus`-based interface with a "collection scope" flag.** The two
  layers reference data through different machinery; one input type forces one
  addressing scheme onto both and re-hardcodes the filesystem assumption the
  storage seam exists to remove.
- **Keep the 11 inspectors, just re-label them by layer.** Misses that five are
  columns of one table and three are facets of one walk; the duplication and the
  cull churn both remain. The primitives are the point.
- **Put everything in the collection layer ("raw is a schemaless collection").**
  Raw-source description must work *before* any collection exists; forcing a
  configured shape onto an unconfigured store inverts the dependency.
- **Re-implement field extraction inside collection inspectors (status quo).**
  Duplicates the check engine and lets semantics drift; checks-as-substrate
  keeps them identical by construction.
- **One unified file-tree inspector instead of shallow + deep.** They share a
  walk but differ by an order of magnitude in cost (paths vs. open-and-parse);
  splitting lets an agent scan cheaply everywhere and parse only where needed.
- **Cull first, then layer.** Culling without the model deletes inspectors the
  primitives instead *reclassify or fold into columns*, discarding signal.

## Test checklist (what the pending tests will assert)

Layers & addressing:
- [ ] a raw-source inspector runs on an unconfigured store, addressed by
      `Reference`, never by collection identity
- [ ] a collection inspector runs on a resolved collection, addressed by
      `Collection` + `Item.ID`, never by raw path
- [ ] a filesystem-specific raw inspector is absent for a non-filesystem
      `StorageType` (`AppliesTo` honored)

Primitives:
- [ ] `object_fields` reports presence, type histogram, cardinality, and common
      values; the five old `object_field_*` map onto its columns
- [ ] running `object_fields` inside `document_shape` clusters yields per-cluster
      marginals (e.g. a field 100% in one class, 0% in another)
- [ ] `markdown_body` reports heading shape, recurring sections, code fences as
      facets of one walk
- [ ] the same `object_fields` primitive runs over collection items (Layer 2)
      and over loose-file frontmatter (Layer 1)

Substrate:
- [ ] a collection field-probe and the corresponding check extract the same
      fields with the same type coercion (no drift)
- [ ] per-key types are reported by `object_fields`; `document_shape` carries
      only its clustering fingerprint (no duplicate type reporting)

Tree walk, shape & summarizer:
- [ ] `file_tree` opens no files; `file_tree_content` parses known types only
- [ ] `document_shape` clusters files on the composite fingerprint
      (frontmatter + body + file dimensions), not frontmatter alone
- [ ] per-directory output dedupes identical profiles into classes and lists
      only outliers/diffs
- [ ] raising the tolerance parameter reduces the number of classes (smaller
      output); lowering it increases them

Registry / command:
- [ ] every inspector declares a layer; registry parity holds for both
      interfaces
- [ ] `inspectors list` groups by layer; `gendocs` renders both layers
- [ ] `inspect` selects the raw layer for an unconfigured path and the
      collection layer for a configured selector
- [ ] a tolerance flag reaches the inspector via `Params`
