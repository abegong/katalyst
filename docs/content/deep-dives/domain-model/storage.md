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
[domain model]({{< relref "_index.md" >}}): the filesystem is one
backend; SQLite, directories of CSVs, S3 buckets, and hosted APIs are others.
The first real stress test will be **SQLite**, because it is the first backend
that forces the scope question below.

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
[Configuration]({{< relref "../../reference/configuration.md" >}})). In code, the
seam is `internal/storage/collection.CollectionDefinition`; `internal/project` consumes it
rather than implementing the filesystem mapping inline.

Storage readers use codecs to decode a matched unit's content into the shape
checks and inspectors consume. The markdown filesystem reader uses
`internal/codec/markdownbodytext` for frontmatter/body parsing; codecs are
shared content adapters, not storage backends.

## The heart: a two-way mapping

Storage mapping has two directions:

- **Forward (discovery):** `path → match pattern → captured groups` become
  the unit's *coordinates*.
- **Reverse (reconstruction):** `coordinates → fill a template → path`.

The reverse direction is **not optional**. Katalyst needs it the moment
`item add notes/dune` has to decide *what file to create*, that is the same
path-reconstruction problem. Today it is the degenerate, stem-only case
(`Reference(c, id) → <dir>/<id>.md`); it grows with the layout.

## The scope principle

**"What does one matched store unit become?" has no global answer, it is a
property each StorageType declares for its backend.**

- **Markdown filesystem:** one file = one **Item**; a directory of files =
  a **Collection**.
- **Tabular (CSV / SQL):** one file/table = one **Collection**; its rows =
  **Items**.

Both mappings are correct because item and collection are domain roles, not
file counts. The definition absorbs that difference: alongside the
path↔coordinates mapping, it declares the **scope** at which a store's units
attach to the collection/item hierarchy.

Implication: **Item and Collection are roles, not file counts.** A backend that
packs many items into one physical unit (rows in a table) and one that spreads a
single item across a whole unit (a markdown file) are both valid.

## Two modes: Configured vs Inferred

- **Configured:** collections and their patterns are declared explicitly (the
  instance's `collections:` block). This is the `check` path: known structure,
  enforced. *Shipped.*
- **Inferred:** collection names and structure are *discovered* by applying
  the pattern to whatever is in the store. This is the `infer` / `profile`
  path: structure read out of the data. *Planned.*

## Unmatched references are first-class

Katalyst treats unmatched references as errors rather than silently dropping
them. A file inside a configured collection's scope that matches no pattern is
usually a signal of config drift: the pattern is wrong, the file is misplaced,
or the project has gained a new shape that has not been modeled yet.

The same evidence can power a future `doctor` / `explain` command: list the
collections, show representative examples, and surface the backend references
that matched nothing.

## Variants route checks, not membership

A collection may run different checks on different items via
[variants]({{< relref "../../reference/configuration.md" >}}#variants), but that is
a *check-engine* concern, not a storage one. A variant's discriminator is a
predicate over an item's **attributes**: portable across every StorageType,
since each yields a structured attribute object (frontmatter fields for a file,
configured column captures for a row). It never touches the seam: membership,
`Unmatched`, and `Reference` stay governed by the definition's `pattern`.
Discriminating by *path* would be a storage-type-scoped condition; it is
deferred precisely to keep the seam closed for now.

## Coordinates are the selector

In Katalyst, the flat `stem` identity is the degenerate one-coordinate case:
`notes/dune.md` becomes the item id `dune`. Richer layouts (`notes/2020/dune`)
grow into multiple coordinates parsed from the path. The selector grammar and
the definition's pattern are two views of the same thing.

## Design lessons

- **The contract is two-way, not one-way.** Discovery and reconstruction are
  both core storage operations.
- **Configured and inferred modes are the same axis.** `check` uses declared
  structure; `infer` / `profile` discovers structure from the data.
- **Surface unmatched references.** Silent skips hide drift between the
  backend's real contents and the configured model.
- **Coordinates and selectors are one concept.** The fields captured from a
  backend reference should be the same fields users and agents use to address
  the item.
- **Prefer an inherently two-way template** (`{name}_{year}.md`) over inverting
  an arbitrary regex. A template is bidirectional by construction; a regex is
  not.
- **The pattern must own the file extension**, or reconstruction is ambiguous
  when several extensions are allowed.
- **Keep collection identity separate from within-collection coordinates.**
  Collection names and item coordinates answer different questions and should
  stay distinct.

## What is built, and the seam left open

- **Built:** the `internal/storage` seam (`StorageType`, `StorageInstance`,
  `CollectionDefinition`, `Reference`), the
  `FilesystemCollectionDefinition` (collection = directory, item = each `*.md`
  file, id = stem, item scope), and the config model where an
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
| **Scope** | The domain level, item or collection, at which a StorageType attaches a store's units to the model. |

[addressing]: {{< relref "_index.md" >}}
