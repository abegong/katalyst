+++
title = "Bases"
weight = 40
+++

# Bases

A **base** is how Katalyst reaches a backend source and maps that source into the domain model.

Every base must include configuration for **raw** access. Raw access gives Katalyst a stable way to locate content in the source. For a filesystem, that can be a root directory. For SQL, that can be connection information for a specific instance.

A **collectionized** base keeps that raw access and adds collection definitions. Those definitions map base-native references into named collections and item identities that Katalyst commands can address directly. This is where two-way mapping applies.

Katalyst's base model covers filesystem and SQLite backends today and is designed to extend to backends such as Postgres, S3, and hosted APIs.

## Terms

The base model uses several named pieces:

| Term | Meaning |
|---|---|
| **Base type** | A known backend source kind capable of holding collections and items: `filesystem` and `sqlite` today; `postgresql`, `mongodb`, and others later. |
| **Base instance** | A specific, connectable instance of a base type, plus the information needed to reach it. |
| **Collection mapping** | The two-way mapping from a base instance's contents to collections and items. One mapping may yield more than one collection. |
| **Base reference** | A base-native locator: a file path, S3 key, table name, or similar backend address. |
| **Coordinates** | The captured fields that identify a unit within its collection. |
| **Scope** | The domain level, item or collection, at which a base type attaches a base's units to the model. |

In config, a base instance declares the collections it maps, and the instance file is where the collection mapping lives. In code, the implementation seam is `internal/storage/collection.CollectionDefinition`; `internal/project` consumes it rather than implementing the filesystem mapping inline.

Base readers use codecs to decode a matched unit's content into the shape checks and inspectors consume. The markdown filesystem reader uses `internal/codec/markdownbodytext` for frontmatter/body parsing; codecs are shared content adapters, not base backends.

## Collectionized bases use a two-way mapping

When a base is collectionized, mapping has two directions:

- **Forward (discovery):** `path -> match pattern -> captured groups` become
  the unit's *coordinates*.
- **Reverse (reconstruction):** `coordinates -> fill a template -> path`.

The reverse direction is **not optional**. Katalyst needs it the moment
`item add notes/dune` has to decide what file to create; that is the same
path-reconstruction problem. Today it is the degenerate, stem-only case
(`Reference(c, id) -> <dir>/<id>.md`); it grows with the layout.

## The scope principle

**"What does one matched source unit become?" has no global answer; it is a property each base type declares.**

- **Markdown filesystem:** one file = one **item**; a directory of files =
  a **collection**.
- **Tabular (CSV / SQL):** one file/table = one **collection**; its rows =
  **items**.

Both mappings are correct because item and collection are domain roles, not file counts. The definition absorbs that difference: alongside the path-to-coordinates mapping, it declares the **scope** at which a base's units attach to the collection/item hierarchy.

Implication: **item and collection are roles, not file counts.** A base that packs many items into one physical unit (rows in a table) and one that spreads a single item across a whole unit (a markdown file) are both valid.

## Base capability stack

- **Raw base:** Katalyst can connect to the source and reference base-native content.
- **Collectionized base:** a raw base plus collection definitions that map base-native references into domain collections and items.

## Unmatched references are first-class

Katalyst treats unmatched references as errors rather than silently dropping them. A file inside a configured collection's scope that matches no pattern is usually a signal of config drift: the pattern is wrong, the file is misplaced, or the project has gained a new shape that has not been modeled yet.

The same evidence can power a future `doctor` / `explain` command: list the collections, show representative examples, and surface the base references that matched nothing.

## Variants route checks, not membership

A collection may run different checks on different items via
[variants]({{< relref "../../reference/configuration.md" >}}#variants), but that is
a *check-engine* concern, not a base one. A variant's discriminator is a
predicate over an item's **metadata**: portable across every base type, since
each yields a metadata map (frontmatter for a file, columns for a row). It never
touches the seam: membership, `Unmatched`, and `Reference` stay governed by the
definition's `pattern`. Discriminating by *path* would be a base-type-scoped
condition; it is deferred precisely to keep the seam closed for now.

## Coordinates are the selector

In Katalyst, the flat `stem` identity is the degenerate one-coordinate case:
`notes/dune.md` becomes the item id `dune`. Richer layouts (`notes/2020/dune`)
grow into multiple coordinates parsed from the path. The selector grammar and
the definition's pattern are two views of the same thing.

## Design lessons

- **The contract is two-way, not one-way.** Discovery and reconstruction are
  both core base operations.
- **Raw and collectionized are one progression.** A base starts with
  base-native references, then gains collection definitions that make
  collection-aware operations possible.
- **Surface unmatched references.** Silent skips hide drift between the
  base's real contents and the configured model.
- **Coordinates and selectors are one concept.** The fields captured from a
  base reference should be the same fields users and agents use to address
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
  item with a stem id; the SQLite implementation maps a table to a collection
  and each row to an item.
- **Extension point:** anything that turns a base-native reference into an item
  identity (or back) passes through `CollectionDefinition`, so backends can be
  added without touching the check engine, the CRUD verbs, or selector parsing.
  Multi-coordinate templates, inferred mode, and additional non-filesystem
  types slot in there.

[addressing]: {{< relref "_index.md" >}}
