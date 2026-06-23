# Storage layer spec

> **Status: planning.** Implements issue #31. Splits today's single hardcoded
> filesystem mapping into three named concepts — **StorageType**,
> **StorageInstance**, **CollectionDefinition** — and retires the umbrella term
> *connector*. Supersedes the framing in
> `docs/content/deep-dives/connectors.md`, which graduates into a reframed
> `storage.md` when this ships.

## Overview

Katalyst reaches exactly one backend today, and it does so by hardcode: a
collection is a directory, an item is a `*.md` file, an item id is the filename
stem. The mapping between a path on disk and an item identity is spread across
`internal/project` (`Items`, `ItemPath`, `Unmatched`, `ItemAt`) with no seam a
second backend could slot into.

This spec introduces the abstraction that issue #31 calls for, using the
vocabulary settled with the maintainer rather than the deep dive's single
"connector":

- **StorageType** — a known backend kind capable of holding collections and
  items: `filesystem` today; `sqlite`, `postgresql`, `mongodb` later.
- **StorageInstance** — a specific, connectable instance of a StorageType, plus
  the information needed to reach it (for `filesystem`, a root directory).
- **CollectionDefinition** — the two-way mapping from a StorageInstance's
  contents to collections and items. `FilesystemCollectionDefinition` is the
  first; one definition may yield **more than one** collection.

The "connector" of the deep dive was doing two jobs at once — *how do I reach
the store* and *how does its content map to the domain model*. Those are
StorageInstance and CollectionDefinition respectively, which also lines up with
the Great Expectations prior art (`Datasource` vs `DataConnector`) more
faithfully than the merged term did.

## Value

- A second backend (SQLite is the planned stress test) can be added without
  touching the check engine, the CRUD verbs, or selector parsing — the whole
  point of issue #31's "leave the seam."
- The mapping logic becomes testable in isolation, so the robust GX-derived
  permutation tests (flat dirs, nested layouts, multi-collection trees) have a
  home to land in.
- Users gain an explicit, declarable storage instance instead of an implicit
  "wherever `.katalyst/` is," which is the precondition for ever pointing
  Katalyst at a database or a bucket.

## Current State

The mapping is hardcoded in two packages:

- `internal/config` (`config.go`): a `Collection` is `{Name, Path, Dir,
  Pattern, Schema, Checks, Query}`, loaded one-file-per-collection from
  `.katalyst/collections/`. There is no notion of *where* the directory lives
  beyond "relative to the repo root."
- `internal/project` (`project.go`): the actual store access.
  - Forward (discovery): `Items(c)` globs `c.Dir/c.Pattern` and sets each
    item id to the filename stem (`rel[:len(rel)-len(c.Ext())]`).
  - Reverse (reconstruction): `ItemPath(c, id)` returns
    `filepath.Join(c.Dir, id+c.Ext())` — the `item add notes/dune → notes/dune.md`
    path. This reverse mapping already exists; it is the degenerate, stem-only
    case of the bidirectional mapping the deep dive describes.
  - `Unmatched(c)` walks `c.Dir` and reports files that fail `c.Pattern`.
  - `ItemAt`, `Resolve` (`selector.go`) sit on top.

Nothing is wrong with the *behavior*; the problem is that all four functions
assume `os` + `filepath` + a flat directory, with the layout knowledge inlined.
There is no interface to implement for a non-filesystem store, and a directory
maps to exactly one collection.

The conceptual framing exists in `docs/content/deep-dives/connectors.md`
(the two-way contract, the granularity principle, the GX lineage, the
configured/inferred axis), and the glossary carries a `Connector` row marked
"(Future)."

## Design

### The three concepts

```
StorageType        filesystem | sqlite | postgresql | mongodb   (kind)
   │
StorageInstance    one configured store of a type + how to reach it
   │                (filesystem: a root directory)
CollectionDefinition  reads ONE StorageInstance → emits N collections,
                      and maps path ⇄ (collection, item-coordinates)
```

`StorageType` is a closed registry in code (an enum plus a constructor table),
mirroring how `internal/checks/registry.go` enumerates check types. Only
`filesystem` is registered now; the registry is the extension point.

`StorageInstance` carries the connection detail for one store. For
`filesystem`, that is a `Root` directory (absolute, symlink-resolved, exactly
like `config.Config.Root` is today). For a future SQL type it would carry a
DSN. Instances are **named** and declared (see config below); an undeclared
default instance preserves today's behavior.

`CollectionDefinition` is the **narrow Go seam** — the interface a backend
implements. It owns the two-way mapping and declares its granularity. The
filesystem implementation wraps exactly today's `project` logic.

### The seam (Go interface)

A new package `internal/storage` holds the types above; `internal/project`
becomes a consumer of the interface instead of an implementer of the mapping.

```go
// internal/storage

// Granularity is the level at which a store's matched units attach to the
// domain model. It is a property of the StorageType, not user config.
type Granularity int
const (
    FileIsItem       Granularity = iota // markdown filesystem: file=item, dir-group=collection
    UnitIsCollection                    // tabular: table/file=collection, row=item
)

// Reference is a backend-native locator: a file path today, a table name or
// S3 key later. Kept opaque so non-filesystem stores need not be paths.
type Reference string

// CollectionDefinition is the two-way mapping between one StorageInstance and
// the collection/item domain model. Forward = discovery; reverse =
// reconstruction. Both directions are mandatory (see the deep dive).
type CollectionDefinition interface {
    Granularity() Granularity

    // Forward (discovery).
    Collections() []config.Collection
    Items(config.Collection) ([]project.Item, error)
    Unmatched(config.Collection) ([]Reference, error)

    // Reverse (reconstruction): item identity → backend locator.
    Reference(c config.Collection, id string) (Reference, error)
}
```

`FilesystemCollectionDefinition` is constructed from an instance's config (its
`root` plus the `collections:` block) and implements every method by lifting the
current `internal/project` bodies verbatim — this spec is a refactor behind a
new boundary, not a behavior change.

**Packaging note (internal only).** `project.Item` is the struct
`{Collection, ID, Path}` defined in `internal/project/project.go`. Because the
new interface lives in `internal/storage` and its methods return items,
`internal/project` would import `internal/storage` while `internal/storage`
would need `Item` — an import cycle. The fix is to relocate the `Item` type
definition down into `internal/storage` (with `project` re-exporting or wrapping
it if convenient). This is pure code organization with no user-visible effect;
it is called out only so the implementer expects it.

### Two-way mapping: keep stem-only now, port the GX template later

Today an item is identified by **one coordinate**, its stem, and reverse
resolution is `Join(dir, id+ext)`. The deep dive's richer story —
multi-coordinate layouts like `notes/2020/dune.md` parsed into `{year, slug}`,
and the inverse path reconstruction — is **defined here but not built here**.

When it lands, the mechanism is the GX-recovered logic the maintainer attached
(`recovered_data_connector/.../util.py`), with one correction GX itself flagged:
prefer an **inherently two-way template** (`{slug}_{year}.md`) over inverting an
arbitrary regex (`_invert_regex_to_data_reference_template`, annotated *"almost
certainly still brittle"*). The template is bidirectional by construction; the
regex is not. The pattern must also **own the file extension**, or
reconstruction is ambiguous when several extensions are allowed (GX's `util.py`
TODO). Its 82 permutation tests are the test contract to port alongside.

Keeping the seam two-way from day one (it already is — `Items` + `Reference`)
means this generalization slots in without re-opening the interface.

### Granularity

Issue #31 asks how a definition reports whether a store unit becomes a
Collection or an Item. Answer: it is a **property of the StorageType**, surfaced
via `Granularity()`, not a user-set knob. `filesystem` markdown is
`FileIsItem`; a tabular backend would be `UnitIsCollection`. This is the
"Item and Collection are roles, not file counts" principle from the deep dive,
made concrete.

### Config surface

A **StorageInstance declares its own collections**, the way a Great
Expectations `Datasource` embeds its `DataConnector` and assets. The instance
config carries the connection detail (`type`, `root`) plus a `collections:`
block — and that block *is* the `FilesystemCollectionDefinition`, mapping the
instance's contents onto one or more named collections. There is no separate
top-level `collections/` kind: a collection is always declared inside the
instance whose store holds it. Schemas stay their own kind (a schema by name
can back collections across instances).

Two equivalent forms, chosen by the same `discovery` machinery
`internal/config` already applies to schemas and collections:

**General case — one file per instance** (`discovery: convention`, the default):

```yaml
# .katalyst/storage/local.yaml — instance name "local" is the filename stem.
type: filesystem
root: .                      # directory, relative to the repo root
collections:
  books:
    path: notes/books
    pattern: "*.md"          # optional; default "*.md"
    schema: book
    checks:
      - kind: markdown_title_matches_h1
  notes:
    path: notes
    schema: note
```

**Very small projects — inline in `config.yaml`** (`discovery: explicit`):

```yaml
# .katalyst/config.yaml
storage:
  discovery: explicit
  defs:
    local:
      type: filesystem
      root: .
      collections:
        notes: { path: notes, schema: note }
```

Both are supported; pick by project size (the maintainer's call: inline for the
very small, a file-per-instance directory for the general case). This is exactly
the convention-vs-explicit choice schemas and collections already offer
(`internal/config/README.md`).

**No implicit instances.** Katalyst never synthesizes a storage instance at
runtime. Instead, `katalyst init` writes a default one — `storage/local.yaml`
with `type: filesystem`, `root: .` — reproducing today's behavior *explicitly*,
on disk and reviewable. A selector that resolves into a missing instance is a
usage error, never a silent default.

**One definition → many collections** is the `collections:` block above: one
`FilesystemCollectionDefinition` (one instance, one tree) yielding several
collections. A directory holding files for two collections is two entries under
`collections:` sharing the instance's `root` with different `path`/`pattern`. A
single file mapping into *more than one* collection stays out of scope —
invariant #4 (a file belongs to exactly one collection) is retained, per the
maintainer's deferral.

### What this spec builds vs. defers

Builds now:

- `internal/storage` package: `StorageType` registry, `StorageInstance`,
  `CollectionDefinition` interface, `Granularity`, `Reference`.
- `FilesystemCollectionDefinition` wrapping today's `Items`/`ItemPath`/
  `Unmatched`/`ItemAt` logic, behavior-preserving, built from an instance's
  `collections:` block.
- `internal/project` refactored to drive the interface; selectors, CRUD verbs,
  and the check engine untouched.
- `storage` as a config kind that **embeds** its collections (convention
  `storage/` directory + explicit `config.yaml` form), replacing the standalone
  `collections/` kind.
- `katalyst init` writing a default `local` filesystem instance.
- The `connector` → triad doc/vocabulary migration.

This **replaces the standalone `.katalyst/collections/` kind**: existing
collection files must move into a storage instance's `collections:` block. Given
the project's early-days status (the docs already carry an "early days" warning),
this is an accepted breaking config change, called out in the configuration
reference rather than papered over with a compatibility shim.

Defers (seam left open, not implemented):

- Multi-coordinate templates and the GX two-way port (above).
- **Inferred** mode — collection names *discovered* from the store rather than
  declared — maps to the future `infer`/`profile` path, not `check`.
- A `doctor`/`explain` command (GX's `self_check`: "here are your collections,
  some examples, and what matched nothing").
- Any non-filesystem StorageType.
- Multiple CollectionDefinitions per instance (GX allowed several
  `DataConnector`s per `Datasource`); one mapping per instance is enough now.
- A file mapping into more than one collection — invariant #4 retained.

## Open Questions

1. **Per-collection reviewability when one instance holds many collections.**

   **Context.** The decided model embeds every collection in its instance's
   `collections:` block. That diverges from a decision the project made
   deliberately and documented in `internal/config/README.md`: collections were
   split into one reviewable file each precisely so a change to one is a small,
   isolated diff. A filesystem instance with twenty collections is now one large
   file — every collection edit touches it, and review loses the per-collection
   locality the directory layout bought. (GX accepted this: a `Datasource`'s
   connectors and assets all live in one block.)

   **Choices & tradeoffs.**
   - **(A) Inline-only.** Collections live only in the instance config. *Buys:*
     one model, closest to GX and to the decision just made; the instance file is
     the single source of its mapping. *Costs:* large single file for large
     instances; reintroduces the very "one big file" the project moved away from.
   - **(B) Inline + optional per-collection files (recommended).** Default is
     inline (A). As an escape hatch, an instance may point a collection at its
     own file (e.g. `collections: { books: { $file: books.yaml } }`, or an
     instance-scoped `storage/local/` directory), restoring one-reviewable-file
     diffs for projects that outgrow inline. *Buys:* small projects stay simple,
     large projects keep locality. *Costs:* a second authoring path to document
     and load; mild "two ways to do it."
   - **(C) Defer.** Ship inline-only now; add the escape hatch only if a real
     project feels the pain. *Buys:* least to build. *Costs:* the large-file
     regression ships first and may bite dogfooding (katalyst's own docs config).

   **Recommendation:** (B) as the eventual shape, but (C) is a reasonable
   *sequencing* — ship inline, hold the per-file hatch until needed — since the
   seam doesn't change either way. This is the one place the decided model
   trades against an established project value, so it's flagged for an explicit
   call rather than assumed.

_All other questions from the prior draft are resolved and folded into Design:
the config is definition-centric with collections embedded in their instance
(OQ1 → "Config surface"); storage supports both a `storage/` directory and an
inline `config.yaml` form (OQ2 → "Config surface"); the file-in-many-collections
case is deferred with invariant #4 retained (OQ3 → "Config surface"); and the
`Item` package relocation is an internal detail (OQ4 → the packaging note under
"The seam")._

## Documentation updates

**User docs (Hugo, `docs/content/`):**

- `deep-dives/connectors.md` → **rename to `storage.md`**, retitle "Storage
  layer," and reframe the body around StorageType / StorageInstance /
  CollectionDefinition. Keep the GX lineage, granularity principle,
  configured/inferred axis, unmatched-is-first-class, and the "do better than
  GX" lessons — they all survive; only the umbrella noun changes. This page is
  evergreen (not retired with the spec).
- `deep-dives/_index.md` (line 15) and `_index.md` (line 41): replace
  "connectors" in the chapter listing with "storage."
- `reference/glossary.md`: remove the `Connector` row; add rows for
  **StorageType**, **StorageInstance**, **CollectionDefinition**, and
  **Granularity** (and keep **Coordinates**/**Data reference** if the reframed
  deep dive retains them).
- `reference/configuration.md`: document the `storage/` kind with embedded
  `collections:`, the `type`/`root` keys, the convention vs. inline-`config.yaml`
  forms, that `init` writes a default `local` instance, and that the standalone
  `collections/` kind is **replaced** (a breaking change for existing configs).
- `getting-started.md`: the walkthrough that writes `.katalyst/collections/`
  moves to declaring collections inside `storage/local.yaml`.
- `deep-dives/domain-model.md` (lines 169–170): update the Selector note's
  `connectors.md` relref to `storage.md` and the "connector coordinates"
  wording; revisit the "A collection owns its checks" invariant text now that
  collections are declared inside an instance.
- `contributing/how-we-document.md` (line 28) and `how-we-plan.md` (line 79):
  swap "connectors" for "storage" in the evergreen-deep-dive references.

**Developer docs:**

- `internal/storage/doc.go`: new package doc — the three concepts, the two-way
  contract, granularity, and the GX provenance/correction.
- `internal/config/README.md`: document the `storage/` kind with embedded
  collections, the two authoring forms, and that `init` writes the default
  instance (no runtime synthesis); update the "Why named collections" section,
  which currently assumes a standalone `collections/` kind.
- `internal/project/` package doc: update to say it now consumes the
  `internal/storage` seam rather than implementing the mapping.
- `AGENTS.md`: record the convention "path ⇄ item-identity translation passes
  through `internal/storage.CollectionDefinition`; do not inline filesystem
  assumptions elsewhere."

**Specs (cross-references, not user docs):** `product/specs/cli-spec.md`
(both its `connectors.md` relrefs *and* its `katalyst init` description, which
says init creates empty `schemas/` and `collections/` dirs — now it writes a
default storage instance), `product/specs/dogfood-docs-spec.md`, and
`product/v0-implementation-plan.md` mention `connectors.md`/`collections/`;
update those when this ships.

## Appendix: GX → Katalyst mapping (carried from the deep dive)

| GX (legacy V3) | Katalyst |
|----------------|----------|
| Datasource | **StorageInstance** (+ its StorageType) |
| DataConnector | **CollectionDefinition** |
| DataAsset (`data_asset_name`) | **Collection** |
| Batch / BatchDefinition | **Item** (markdown) / Collection (tabular) — per granularity |
| PartitionDefinition (`group_names` → values) | item **coordinates** (today: the stem) |
| BatchRequest / PartitionQuery | a **selector** |
| BatchSpec | the resolved `Reference` (file path) |
| Configured vs. Inferred | `check` (declared) vs. `infer`/`profile` (discovered) |
| `get_unmatched_data_references` | `Unmatched()` → `check` errors |
| `self_check` | a future `doctor`/`explain` |

Lessons carried verbatim from GX's own TODOs: prefer a two-way **template** over
regex inversion; the **pattern owns the extension**; keep **collection identity
separate from coordinates** (GX leaked `data_asset_name` into the coordinate map
and regretted it — see `util.py` line 116).
