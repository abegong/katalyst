# Cross-document checks — Tier 3

> **Status: planning.** Checks whose verdict depends on more than one document:
> foreign keys between collections, uniqueness across a collection, mirror
> invariants, and link/anchor resolution. Deterministic, no LLM — but they need
> a collection-aware context the engine doesn't have today.

## Overview

Every check today sees exactly one document. This spec adds a `relation` family
whose checks see the *project*: a frontmatter field that must reference an item
in another collection, a field that must be unique across a collection, a
collection that must mirror another item-for-item, a relative link or `#anchor`
that must resolve. The cost is structural — checks need access to siblings and
to other collections — so the spec is as much about the engine seam as the
checks themselves.

## Value

This is the "relations between documents" capability the domain model lists
under **Out of scope (today)** — *"A schema can constrain one document at a
time. No `$ref` to other documents, no foreign keys. Planned."*
([domain model](../../docs/content/explanation/domain-model.md)).
It is the largest deterministic gap in the product: the moment a project has
more than one collection, users want referential integrity between them, and
nothing can express it.

## Current state

`Context` carries one file (`FilePath`, `Doc`, `Meta` in
`internal/checks/checks.go`). The `check` lifecycle streams per item: resolve
selectors to items, then for each item read bytes, parse, resolve schema, build
the check list, run, format (domain model → Lifecycle of `check`). Checks are
independent and stateless — domain-model invariant 4, "a collection owns its
checks; an item belongs to one collection," and the explicit stance under
Out of scope: *"Derived state. No index, no cache file… Every run is
stateless."*

The pieces a cross-document check needs already exist but aren't wired into
`Context`:

- The config knows every collection, its directory, and its `pattern`
  (`internal/config/config.go`, `Collections`).
- Item identity is the filename stem; reverse resolution (id → path) is the
  path-reconstruction problem the connector layer is being designed around
  ([connectors](../../docs/content/explanation/connectors.md) →
  "The reverse direction is not optional").
- Selectors already address project / collection / item depth
  (`internal/project/selector.go`).

## Design

### A project view in the check context

Cross-document checks need to answer "does item X exist in collection Y" and
"what values does field F take across collection Y." Add a read-only project
accessor to `Context`, populated once per `check` invocation:

```go
type Context struct {
    FilePath string
    Doc      *frontmatter.Document
    Meta     map[string]any
    Project  ProjectView // nil for single-document checks; set for relation checks
}

// ProjectView answers cross-document questions without exposing the loader.
type ProjectView interface {
    HasItem(collection, id string) bool
    Items(collection string) []ItemRef          // id + resolved path
    FieldValues(collection, field string) []ValueRef // for uniqueness / joins
}
```

Single-document checks (Tiers 1–2) ignore `Project`; relation checks require it.
The interface — not a concrete loader — keeps the connector seam intact: a
SQLite connector later answers the same questions without the check engine
knowing how.

### Two-phase run

Per-item streaming can't satisfy a uniqueness check that must see all items
first. Split the relation family into a phase that runs after the per-item pass:

1. **Index pass.** Walk every in-scope collection once, building the
   `ProjectView` (id sets per collection, field-value multimaps for fields named
   by relation checks). One pass, in memory, discarded at process exit.
2. **Relation pass.** Run relation checks with the populated `ProjectView`.

This is the schema-compile cache pattern (domain-model invariant 7, "once per
process per absolute path") applied to collection contents: a per-run in-memory
index, not a persisted one. It does **not** introduce a cache file — the
stateless-run invariant holds; "stateless across runs" and "indexed within a
run" are compatible.

### The relation checks (catalog)

| Kind | Enforces |
|---|---|
| `relation_field_references` | `field`'s value is the id of an item in `collection` (foreign key). `field` may be a scalar or a list. |
| `relation_unique` | No two items in this collection share `field`'s value (e.g. `isbn`). |
| `relation_mirror` | Every item here has a counterpart in `collection` with the same id (parity between, e.g., `posts` and `summaries`). |
| `relation_no_cycles` | Following `field` (an id or list of ids) across items forms no cycle / a valid tree. |
| `markdown_link_target_exists` | Each relative link/image target resolves on disk, relative to `ctx.FilePath`. Single-file — needs only `FilePath` — but ships with this family for cohesion. |
| `markdown_anchor_exists` | Each in-page `#fragment` matches a heading in the same file (reuses the Tier 1–2 heading walk). |

`relation_field_references` is the keystone: resolving a field value to a target
item path is exactly the connector's coordinates→path reconstruction. Build it
against `ProjectView.HasItem`, so the connector layer supplies the resolution
when it lands rather than this spec hard-coding stem lookups.

### Reporting across files

A violation is still anchored to one file: the `Path`/`Line` point at the
offending field in the *referencing* item, and the `Message` names the target
("references unknown item \"authors/alice\""). This keeps the existing
`path:line: /pointer: message` format (domain model → Validation result) and
avoids inventing a two-file violation shape.

## Open questions

1. **Scope vs. dependencies.** Checking one item (`check notes/dune`) may
   require loading *another* collection to resolve a foreign key. Does a narrow
   selector still trigger a full index pass over referenced collections, or do
   we resolve targets lazily per reference? _Leaning: lazy `HasItem` for
   `references`, full index only for whole-collection checks (`unique`,
   `mirror`) — but confirm the performance shape._
2. **Connector timing.** `relation_field_references` reuses the connector's
   reverse mapping, which is itself unshipped (connectors doc is "future — not
   shipped"). Do we build `ProjectView` against today's hardcoded stem identity
   and adopt the connector interface when it lands, or block this work on the
   connector seam? _Leaning: build against stem identity behind the
   `ProjectView` interface so the swap is internal._
3. **`additionalProperties: false` interaction.** A foreign-key field is user
   data, unlike the `schema:` directive — but confirm relation checks compose
   cleanly with object-schema validation on the same field (e.g. a field that is
   both a string per JSON Schema and a foreign key per `relation`).

## Rejected alternatives

- **JSON Schema `$ref` across documents.** JSON Schema `$ref` composes schemas,
  not data — it can't express "this value is an id that exists in collection Y."
  Foreign keys are a data-integrity concern; modeling them as schema refs would
  overload `$ref` and tie referential integrity to the object family.
- **A persisted index / `.katalyst/` cache file.** Tempting for large projects,
  but it breaks the stateless-run invariant and adds cache-invalidation
  surface. The per-run in-memory index is enough until profiling says otherwise.
- **Embedding relations in the object family.** Relations are inherently
  multi-document; the object family is single-document by definition. A separate
  `relation` family keeps the families' meaning clean and the docs navigable.
