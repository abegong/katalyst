# Plan — collection consolidation and fix extraction

> References [`collection-reorg-spec.md`](./collection-reorg-spec.md) (Spec 1).
> **Status: implementing.**

## Approach

This is a behavior-preserving relocation plus one refactor (the `fix` split), so
the existing suite is the regression guard: every phase ends with `make all`
green and `make docs-gen-check` byte-identical. Each phase is one commit. Package
*names* stay the same where possible (`query`, `filesystem`); only import paths
and a few package homes change, so most edits are mechanical (`git mv` + import
rewrite), exactly like the #83 move.

Order is chosen so each step compiles on its own and the dependency graph never
cycles. Leaves move first; the `fix` split lands last.

## Phases

### Phase 1 — Move `query` down

`internal/project/collection/query` → `internal/storage/collection/query`
(package `query` unchanged). Rewrite the two importers (`config`, `cmd/item`) and
delete the now-empty `project/collection/` (its `doc.go` placeholder goes).
`project/collection/` is dissolved. **Green check:** build + test.

### Phase 2 — Lift `Item` and `CollectionDefinition` into `storage/collection`

New package `collection` at `internal/storage/collection` holding the
`CollectionDefinition` interface (from `storage/storage.go`) and `Item` (from
`storage/item.go`). `storage/` root keeps `StorageType`, `Known`, `Granularity`,
`Reference`. `storage/collection` imports `storage` (for `Granularity`/`Reference`)
and `config`. Update re-exports (`project.Item`) and importers (`inspect`, `cmd`).
**Green check.**

### Phase 3 — Move the filesystem reader

`storage/filesystem.go` → `storage/collection/filesystem/collection.go`
(package `filesystem`, type `FilesystemCollectionDefinition`, constructor
`filesystem.New`). Repoint `project` (was `storage.NewFilesystem`). **Green check.**

### Phase 4 — Split `frontmatter` into the `document` codec

Read half (`Parse`, `Document`, `Kind`, line map; `frontmatter.go`) →
`storage/collection/document` (package `document`). Add `Encode` (the serialize
dual of `Parse`) — initially by relocating the serialization helpers from
`format.go`; `fix` will consume it in Phase 5. Rewrite all codec consumers
(`checks`, `inspect`, `cmd/item`, `cmd/write_validation`, `checktest`,
`cmd/fix`). `internal/frontmatter` is left holding only `Format` + its helpers,
temporarily, for Phase 5 to claim. **Green check.**

### Phase 5 — Split `fix`: transform engine + backend persist

- New package `internal/fix`: the canonical-form transform (`Format` + helpers,
  moved from the `frontmatter` remnant) and the text-fix logic (`applyTextFixes`,
  `textFixers`, moved from `cmd/fix.go`). No file IO. Composes `document.Encode`.
- The atomic temp-rename persist moves from `cmd/fix.go` into
  `storage/collection/filesystem` as a `Write`/persist function beside the read.
- `cmd/fix.go` becomes a thin shell: read item → `fix` transform → if changed,
  filesystem persist.
- Delete the emptied `internal/frontmatter`.
- **Green check**, and assert `fix` output is byte-identical via the existing
  `cmd/fix_test.go` golden/snapshot tests.

### Phase 6 — Docs and final sweep

Root `AGENTS.md` layout tree; new `AGENTS.md` for `storage/collection` and
`internal/fix`; delete `frontmatter/AGENTS.md`; update `formatting.md`,
`storage.md`, `collections.md`; refresh the terminology matrix's Internal-code
column. Confirm `make all` + `make docs-gen-check`. (Glossary needs no new terms.)

## Done criteria

- `make all` green; `make docs-gen-check` byte-identical.
- `internal/frontmatter` and `internal/project/collection` gone.
- `storage/` = backend-kind registry; `storage/collection/` = the read stack
  (`collection.go`, `query/`, `document/`, `filesystem/`); `internal/fix` = the
  transform engine; `cmd/fix.go` thin.
- The only `project/ ↔ storage/` cross-tree edge is `config → …/query` (the
  compromise Spec 2 retires); nothing new deepens centralized config.
