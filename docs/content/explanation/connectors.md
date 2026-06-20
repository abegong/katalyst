+++
title = "Connectors"
weight = 80
+++

# Connectors

> **Status: future — not shipped.** Captured now because it shapes the seams
> the current code leaves open. Katalyst today ships a single *trivial*
> connector (one flat directory, `*.md` files, filename stem = item id);
> everything here is what that abstraction grows into.

## What a connector is

A **connector** is the two-way mapping between a backend store and the
Katalyst domain model. It answers: *what collections and items does this
store contain, and where does each one live?* — in both directions.

It is Katalyst's realization of the **data interface** concept from
the [general model]({{< relref "general-model.md" >}}): the filesystem is one backend;
SQLite, directories of CSVs, S3 buckets, and hosted APIs are others. The
first real stress test will be **SQLite**, because it is the first backend
that forces the granularity question below.

## Lineage: GX legacy DataConnectors

The design is adapted from Great Expectations' V3 `DataConnector` layer
(recovered for reference *outside this repo* at
`great_expectations/recovered_data_connector/`; originally GX commit
`6cd804579`, removed in `27eb8d28b`). A GX DataConnector defined a
`regex + group_names` naming convention that mapped each file/key in a store
to a `BatchDefinition`, plus the inverse mapping back to a path.

### The heart: a two-way mapping

- **Forward (discovery):** `path → match pattern → captured groups` become
  the unit's *coordinates*.
- **Reverse (reconstruction):** `coordinates → fill a template → path`.

The reverse direction is **not optional**. Katalyst needs it the moment
`item add notes/dune` has to decide *what file to create* — that is the same
path-reconstruction problem. It is trivial while a collection is a flat
directory and grows with the layout.

## Concept mapping: GX → Katalyst

| GX (legacy V3) | Katalyst |
|----------------|----------|
| Datasource | data interface / backend (a directory, today) |
| **DataConnector** | the backend↔domain mapping described here |
| DataAsset (`data_asset_name`) | **Collection** |
| **Batch / BatchDefinition** | **Item** *(in markdown)* — see granularity principle |
| PartitionDefinition (`group_names` → values) | the item's **coordinates** |
| BatchRequest / PartitionQuery | a **selector** (the [addressing] grammar) |
| BatchSpec | the resolved fetch instruction (the file path) |

### The granularity principle (locked)

**"What does one matched store unit become?" has no global answer — it is a
property each connector declares for its backend.**

- **Markdown filesystem:** one file = one **Item**; a directory of files =
  a **Collection**.
- **Tabular (CSV / SQL):** one file/table = one **Collection**; its rows =
  **Items**.

This is why a GX *Batch* maps to a Katalyst *Item* in the markdown world but
to a *Collection* in the tabular world — and both are correct. The connector
exists precisely to absorb that impedance: alongside the path↔coordinates
mapping, a connector declares the **level** at which a store's units attach
to the collection/item hierarchy.

Implication for the domain model: **Item and Collection are roles, not file
counts.** A backend that packs many items into one physical unit (rows in a
table) and one that spreads a single item across a whole unit (a markdown
file) are both valid; the connector names which.

## Two modes: Configured vs Inferred

GX shipped both, and they map cleanly onto Katalyst verbs:

- **Configured** — collections and their patterns are declared explicitly
  (Katalyst's `katalyst.yaml`). This is the `check` path: known structure,
  enforced.
- **Inferred** — collection names and structure are *discovered* by applying
  the pattern to whatever is in the store. This is the `infer` / `profile`
  path: structure read out of the data.

## Unmatched references are first-class

GX tracked files that matched no pattern (`get_unmatched_data_references`)
rather than silently dropping them. Katalyst already treats unmatched as an
error (see [Configuration]({{< relref "configuration.md" >}})). GX's `self_check` — "here are
your collections, some examples, and the files that matched nothing" — is the
template for a future `katalyst doctor` / `explain` that diagnoses a
connector's mapping.

## Coordinates are the selector

GX's `group_names` *are* the addressing grammar: a batch is addressed by its
asset plus its captured coordinates (`{year, letter, …}`). In Katalyst, the
flat `stem` identity is the degenerate one-coordinate case; richer layouts
(`notes/2020/dune`) grow into multiple coordinates parsed from the path. The
selector grammar and the connector pattern are two views of the same thing.

## Design lessons (carried + corrected)

Reuse as-is:

- The contract is **two-way**, not one-way (discovery *and* reconstruction).
- **Configured / Inferred ≙ `check` / `infer`** — same axis, already
  planned.
- **Surface unmatched**, don't swallow it.
- **Coordinates = selector** — design them as one concept.

Do better than GX did (straight from its own TODOs in the recovered code):

- **Prefer an inherently two-way pattern** (a template like `{name}_{year}.md`)
  over inverting an arbitrary regex. GX inverted a capture-group regex into a
  `str.format` template and the author flagged it as *"almost certainly still
  brittle"* — a template is bidirectional by construction; a regex is not.
- **The pattern must own the file extension**, or reconstruction is ambiguous
  when several extensions are allowed (a GX limitation noted in `util.py`).
- **Keep collection identity separate from within-collection coordinates.**
  GX leaked `data_asset_name` into the coordinate map and regretted it; keep
  them distinct fields.

## v0 stance and the seam to leave

- v0 = **one connector, hardcoded**: collection = the directory, item = each
  `*.md` file, id = filename stem, granularity = *file-is-item*.
- **Leave the seam:** anything that turns a path into an item identity (or
  back) should pass through one narrow interface, so a second connector
  (SQLite) can be added later without touching the check engine, the CRUD
  verbs, or selector parsing.

## Terms

| Term | Meaning |
|---|---|
| **Connector** | The backend↔domain two-way mapping. |
| **Data reference** | A backend-native locator (file path, S3 key, table name). |
| **Coordinates** | The captured fields that identify a unit within its collection. |
| **Granularity** | The level (item vs. collection) at which a connector attaches a store's units to the domain model. |

[addressing]: {{< relref "domain-model.md" >}}
