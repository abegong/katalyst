# Spec — collection consolidation and fix extraction

> **Status: planning.** Phase 2 of package realignment (#39), building on the
> `project/{config, collection/query}` nesting shipped in #83. **Supersedes**
> `frontmatter-split-spec.md` (which recommended an `internal/document` home and
> rejected a collection home — both overturned below). Pairs with
> `config-distribution-spec.md` (Spec 2), which removes the one dependency
> compromise this spec deliberately tolerates.
>
> **Follow-up:** `codec-layer-spec.md` supersedes this spec's decision to keep
> the markdown/frontmatter codec under `storage/collection/document`; the codec
> now lives under `internal/codec/markdownbodytext`.

## Overview

The collection concept is the most scattered thing in the tree: its declaration,
its backend read, its content parsing, and its query grammar live in four
different packages, and the one module whose name maps to no concept —
`frontmatter/` — holds a piece of it. This spec consolidates the collection
**read path** into a single home under `storage/`, and splits `fix` into a
backend-agnostic transform engine plus a backend-owned persist step that mirrors
the read. It is relocation plus one behavior-preserving refactor; it does not
change how config is parsed (that is Spec 2).

## Value

After this lands, a reader can answer "how does katalyst read a collection's
items from a backend?" by opening one directory, and "how does `fix` work?" by
opening one more — instead of stitching together `config`, `storage`,
`frontmatter`, and `cmd/fix.go`. It also makes the SQL backend a localized
addition (`storage/collection/sql/`) rather than a cross-package change, and it
finishes retiring the `frontmatter` name, the last module that names a
file-format detail instead of a concept.

## Current State

"Read a markdown collection's items" is currently four packages and a verb:

| Piece | Today | Concept |
|---|---|---|
| Collection declaration | `config.Collection` (`project/config`) | Collection (config) |
| Structural read (locate items) | `storage.CollectionDefinition` + `FilesystemCollectionDefinition` | Collection (backend read) |
| Content read (parse a file) | `frontmatter.Parse` / `Document` / `Kind` | Item file-form |
| Query grammar | `query` (`project/collection/query`) | Operation (filter) |
| Item locator | `storage.Item` (`{Collection, ID, Path}`) | Item |

Two facts from the code anchor the redesign:

- **Items are already thin.** `storage.Item` is `{Collection, ID, Path}` — it
  *locates*, it carries no logic (`storage/item.go`).
- **Parsing already happens in the collection layer.** `inspect`'s
  `CollectionView` resolves items via the `CollectionDefinition` and parses each
  one inside the collection-addressed read surface (`inspect/collection.go`).
- **The storage layer already designates the read seam as the backend extension
  point.** "A new StorageType is added here when its CollectionDefinition lands"
  (`storage/storage.go`), and `Granularity` already reserves the tabular case.

`fix` is split too: the canonical formatter `frontmatter.Format` has exactly one
caller — `cmd/fix.go:82` — while the orchestration (`fixOne`, `applyTextFixes`,
`textFixers`, the atomic temp-rename write) sits in `cmd/fix.go`. Unlike its
sibling `check` (engine in `internal/checks`, thin `cmd/check.go`), `fix` has no
engine package.

## Design

A **collection is the read abstraction over a backend**: it makes a store's data
readable as items. Markdown-on-filesystem parsing is one special case of that
read, parallel to what a SQL `SELECT` + row-decode will be for a SQL backend.
Items stay thin; collections carry the logic. A storage backend *contains*
collections (a `StorageInstance` already "embeds the collections it maps"), so
the containment is `storage ⊃ collection ⊃ item`, and the directory tree should
say so.

### Target layout

```
internal/
  storage/
    storage.go            StorageType, Known, Granularity, Reference   — the backend-kind registry
    collection/
      collection.go       CollectionDefinition interface, Item          (moved up from storage/)
      query/              filter/sort grammar                           (moved DOWN from project/collection/)
      document/           Parse + Encode, Document, Kind  — the markdown codec (decode/encode)
      filesystem/
        collection.go     FilesystemCollectionDefinition: read (glob/locate, document.Parse)
                          AND persist (atomic temp-rename write) — both halves of FS item IO
      sql/                (future) SQLCollectionDefinition: SELECT→item, UPDATE←item
  fix/
    fix.go                the transform engine: canonical-form + text fixes — decides WHAT to write
                          (backend-agnostic; no file IO)
  project/
    config/               unchanged here (Spec 2 owns it)
    project.go, selector.go
  checks/  inspect/       unchanged in place; import paths updated
  cmd/
    fix.go                thin cobra shell over internal/fix
```

`storage/` stops being a vague "seam" and becomes exactly the **Storage** concept
— the registry of backend kinds. Everything about turning a backend's contents
into items lives under `storage/collection/`.

### One collection home; query moves down

There is a single `collection` package, with `query`, `document`, and the
per-backend readers beneath it. This **reverses the `project ⊃ collection`
nesting from #83**: `project/collection/query` moves to
`storage/collection/query`, and `project/collection/` is dissolved. That reversal
is deliberate — the collection concept is read-centric (it exists to read from a
backend), so it belongs with `storage`, not with the workspace/config layer.
`project/` keeps `config/` plus selectors and resolution.

### The read stack, split three ways

- **`collection/collection.go`** — the backend-neutral contract: the
  `CollectionDefinition` interface and the thin `Item`. Imports `config` (item
  carries its `config.Collection`) and `storage` (for `Granularity`/`Reference`).
- **`collection/document/`** — the markdown codec (`Parse`, `Document`, `Kind`).
  A **leaf** (no internal imports), and a *sibling* of the backend readers rather
  than inside `filesystem/`, because parsing markdown is format-specific, not
  filesystem-specific: a future backend that stores markdown bytes elsewhere
  reuses it, while the SQL reader never touches it.
- **`collection/filesystem/collection.go`** — the filesystem
  `CollectionDefinition`: globs and locates items (structural read), reads their
  bytes, and calls `document.Parse` for the content read. The two halves of "read
  a markdown item" finally sit together.

The broadly-shared consumers of the codec (`checks`, `inspect`, `cmd/item`) move
from `frontmatter.Parse`/`frontmatter.Document` to
`collection/document.Parse`/`.Document`. This is a wide but mechanical rename.

### `fix` splits: transform (agnostic) vs persist (backend)

`fix` mirrors the read path. Reading is *fetch bytes (backend) → decode
(`document`) → `Document`*; writing is its dual: *compute desired content (`fix`)
→ encode (`document`) → persist (backend)*. So `fix` splits along the same seam
the read path already has:

- **`internal/fix` — the transform engine, backend-agnostic.** It decides *what*
  the corrected content should be: the canonical-form policy (sorted keys, block
  style, single trailing newline) and the `text_forbids` body fixes, including the
  re-verification that a fix actually resolved its violation. It operates on
  content, not files — **no file IO.** This is the top-level operation engine,
  parallel to `check`/`inspect`, with `cmd/fix.go` a thin shell over it.
- **The persist step lives with the backend read**, in
  `storage/collection/filesystem`: the atomic temp-rename `os.Rename` write moves
  out of `cmd/fix.go` and sits beside the FS read, because *how* you write is
  backend-specific. **SQL test:** the FS backend persists with a temp-rename; a
  SQL backend would persist the same corrected value with an `UPDATE`. The
  transform that produced the value is identical for both — that is why it is
  `fix`'s, and the write is the backend's.
- **The serialize mechanism is the codec's.** Turning a `Document` back into bytes
  is the encode dual of `document.Parse`, so it belongs in `collection/document`
  (as `Encode`), which `fix` composes — keeping `fix` pure policy, not byte
  plumbing.

`cmd/fix.go` orchestrates the three: read each item (backend) → transform (`fix`)
→ if changed, persist (backend). `fix` imports `checks`/`plaintext` (text-fix
re-verification) and `collection/document` (decode/encode); none import `fix`, so
no cycle. Whether the backend write becomes a formal `CollectionDefinition`
method or stays a package function in `filesystem` is a plan-level detail —
leaning a function now, formalized onto the interface when a second backend lands.

### Dependency analysis

The move is cycle-free. `document` and `query` are leaves. `storage/` (root)
becomes a leaf once `CollectionDefinition` leaves it. The chains:

- `storage/collection → config → storage/collection/query` (query is a leaf — no
  loop back).
- `storage/collection/filesystem → {config, storage/collection, storage,
  document}` — all lower, acyclic.
- `checks`/`inspect`/`fix → storage/collection/document` (leaf) — acyclic.

**The one compromise, deliberately tolerated.** `config` (under `project/`)
imports `query` (now under `storage/`), while `storage` imports `config`. No
import cycle, but `project/` and `storage/` each reference into the other's
subtree, so neither is a clean layer. This is the cost of keeping `config`
centralized. **Spec 2 (config distribution) removes it**: when collection config
is owned by the collection rather than a central `config` package, the
`config → query` edge becomes intra-collection and the interleaving disappears.
This spec therefore accepts the edge as temporary and, per the constraint below,
does nothing to deepen it.

### What does NOT change here

- **The config parsing model.** `config.Collection`, `normalizeCheck`, the
  central typed config — untouched. Spec 2 owns that. This spec must not deepen
  centralized config (e.g. don't add new central typed config to smooth the move).
- **Check and inspector internal logic.** Only their import paths to the codec
  change.
- **Behavior.** `make all` must stay green throughout, and `fix`'s output and
  on-disk results stay byte-identical. The one structural change beyond relocation
  is splitting `fix`'s transform from its persist; both are behavior-preserving.

## Open Questions

_None._ Both prior questions are resolved:

1. **`Reference` and `Granularity` stay in `storage/` root** for now — they
   describe the backend kind (`Granularity`'s own comment calls it "a property of
   the StorageType"), and `collection/` imports `storage/` for them. Revisit only
   if a second backend makes the split awkward.
2. **The write splits** (see "`fix` splits" above): the backend-agnostic
   transform is `internal/fix`; the persist (atomic temp-rename today, `UPDATE`
   for SQL) lives with the backend read in `storage/collection/filesystem`. The
   only residual detail — backend write as a `CollectionDefinition` method vs a
   package function — is plan-level, leaning function now.

## Documentation updates

- **Root `AGENTS.md`** — rewrite the layout tree for the new `storage/collection/`
  shape and `internal/fix`; remove the `internal/frontmatter` and
  `internal/project/collection` lines.
- **`internal/frontmatter/AGENTS.md`** — deleted; new `AGENTS.md` files for
  `storage/collection/` and `internal/fix` pointing at the frontmatter and fix
  deep dives.
- **`docs/content/deep-dives/domain-model/frontmatter.md`** and
  **`docs/content/deep-dives/domain-model/fix.md`** — update the "parsing and
  formatting live in `internal/frontmatter`" line to the
  `collection/document` + `internal/fix` split.
- **`docs/content/deep-dives/domain-model/storage.md`, `collections.md`** — align to the new
  module homes (storage = backend registry; collection = read stack).
- **`docs/content/reference/glossary.md`** — confirm Document, Item, and the
  (existing) fix wording point at the new packages; no new terms.
- **`product/specs/domain-model-terminology-matrix.md`** — refresh the
  Internal-code column for Collection, Item, Document, Query, and fix once landed.
- **Generated reference** — `make docs-gen-check` must stay byte-identical (no
  registry labels change).

## Rejected alternatives

- **`storage/collection/` *and* `project/collection/` both (option i).** Rejected:
  two `collection` packages, the concept split across two parents. The single home
  is the point.
- **Pull the read stack *up* into `project/collection/` (option ii).** Viable and
  dependency-cleaner (storage collapses to a leaf, no interleaving), but it frames
  collection as a workspace/config concern rather than a read-from-backend one.
  The chosen `storage ⊃ collection` matches the read-centric model; the
  interleaving it costs is retired by Spec 2 anyway.
- **`internal/document` as a top-level home (the prior spec's pick).** Rejected:
  it separates the content read from the structural read and frames parsing as
  backend-neutral when it is part of collection read. The codec lives under
  `collection/` instead.
- **Fold `document` into `filesystem`.** Rejected: names a markdown codec
  "filesystem" and blocks reuse by a future non-filesystem markdown store.
- **Leave `fix` in `cmd/` / the codec package.** Rejected: keeps `fix`'s defining
  logic out of an engine and leaves it the only operation without one.
