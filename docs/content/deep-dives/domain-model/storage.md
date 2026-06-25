+++
title = "Bases"
weight = 40
+++

# Bases

The **base layer** is how Katalyst reaches a backend store and maps that store
into the domain model.

Every base includes configuration for **raw** access. Raw access gives Katalyst
a stable way to locate content in the store. For a filesystem, that can be a
root directory. For SQL, that can be connection information for a specific
instance.

A **collectionized** base keeps that raw access and adds collection
definitions. Those definitions map backend-native references into named
collections and item identities that Katalyst commands can address directly.
This is where two-way mapping applies.

Katalyst's storage model covers filesystem backends today and is designed to
extend to backends such as SQLite, Postgres, S3, and hosted APIs.

## Three concepts

The layer is three named pieces, not one. Earlier drafts called the whole thing
a *connector*; that single word was doing two jobs, *how do I reach the store*
and *how does its content map to the model*, so it was split:

| Concept | Meaning |
|---|---|
| **BaseType** | A known backend kind capable of holding collections and items: `filesystem` today; `sqlite`, `postgresql`, `mongodb` later. |
| **BaseInstance** | A specific, connectable instance of a BaseType, plus the information needed to reach it (for `filesystem`, a root directory). |
| **CollectionDefinition** | The two-way mapping from a BaseInstance's contents to collections and items. `FilesystemCollectionDefinition` is the first; one definition may yield **more than one** collection. |

In config, a BaseInstance declares the collections it maps, the base file *is*
where the CollectionDefinition lives (see
[Configuration]({{< relref "../../reference/configuration.md" >}})). In code, the
seam is `internal/storage/collection.CollectionDefinition`; `internal/project` consumes it
rather than implementing the filesystem mapping inline.

Base readers use codecs to decode a matched unit's content into the shape
checks and inspectors consume. The markdown filesystem reader uses
`internal/codec/markdownbodytext` for frontmatter/body parsing; codecs are
shared content adapters, not storage backends.

## Collectionized bases use a two-way mapping

When a base is collectionized, mapping has two directions:

- **Forward (discovery):** `path → match pattern → captured groups` become
  the unit's *coordinates*.
- **Reverse (reconstruction):** `coordinates → fill a template → path`.

The reverse direction is **not optional**. Katalyst needs it the moment
`item add notes/dune` has to decide *what file to create*, that is the same
path-reconstruction problem. Today it is the degenerate, stem-only case
(`Reference(c, id) → <dir>/<id>.md`); it grows with the layout.

## The scope principle

**"What does one matched store unit become?" has no global answer, it is a
property each BaseType declares for its backend.**

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

## Base capability stack

- **Raw base:** Katalyst can connect to the store and reference backend-native
  content.
- **Collectionized base:** a raw base plus collection definitions that map
  backend-native references into domain collections and items.

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
predicate over an item's **metadata**: portable across every BaseType, since
each yields a metadata map (frontmatter for a file, columns for a row). It never
touches the seam: membership, `Unmatched`, and `Reference` stay governed by the
definition's `pattern`. Discriminating by *path* would be a storage-type-scoped
condition; it is deferred precisely to keep the seam closed for now.

## Coordinates are the selector

In Katalyst, the flat `stem` identity is the degenerate one-coordinate case:
`notes/dune.md` becomes the item id `dune`. Richer layouts (`notes/2020/dune`)
grow into multiple coordinates parsed from the path. The selector grammar and
the definition's pattern are two views of the same thing.

## Design lessons

- **The contract is two-way, not one-way.** Discovery and reconstruction are
  both core storage operations.
- **Raw and collectionized are one progression.** A base starts with
  backend-native references, then gains collection definitions that make
  collection-aware operations possible.
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

## Seam and extension points

- **Core seam:** `internal/storage` defines `BaseType`, `Scope`, and
  `Reference`; `internal/project` assembles `BaseInstance` values, and
  `internal/storage/collection` defines `CollectionDefinition`. The filesystem
  implementation maps a directory to a collection and each `*.md` file to an
  item with a stem id.
- **Extension point:** anything that turns a path into an item identity (or back)
  passes through `CollectionDefinition`, so a second backend (SQLite) can be
  added later without touching the check engine, the CRUD verbs, or selector
  parsing. Multi-coordinate templates, inferred mode, and non-filesystem types
  slot in there.

## Terms

| Term | Meaning |
|---|---|
| **BaseType** | A known backend kind (filesystem, sqlite, ...). |
| **BaseInstance** | A configured instance of a BaseType plus how to reach it. |
| **CollectionDefinition** | The backend↔domain two-way mapping; yields one or more collections. |
| **Data reference** | A backend-native locator (file path, S3 key, table name). |
| **Coordinates** | The captured fields that identify a unit within its collection. |
| **Scope** | The domain level, item or collection, at which a BaseType attaches a store's units to the model. |

[addressing]: {{< relref "_index.md" >}}
