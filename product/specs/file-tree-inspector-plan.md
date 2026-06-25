# Plan - file tree inspector output
> Spec: [File tree inspector output](./file-tree-inspector-spec.md)
> **Status: planning.**

## Current State

- `internal/inspect/source.go` walks the raw source tree into `SourceView.files`.
  It records relative path, directory, and extension for every non-hidden file.
  `file_tree` reads this metadata and opens no files.
- `internal/inspect/inspectors_source.go` implements `FileTree.Inspect` by
  grouping refs per directory, turning each directory into feature tokens with
  `dirFeatures`, and passing those profiles to `summarize`.
- `internal/inspect/filemeta.go` provides the existing path primitive:
  extension histogram, basic casing buckets (`kebab`, `snake`, `other`),
  `with_spaces`, and max depth over a set of refs.
- `internal/inspect/summarize.go` clusters profiles into `classes` and
  `outliers`. `file_tree_content` and `document_shape` still use this model.
- `internal/inspect/render.go` renders every inspector through the generic
  Markdown key/value renderer. It has no `file_tree`-specific projection.
- `internal/inspect/source_test.go` proves `file_tree` applies to filesystem
  storage, opens no files, and produces directory profile classes.
- `cmd/inspect_test.go` snapshots the source-layer Markdown report and verifies
  JSON parseability, output-file parity, truncation, and `-v`.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Failing contracts | source inspector unit tests, CLI snapshots, JSON shape expectations |
| 2 | Filesystem summary model | path-derived summary helpers, thresholds, naming buckets, representative paths |
| 3 | Inspector payload | `FileTree.Inspect` returns structured map evidence, no file reads |
| 4 | Markdown rendering | `file_tree` projection, default caps, verbose expansion |
| 5 | Docs and verification | inspector docs, snapshots, focused test suite |

The order keeps the TDD contract visible. First pin the desired behavior, then
replace the evidence model, then teach the renderer how to present it.

## Phases

### Phase 1 - Failing contracts

**Goal:** the suite describes the new filesystem-map behavior before production
code changes.

1. **File:** `internal/inspect/source_test.go`.
   Replace the directory-class assertion in
   `TestFileTree_opensNothingAndProfilesDirs` with assertions for structured
   filesystem facts: total files, directory count, max depth, extension counts,
   and top-level regions. Keep the `ParseCount() == 0` assertion.
2. **File:** `internal/inspect/filemeta_test.go`.
   Add tests for the richer path classifiers: extension histogram, depth,
   directory counts, top-level region selection, naming bucket classification,
   and deterministic representative path selection.
3. **File:** `cmd/inspect_test.go`.
   Add CLI tests for:
   - tiny tree output includes a tree-like listing
   - medium tree output includes top-level regions
   - default output caps long summaries with a `pass -v` notice
   - verbose output expands the capped filesystem evidence
4. **File:** `cmd/testdata/snapshots/inspect/source-report.txt`.
   Update the source-layer snapshot after the failing test captures the new
   expected shape.

### Phase 2 - Filesystem summary model

**Goal:** path-derived facts are computed once, deterministically, without
opening files.

1. **File:** `internal/inspect/filetree.go` (new).
   Add a `fileTreeSummary` builder over `SourceView.files`. Return a
   `map[string]any` payload for `Evidence.Data` while keeping typed internal
   structs for construction if useful.
2. **File:** `internal/inspect/filetree.go` (new).
   Compute whole-tree facts: `file_count`, `dir_count`, `max_depth`,
   `extensions`, top-level regions, directory summaries, and tree entries for
   small trees.
3. **File:** `internal/inspect/filetree.go` (new).
   Add deterministic threshold helpers for small tree, major region, dominant
   extension, Markdown-heavy directory, and dominant naming bucket.
4. **File:** `internal/inspect/filetree.go` (new).
   Add representative path selection: prefer different top-level regions, sort
   lexicographically within each region, cap the returned list, and record hidden
   counts.
5. **File:** `internal/inspect/filemeta.go`.
   Either expand the existing casing helper or move the naming classifier into
   `filetree.go`. Support `kebab-case`, `snake_case`, `camelCase`, `PascalCase`,
   `title/spaces`, `lowercase`, `uppercase`, `numeric`, and `mixed/other`.

### Phase 3 - Inspector payload

**Goal:** `file_tree` emits filesystem-map evidence instead of profile clusters.

1. **File:** `internal/inspect/inspectors_source.go`.
   Change `FileTree.Inspect` to call the new summary builder and return
   `Evidence{Inspector: "file_tree", Scope: v.root, N: v.N(), Data: summary}`.
2. **File:** `internal/inspect/inspectors_source.go`.
   Leave `FileTreeContent` and `DocumentShape` on `summarize`. Do not change
   their payloads in this issue.
3. **File:** `internal/inspect/inspectors_source.go`.
   Remove `dirFeatures` only if no production code or tests still use it after
   `file_tree` moves off directory clustering.
4. **File:** `internal/inspect/source_test.go`.
   Keep the storage applicability and no-parse assertions green against the new
   payload.
5. **File:** `cmd/inspect_test.go`.
   Ensure `TestInspect_jsonEmitsSameEvidence` still passes and add direct JSON
   assertions for complete `file_tree` fields.

### Phase 4 - Markdown rendering

**Goal:** default output reads as a filesystem map, while verbose output shows
the supporting evidence.

1. **File:** `internal/inspect/render.go`.
   Add an inspector-specific branch for `file_tree` before the generic
   `dataLines` path. Keep the generic path for all other inspectors.
2. **File:** `internal/inspect/render.go`.
   Render default `file_tree` Markdown as:
   - overview sentence
   - tree listing for small trees
   - top-level regions for larger trees
   - top file types
   - naming summary when a dominant pattern exists
   - capped representative paths or exceptions
3. **File:** `internal/inspect/render.go`.
   Treat `maxLines <= 0` as expanded output for `file_tree`, matching today's
   `-v` behavior. Expanded output shows fuller region, extension, directory, and
   naming evidence.
4. **File:** `internal/inspect/render.go`.
   Make hidden-data notices explicit and actionable: `pass -v to show all`.
5. **File:** `internal/inspect/render_test.go`.
   Add renderer tests for default capping, verbose expansion, and preservation
   of generic rendering for other inspectors.

### Phase 5 - Docs and verification

**Goal:** docs explain the new boundary and the focused suite verifies behavior.

1. **File:** `docs/content/deep-dives/inspectors.md`.
   Update the raw-source inspector explanation: `file_tree` is the filesystem
   map, `file_tree_content` is content facts, and `document_shape` is document
   grouping.
2. **File:** `internal/inspect/doc.go`.
   Align package-level wording if it implies all source inspectors use profile
   clustering.
3. **File:** `cmd/testdata/snapshots/inspect/source-report.txt`.
   Regenerate the source report snapshot after the renderer lands.
4. **Validation:** run
   `go test ./internal/inspect ./cmd`.
   If renderer or CLI changes ripple farther, run the broader targeted suite
   used for inspect work.

## Key Files

| File | Role |
|---|---|
| `internal/inspect/source.go` | owns `SourceView.files`, the no-read path metadata that feeds `file_tree` |
| `internal/inspect/inspectors_source.go` | changes `FileTree.Inspect` from clustering to filesystem summary evidence |
| `internal/inspect/filetree.go` | new filesystem-map summary builder, thresholds, regions, naming buckets, and representatives |
| `internal/inspect/filemeta.go` | existing path metadata helper, either expanded or left as a compatibility primitive |
| `internal/inspect/render.go` | adds the `file_tree` Markdown projection and keeps generic rendering for other inspectors |
| `internal/inspect/source_test.go` | verifies no file reads and structured filesystem facts |
| `internal/inspect/filemeta_test.go` | verifies path-derived helper behavior |
| `internal/inspect/render_test.go` | verifies default and verbose `file_tree` rendering |
| `cmd/inspect_test.go` | verifies CLI output, JSON, snapshots, and verbosity behavior |
| `cmd/testdata/snapshots/inspect/source-report.txt` | pins the source-layer first-run report |
| `docs/content/deep-dives/inspectors.md` | documents the raw-source layering and evidence boundary |
| `internal/inspect/doc.go` | package-level architecture summary if wording needs alignment |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Inspector identity | keep `file_tree` | callers and docs already know this inspector name |
| Evidence model | structured filesystem summary | a map needs counts, regions, examples, and naming evidence, not profile clusters |
| Existing summarizer | keep for `file_tree_content` and `document_shape` | those inspectors still answer similarity questions in this issue's scope |
| Renderer | inspector-specific Markdown branch | generic key/value rendering cannot produce a readable tree or capped region summary |
| Verbosity | reuse `-v` / `--max-lines 0` as expanded output | avoids a new CLI flag while matching current inspect behavior |
| Claims | deterministic path-derived observations only | keeps `file_tree` from overlapping parse and document-shape inspectors |
| JSON | complete structured evidence | tools need the full deterministic payload even when Markdown is capped |

## Documentation updates

- **Phase 5, File:** `docs/content/deep-dives/inspectors.md`. Describe the
  raw-source layering as filesystem map, content facts, document grouping.
- **Phase 5, File:** `internal/inspect/doc.go`. Align package architecture
  wording if it says every raw-source inspector emits profile classes.
- **Phase 5, File:** `docs/content/reference/inspectors/`. Regenerate with
  `make docs-gen` only if the registry descriptor summary changes.

## Out of Scope

- `file_tree_content` output changes.
- `document_shape` output changes.
- Semantic labels such as blog, wiki, docs site, or collection.
- Schema recommendations.
- A new inspect verbosity flag.
- Collection-layer inspector rendering.
