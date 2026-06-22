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

The exact home of `Item` (today in `internal/project`) versus a move into
`internal/storage` is a packaging detail resolved during implementation; the
contract is what matters. `FilesystemCollectionDefinition` implements every
method by lifting the current `project` bodies verbatim — this spec is a
refactor behind a new boundary, not a behavior change.

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

A `StorageInstance` becomes a declarable config object, joining `schemas/` and
`collections/` as a convention-discovered kind:

```
.katalyst/
  config.yaml
  schemas/
  collections/
  storage/
    notes-vault.yaml          # one StorageInstance per file; name = stem
```

```yaml
# .katalyst/storage/notes-vault.yaml
type: filesystem
root: .                       # directory, relative to the repo root
```

A collection points at the instance whose contents it maps:

```yaml
# .katalyst/collections/books.yaml
storage: notes-vault          # optional; defaults to the implicit local instance
path: notes/books
pattern: "*.md"
schema: book
checks: [ ... ]
```

**Backward compatibility.** When no `storage/` is declared and a collection
omits `storage:`, Katalyst synthesizes one implicit `filesystem` instance
rooted at the repo root — exactly today's behavior. Every existing config keeps
working untouched; `storage/` is purely additive.

**One definition → many collections** is realized as follows: a single
StorageInstance (one directory tree) backs multiple collections that each
declare their own `path`/`pattern` against it. A directory holding files for
two collections is two collection files sharing one `storage:` instance with
different patterns. The `FilesystemCollectionDefinition` for an instance
enumerates all collections bound to it and answers `Collections()` with that
set. (Whether collections are *also* expressible inline in a single definition
file, GX-asset-style, is Open Question 1.)

### What this spec builds vs. defers

Builds now:

- `internal/storage` package: `StorageType` registry, `StorageInstance`,
  `CollectionDefinition` interface, `Granularity`, `Reference`.
- `FilesystemCollectionDefinition` wrapping today's `Items`/`ItemPath`/
  `Unmatched`/`ItemAt` logic, behavior-preserving.
- `internal/project` refactored to drive the interface; selectors, CRUD verbs,
  and the check engine untouched.
- `storage/` config kind + collection `storage:` ref + the implicit default
  instance.
- The `connector` → triad doc/vocabulary migration.

Defers (seam left open, not implemented):

- Multi-coordinate templates and the GX two-way port (above).
- **Inferred** mode — collection names *discovered* from the store rather than
  declared — maps to the future `infer`/`profile` path, not `check`.
- A `doctor`/`explain` command (GX's `self_check`: "here are your collections,
  some examples, and what matched nothing").
- Any non-filesystem StorageType.
- A file mapping into more than one collection (see Open Question 3).

## Open Questions

1. **Definition-centric vs. collection-centric config for "one definition →
   many collections."**

   **Context.** The maintainer chose "one CollectionDefinition yields N
   collections." There are two ways to express that in `.katalyst/`, and they
   differ in where a collection's `checks:` are authored.

   **Choices & tradeoffs.**
   - **(A) Collection-centric (recommended).** Keep one file per collection
     under `collections/`, each with its own `checks:`; collections share a
     StorageInstance via `storage:`. "One definition → many collections" is an
     emergent property (many collection files, one instance). *Buys:* minimal
     churn, today's per-collection check authoring is preserved, invariant
     "a collection owns its checks" is untouched. *Costs:* the
     `CollectionDefinition` is then mostly a *code* seam, not a first-class
     config file — slightly less faithful to the maintainer's phrasing of a
     definition that "maps a filesystem to one or more collections."
   - **(B) Definition-centric (GX-faithful).** A new `collections/`-style file
     *is* the definition: it names a StorageInstance and enumerates several
     collections inline (GX "configured assets"), each with its pattern and
     checks. *Buys:* one file describes a whole tree's mapping; closest to GX
     and to "a definition yields many collections." *Costs:* larger change to
     how checks are authored; a collection's config is no longer one
     self-contained file; the load/merge path in `internal/config` grows.
   - **(C) Hybrid.** Support both: the implicit default for a lone collection
     stays one-file (sugar), and a definition file can group several. *Costs:*
     two code paths to maintain and document.

   **Recommendation:** (A). It satisfies the locked decision with the least
   disruption and keeps `CollectionDefinition` as the narrow seam issue #31
   actually asks for. Revisit (B)/(C) only if a real layout proves (A)
   awkward.

2. **StorageInstance config: own `storage/` kind vs. a `storage:` section in
   `config.yaml`.**

   **Context.** Instances need a config home. Existing kinds (`schemas`,
   `collections`) are convention-discovered directories *and* can be listed
   explicitly under `config.yaml` (`discovery: explicit`).

   **Choices & tradeoffs.**
   - **(A) New `storage/` directory kind (recommended).** Mirrors
     `schemas/`/`collections/` exactly, including the `discovery`/`format`
     machinery in `internal/config`. *Buys:* consistency, one file per instance,
     name-from-stem. *Costs:* a third kind to wire through `loadX`.
   - **(B) A `storage:` map in `config.yaml`.** Instances are usually few and
     small (a type + a root). *Buys:* less ceremony for the common one-instance
     case. *Costs:* inconsistent with the directory-per-definition convention
     the project deliberately chose (`internal/config/README.md`).

   **Recommendation:** (A), for consistency. The implicit default instance
   means the common case still writes nothing.

3. **The "one file maps to more than one collection" case — confirm deferral.**

   **Context.** The maintainer flagged this as exotic and chose to defer it.
   Recording it explicitly so the interface doesn't accidentally foreclose it.
   Domain-model invariant #4 ("a file belongs to exactly one collection")
   currently holds and `Resolve` de-duplicates by path.

   **Choices & tradeoffs.** Keep invariant #4 (a `Reference` resolves to one
   collection+item) — simplest, matches today, keeps check ownership and
   selector resolution unambiguous. The seam does not *prevent* a future
   many-to-one mapping (nothing in the interface asserts uniqueness), so
   lifting it later is additive.

   **Recommendation:** Confirm deferral; keep invariant #4 and note the
   future possibility in the reframed deep dive. *(Pre-answered as "defer";
   listed so it's recorded in Design when this closes.)*

4. **Package boundary: does `Item` move to `internal/storage`?**

   **Context.** `project.Item` is `{Collection, ID, Path}`. If
   `CollectionDefinition` lives in `internal/storage` and returns items, either
   `storage` imports `project` (and `project` imports `storage` → cycle) or
   `Item` moves down to `storage`.

   **Choices & tradeoffs.** (A) Move `Item` (and likely `Selector`/`Resolution`
   stay in `project`) into `internal/storage`; `project` re-exports or wraps.
   (B) Define a smaller `storage.Unit` and have `project` adapt it to `Item`.
   *Recommendation:* (A) — fewest types, no adapter layer — but this is an
   implementation detail to settle in the plan, not a user-visible decision.

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
- `reference/configuration.md`: document the `storage/` kind, the
  `type`/`root` keys, the collection `storage:` ref, and the implicit default
  instance.
- `deep-dives/domain-model.md` (lines 169–170): update the Selector note's
  `connectors.md` relref to `storage.md` and the "connector coordinates"
  wording.
- `contributing/how-we-document.md` (line 28) and `how-we-plan.md` (line 79):
  swap "connectors" for "storage" in the evergreen-deep-dive references.

**Developer docs:**

- `internal/storage/doc.go`: new package doc — the three concepts, the two-way
  contract, granularity, and the GX provenance/correction.
- `internal/config/README.md`: document the `storage/` kind and the
  collection `storage:` field; note the implicit default instance.
- `internal/project/` package doc: update to say it now consumes the
  `internal/storage` seam rather than implementing the mapping.
- `AGENTS.md`: record the convention "path ⇄ item-identity translation passes
  through `internal/storage.CollectionDefinition`; do not inline filesystem
  assumptions elsewhere."

**Specs (cross-references, not user docs):** `product/specs/cli-spec.md`,
`product/specs/dogfood-docs-spec.md`, and `product/v0-implementation-plan.md`
mention `connectors.md`; update those relrefs to `storage.md` when this ships.

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
</content>
</invoke>
