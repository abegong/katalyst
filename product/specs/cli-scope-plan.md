# Plan - CLI scope
> Spec: [CLI scope](./cli-scope-spec.md)
> **Status: planning.** Decisions 1 to 5 shipped as prose. This plan implements
> decision 6: `fix` gates on whether a collection exposes a text body, not on
> the backend name.

## Current State

The spec's first five decisions are documentation and conventions, and landed
already. Decision 6 is the only behavior change.

| File | Current behavior | Plan pressure |
|---|---|---|
| `cmd/fix.go:171-196` | `fixOne(path string, c project.Collection, check bool)` refuses SQLite outright, then does `os.ReadFile(path)` and `filesystem.Write(path, result)` directly. | Take a `project.Item`, route IO through `project.ReadItem` / `project.UpdateItem`, and gate on the collection's text-content capability. |
| `cmd/fix.go:152-165` | `fixResolution` iterates `res.Items` and calls `fixOne(item.Path, item.Collection, ...)`. | Pass the item and the `*project.Project` it was resolved from. |
| `cmd/fix.go:22-32` | The `Long` help describes frontmatter canonicalization only, and is snapshot-pinned. | State the text-content requirement; regenerate the snapshot. |
| `internal/storage/collection/parse.go:14-55` | `Collection` carries `StorageType`, `ContentKind`, and `ContentColumn`, with no predicate over them. | Add `HasTextContent()`, the one place the backend conditional lives. |
| `internal/project/project.go:168-190` | `ReadItem` returns `ItemContent{Raw, Doc}`, with a working SQLite branch that synthesizes frontmatter plus body via `rawDocument`. | Reuse as-is. It already produces the bytes `fix.Apply` consumes. |
| `internal/project/project.go:222-238` | `UpdateItem(c, id, meta, body)` has a working SQLite branch; the filesystem branch returns `"filesystem item writes are handled by cmd"`. | Reuse the SQLite branch. The filesystem path keeps writing through `filesystem.Write`. |
| `internal/storage/collection/sqlite/collection.go:154-193` | `Update` sets the content column only when `body != nil`, and `configuredValues(c, nil, true)` returns an empty map. | Call it with `meta = nil` so the UPDATE touches the content column alone. |
| `internal/fix/fix.go` | `Apply` and `Canonical` are already backend-agnostic and do no IO. | Unchanged. |
| `cmd/testdata/snapshots/help/fix.txt` | Pins the current help text. | Regenerate with `-update` in the same change as the help edit. |

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Failing tests | Scaffold the spec's seven-item test checklist against the current code. |
| 2 | Capability predicate | Add `Collection.HasTextContent()` and its tests. |
| 3 | Route `fix` through the project | Change `fixOne`'s signature, replace the backend gate, persist per backend. |
| 4 | Docs and verification | Help text, snapshot, `AGENTS.md`, CLI reference, `make all`. |

Phase 2 lands before Phase 3 so the gate exists before the call site needs it.
Phase 1 comes first per `AGENTS.md`: new behavior arrives with a failing test.

## Phases

### Phase 1

**Goal.** Pin decision 6's behavior with failing tests.

1. Add capability tests for the collection predicate.

   **File:** `internal/storage/collection/parse_test.go` (new)

   Assert `HasTextContent()` is true for a filesystem collection, true for a
   SQLite collection with `ContentColumn` set, and false for a SQLite collection
   with attributes only. Package `collection_test`, per the external-test-package
   convention.

2. Add `fix` behavior tests for the SQLite content-column path.

   **File:** `cmd/fix_test.go`

   Drive the real Cobra root via `cmd.NewRootCmd()`. Scaffold a SQLite base into
   `t.TempDir()` following the fixture shape of `setupSQLiteRepo` in
   `cmd/sqlite_item_test.go`. Both files are `package cmd_test`, but `AGENTS.md`
   keeps helpers per-file, so prefer a local scaffold unless the duplication is
   more than a few lines. Cover:
   a `text_forbids` body fix applied to an item and persisted to the content
   column; `--check` reporting the change and exiting 1 without writing; a
   collection with no content column exiting with the attributes-only message
   rather than the old "not supported yet" text; and a bad fix template failing
   the run, matching the re-check contract in `applyTextFixes`.

3. Assert the filesystem path is untouched.

   **File:** `cmd/fix_test.go`

   Add a byte-for-byte regression test over a filesystem item that exercises
   both halves of `fix` (a text fix plus frontmatter reordering), so Phase 3's
   rewiring cannot silently change filesystem output.

### Phase 2

**Goal.** One predicate carries the backend conditional.

1. Add the capability predicate.

   **File:** `internal/storage/collection/parse.go`

   ```go
   // HasTextContent reports whether the collection exposes a text body that
   // fix can rewrite. Filesystem items are text by construction; a SQLite
   // collection has a body only when it maps a content column.
   func (c Collection) HasTextContent() bool {
       if c.StorageType == string(storage.SQLite) {
           return c.ContentColumn != ""
       }
       return true
   }
   ```

   Name the reason in the comment: this is the one place the conditional lives,
   so callers gate on the capability rather than on the backend.

   `parse.go` does not currently import `internal/storage`, so add it. This
   introduces no new package edge: `collection.go` in the same package already
   imports `storage`, and `storage` imports nothing from `collection`.

### Phase 3

**Goal.** `fix` reads and writes through the project, gated on capability.

1. Change `fixOne` to take an item and a project.

   **File:** `cmd/fix.go`

   New signature: `fixOne(p *project.Project, item project.Item, check bool) (bool, error)`.
   `item.Path` is a synthetic label for SQLite, so the item identity, not the
   path, is what the backend needs.

2. Replace the backend gate with the capability gate.

   **File:** `cmd/fix.go`

   Drop `if c.StorageType == string(storage.SQLite)`. Gate on
   `!item.Collection.HasTextContent()` and return
   `fmt.Errorf("fix requires a text content mapping; collection %q maps only attributes", c.Name)`.

3. Read through the project.

   **File:** `cmd/fix.go`

   Replace `os.ReadFile(path)` with `p.ReadItem(item)` and pass `content.Raw`
   to `fix.Apply`. This is the same bytes for the filesystem and the synthesized
   document for SQLite.

4. Persist per backend.

   **File:** `cmd/fix.go`

   Filesystem keeps `filesystem.Write(item.Path, result)`, an atomic replace.
   For a SQLite item, reparse the fixed bytes with `markdownbodytext.Parse` and
   call `p.UpdateItem(c, item.ID, nil, doc.Body)`. Pass `nil` metadata
   deliberately: frontmatter canonicalization is a no-op for a row (the YAML was
   synthesized from a map by `rawDocument` moments earlier), so only the body
   changed, and `configuredValues(c, nil, true)` yields no attribute values, so
   the UPDATE touches the content column alone.

5. Thread the project through the callers.

   **File:** `cmd/fix.go`

   `fixResolution` gains a `*project.Project` parameter. Both call sites already
   have one in scope: the root run builds `projectFor(plan.Root)`, and
   `fixDelegates` builds `projectFor(delegate.Config)` per delegate. Delegated
   nested configs inherit the new behavior with no further change.

6. Drop the now-unused imports.

   **File:** `cmd/fix.go`

   `os` and `storage` fall out once the direct read and the backend gate are
   gone. Run `gofmt -w .`.

### Phase 4

**Goal.** Docs match the behavior; the build is green.

1. Update the `fix` help text.

   **File:** `cmd/fix.go`

   Add to `Long`: `fix` operates on an item's text form, so a collection must
   expose a text body: every filesystem collection does, and a SQLite collection
   does when it maps a content column.

2. Regenerate the help snapshot.

   **File:** `cmd/testdata/snapshots/help/fix.txt`

   `go test ./cmd -run TestTopLevelHelpSnapshots -update` (the test lives in
   `cmd/help_snapshot_test.go`; the `-update` flag is defined in
   `cmd/snapshot_test.go`). Snapshots are the published CLI text contract, so
   the regeneration lands in the same commit as the help edit.

3. Drop the stale SQLite convention.

   **File:** `internal/storage/collection/sqlite/AGENTS.md`

   Remove "`fix` is not part of the first SQLite cut. `item add`, `item update`,
   and `item delete` own the write-path coverage for now." Keep the
   "Do not make check families backend-aware to compensate" line: this change is
   an instance of it.

4. Record the capability rule.

   **File:** `docs/content/deep-dives/domain-model/fix.md`

   State that `fix` is a text-form verb, that frontmatter canonicalization is a
   no-op for a row by construction, and that the gate is a capability rather
   than a backend name. This is the behavioral *why* a user can observe, so it
   belongs on the deep-dive page per `how-we-plan.md`.

5. Note the requirement in the CLI reference.

   **File:** `docs/content/reference/cli.md`

   One line under the exit-code section. The page is deliberately thin and
   defers to `--help`, so keep it to the cross-cutting fact.

6. Verify.

   Run `make all`, then `./bin/katalyst check` for the docs dogfood.

## Key Files

| File | Role |
|---|---|
| `internal/storage/collection/parse.go` | Gains `HasTextContent()`, the single backend conditional |
| `internal/storage/collection/parse_test.go` (new) | Capability predicate tests |
| `cmd/fix.go` | Signature change, capability gate, project-routed IO, help text |
| `cmd/fix_test.go` | SQLite content-column coverage plus the filesystem regression test |
| `cmd/testdata/snapshots/help/fix.txt` | Regenerated help contract |
| `internal/storage/collection/sqlite/AGENTS.md` | Drops the "not part of the first cut" convention |
| `docs/content/deep-dives/domain-model/fix.md` | The capability rule and why canonicalization is a row no-op |
| `docs/content/reference/cli.md` | One-line cross-cutting note |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Where the gate lives | `Collection.HasTextContent()` | Consolidates the conditional into one predicate next to the fields it reads, instead of a tenth `StorageType == sqlite` branch at the call site. Follows `sqlite/AGENTS.md`: gate on capability, not backend identity. |
| What `fixOne` takes | `project.Item`, not a path | `Item.Path` is an absolute path for the filesystem and a synthetic label for SQLite. Only the item identity is meaningful across both. |
| Metadata on the SQLite write | `nil` | Frontmatter canonicalization cannot change a row: the YAML is synthesized from a map by `rawDocument` immediately before `fix` sees it. Writing attribute columns back would be a write with no cause. |
| Whether `fix` joins `CollectionDefinition` | No | Read and write already run through `Project`'s backend switches; widening the seam is a larger change the spec leaves alone. This plan removes one conditional without adding an interface method. |
| Filesystem write path | Unchanged | `filesystem.Write` is an atomic replace. `project.UpdateItem`'s filesystem branch deliberately returns "handled by cmd", and this plan does not relitigate that split. |

## Documentation updates

Phase 4 carries all of them: the `fix` help text and its snapshot (steps 1 and
2), `internal/storage/collection/sqlite/AGENTS.md` (step 3),
`docs/content/deep-dives/domain-model/fix.md` (step 4), and
`docs/content/reference/cli.md` (step 5). The spec's other documentation
updates shipped with the prose commit.

## Out of Scope

- **Widening `CollectionDefinition` to cover read and write.** The six
  `case storage.SQLite:` branches with concrete type assertions in
  `internal/project/project.go` stay. This plan removes one conditional in
  `cmd/`, not the seam's read/write gap.
- **`ItemPath` (`project.go:82-85`) hardcoding the filesystem definition.** A
  real leak, unrelated to `fix`.
- **Collapsing `storage.Reference` into a plain `string`.** Cleanup the spec
  raised under scope B; it needs its own change.
- **Structured attribute writes.** `configuredValues` still rejects attributes
  with no column for `item add` and `item update`. `fix` passing `nil` metadata
  sidesteps this rather than fixing it.
- **`fix` inventing semantic values.** Unchanged and still refused; see
  `docs/content/deep-dives/domain-model/fix.md`.
