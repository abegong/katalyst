+++
title = "Storage layer"
weight = 40
+++

# Storage layer

> **Status: partly shipped.** The seam and the config model exist
> (`internal/storage`, storage instances under `.katalyst/storage/`); the
> richer mapping (multi-coordinate templates, inferred mode, non-filesystem
> backends) is still ahead. This page describes the whole arc, and notes what
> is built versus planned.

## What the storage layer is

The **storage layer** is the two-way mapping between a backend store and the
Katalyst domain model. It answers: *what collections and items does this store
contain, and where does each one live?*, in both directions. It is Katalyst's
realization of the general **storage** concept from
[core concepts]({{< relref "core-concepts.md" >}}): the filesystem is one
backend; SQLite, directories of CSVs, S3 buckets, and hosted APIs are others.
The first real stress test will be **SQLite**, because it is the first backend
that forces the granularity question below.

## Three concepts

The layer is three named pieces, not one. Earlier drafts called the whole thing
a *connector*; that single word was doing two jobs, *how do I reach the store*
and *how does its content map to the model*, so it was split:

| Concept | Meaning |
|---|---|
| **StorageType** | A known backend kind capable of holding collections and items: `filesystem` today; `sqlite`, `postgresql`, `mongodb` later. |
| **StorageInstance** | A specific, connectable instance of a StorageType, plus the information needed to reach it (for `filesystem`, a root directory). |
| **CollectionDefinition** | The two-way mapping from a StorageInstance's contents to collections and items. `FilesystemCollectionDefinition` is the first; one definition may yield **more than one** collection. |

In config, a StorageInstance declares the collections it maps, the instance
file *is* where the CollectionDefinition lives (see
[Configuration]({{< relref "../reference/configuration.md" >}})). In code, the
seam is `internal/storage/collection.CollectionDefinition`; `internal/project` consumes it
rather than implementing the filesystem mapping inline.

Storage readers use codecs to decode a matched unit's content into the shape
checks and inspectors consume. The markdown filesystem reader uses
`internal/codec/markdownbodytext` for frontmatter/body parsing; codecs are
shared content adapters, not storage backends.

## Lineage: GX legacy DataConnectors

The design is adapted from Great Expectations' V3 `DataConnector` layer
(recovered for reference *outside this repo*; originally GX commit
`6cd804579`, removed in `27eb8d28b`). A GX `DataConnector` defined a
`regex + group_names` naming convention that mapped each file/key in a store
to a `BatchDefinition`, plus the inverse mapping back to a path. GX's
`Datasource` (the store) versus `DataConnector` (the mapping) split is exactly
the StorageInstance versus CollectionDefinition split here.

### The heart: a two-way mapping

- **Forward (discovery):** `path → match pattern → captured groups` become
  the unit's *coordinates*.
- **Reverse (reconstruction):** `coordinates → fill a template → path`.

The reverse direction is **not optional**. Katalyst needs it the moment
`item add notes/dune` has to decide *what file to create*, that is the same
path-reconstruction problem. Today it is the degenerate, stem-only case
(`Reference(c, id) → <dir>/<id>.md`); it grows with the layout.

## Concept mapping: GX → Katalyst

| GX (legacy V3) | Katalyst |
|----------------|----------|
| Datasource | **StorageInstance** (+ its StorageType) |
| **DataConnector** | **CollectionDefinition** |
| DataAsset (`data_asset_name`) | **Collection** |
| **Batch / BatchDefinition** | **Item** *(markdown)* / Collection *(tabular)*, see granularity |
| PartitionDefinition (`group_names` → values) | the item's **coordinates** (today: the stem) |
| BatchRequest / PartitionQuery | a **selector** (the [addressing] grammar) |
| BatchSpec | the resolved fetch instruction (a `Reference`, the file path) |
| Configured vs. Inferred | `check` (declared) vs. `infer` / `profile` (discovered) |

### The granularity principle (locked)

**"What does one matched store unit become?" has no global answer, it is a
property each StorageType declares for its backend.**

- **Markdown filesystem:** one file = one **Item**; a directory of files =
  a **Collection** (`Granularity` is `FileIsItem`).
- **Tabular (CSV / SQL):** one file/table = one **Collection**; its rows =
  **Items** (`UnitIsCollection`).

This is why a GX *Batch* maps to a Katalyst *Item* in the markdown world but
to a *Collection* in the tabular world, and both are correct. The definition
absorbs that impedance: alongside the path↔coordinates mapping, it declares the
**level** at which a store's units attach to the collection/item hierarchy.

Implication: **Item and Collection are roles, not file counts.** A backend that
packs many items into one physical unit (rows in a table) and one that spreads a
single item across a whole unit (a markdown file) are both valid.

## Two modes: Configured vs Inferred

GX shipped both, and they map cleanly onto Katalyst verbs:

- **Configured:** collections and their patterns are declared explicitly (the
  instance's `collections:` block). This is the `check` path: known structure,
  enforced. *Shipped.*
- **Inferred:** collection names and structure are *discovered* by applying
  the pattern to whatever is in the store. This is the `infer` / `profile`
  path: structure read out of the data. *Planned.*

## Unmatched references are first-class

GX tracked files that matched no pattern (`get_unmatched_data_references`)
rather than silently dropping them. Katalyst already treats unmatched as an
error. GX's `self_check`, "here are your
collections, some examples, and the files that matched nothing", is the
template for a future `doctor` / `explain` that diagnoses a definition's mapping.

## Variants route checks, not membership

A collection may run different checks on different items via
[variants]({{< relref "../reference/configuration.md" >}}#variants), but that is
a *check-engine* concern, not a storage one. A variant's discriminator is a
predicate over an item's **metadata**: portable across every StorageType, since
each yields a metadata map (frontmatter for a file, columns for a row). It never
touches the seam: membership, `Unmatched`, and `Reference` stay governed by the
definition's `pattern`. Discriminating by *path* would be a storage-type-scoped
condition; it is deferred precisely to keep the seam closed for now.

## Coordinates are the selector

GX's `group_names` *are* the addressing grammar: a batch is addressed by its
asset plus its captured coordinates (`{year, letter, …}`). In Katalyst, the
flat `stem` identity is the degenerate one-coordinate case; richer layouts
(`notes/2020/dune`) grow into multiple coordinates parsed from the path. The
selector grammar and the definition's pattern are two views of the same thing.

## Design lessons (carried + corrected)

Reuse as-is:

- The contract is **two-way**, not one-way (discovery *and* reconstruction).
- **Configured / Inferred ≙ `check` / `infer`:** same axis, already planned.
- **Surface unmatched**, don't swallow it.
- **Coordinates = selector:** design them as one concept.

Do better than GX did (straight from its own TODOs in the recovered code):

- **Prefer an inherently two-way template** (`{name}_{year}.md`) over inverting
  an arbitrary regex. GX inverted a capture-group regex into a `str.format`
  template and the author flagged it as *"almost certainly still brittle"*, a
  template is bidirectional by construction; a regex is not.
- **The pattern must own the file extension**, or reconstruction is ambiguous
  when several extensions are allowed (a GX limitation noted in `util.py`).
- **Keep collection identity separate from within-collection coordinates.**
  GX leaked `data_asset_name` into the coordinate map and regretted it; keep
  them distinct fields.

## What is built, and the seam left open

- **Built:** the `internal/storage` seam (`StorageType`, `StorageInstance`,
  `CollectionDefinition`, `Granularity`, `Reference`), the
  `FilesystemCollectionDefinition` (collection = directory, item = each `*.md`
  file, id = stem, granularity = *file-is-item*), and the config model where an
  instance declares its collections.
- **Open seam:** anything that turns a path into an item identity (or back)
  passes through `CollectionDefinition`, so a second backend (SQLite) can be
  added later without touching the check engine, the CRUD verbs, or selector
  parsing. Multi-coordinate templates, inferred mode, and non-filesystem types
  slot in there.

## Terms

| Term | Meaning |
|---|---|
| **StorageType** | A known backend kind (filesystem, sqlite, ...). |
| **StorageInstance** | A configured instance of a StorageType plus how to reach it. |
| **CollectionDefinition** | The backend↔domain two-way mapping; yields one or more collections. |
| **Data reference** | A backend-native locator (file path, S3 key, table name). |
| **Coordinates** | The captured fields that identify a unit within its collection. |
| **Granularity** | The level (item vs. collection) at which a StorageType attaches a store's units to the domain model. |

[addressing]: {{< relref "core-concepts.md" >}}
</content>
