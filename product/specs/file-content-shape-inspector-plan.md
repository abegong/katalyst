# Plan - file content shape inspector

> Spec: [File content shape inspector](./file-content-shape-inspector-spec.md)
> **Status: executed.**

## Current State

- `katalyst inspect <path-or-collection>` infers one layer from the target. A
  configured collection runs collection inspectors; a filesystem directory runs
  source inspectors. `--inspector` already narrows by registry name.
- `internal/inspect/params.go` carries source and collection inspector
  parameters. After retiring grouping, it carries only selected-file state.
- `internal/inspect/source.go` walks non-hidden filesystem paths into
  `SourceView.files`, and content inspectors read selected files explicitly.
- `file_tree` now owns the path-only store map. The old content inspector parsed
  Markdown per directory and rendered generic `classes` / `outliers`.
- `document_shape` and the summarizer-backed grouping path were retired after
  review; explicit selections are the content-shape workflow.
- `internal/inspect/render.go` has a custom Markdown renderer for `file_tree`;
  all other inspectors use generic key/value rendering.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | CLI and selection contracts | `--select` flag, params plumbing, usage errors, selection resolver tests |
| 2 | Content-shape model | selected file reads, Markdown/CSV/JSON parsers, evidence payload |
| 3 | Registry transition | replace `file_tree_content` with `file_content_shape`, docs-generation snapshots |
| 4 | Markdown rendering | inspector-specific report, default caps, verbose expansion |
| 5 | Docs and verification | CLI/reference/deep-dive updates, full test suite |

The order keeps the user-facing contract stable first: `inspect` remains one
command, `file_content_shape` remains a normal source inspector, and `--select`
is scoped to that inspector before any parser behavior depends on it.

## Phases

### Phase 1 - CLI and selection contracts

**Goal:** `--select` reaches source inspectors through `inspect.Params`, and
invalid combinations fail before files are opened.

1. **File:** `internal/inspect/params.go`.
   Add a `Selection` value on `Params`, with `Label`, `Mode`, and `Pattern`.
   Keep selection validation in `cmd/inspect.go`.
2. **File:** `cmd/inspect.go`.
   Add `--select string`. It is valid only when the target resolves to the
   source layer, exactly one `--inspector` is supplied, and that inspector is
   `file_content_shape`. Passing `--select` with a collection target, no
   inspector, multiple inspectors, or another inspector returns a usage error.
3. **File:** `internal/inspect/selection.go` (new).
   Resolve selection after the `SourceView` walk and before content reads.
   Support:
   - default all files when no selection is set
   - directory prefix (`content/books/`)
   - doublestar-style glob or `path.Match`-compatible glob
   - path query `ext = ".csv"`
   - path query `path under "docs/reference"`
4. **File:** `internal/inspect/source.go`.
   Add small helpers for selected files and reading a relative file path. Keep
   the selection path-derived; no content predicate support.
5. **Tests:** `cmd/inspect_test.go`, `internal/inspect/params_test.go`, and a
   new `internal/inspect/selection_test.go`.
   Pin valid and invalid `--select` combinations plus deterministic selected
   path ordering.

### Phase 2 - Content-shape model

**Goal:** selected files produce complete JSON evidence for Markdown, CSV, JSON,
and unsupported/read-failure cases.

1. **File:** `internal/inspect/filecontentshape.go` (new).
   Add `FileContentShape.Inspect`, a typed summary builder, and map conversion
   for JSON evidence.
2. **File:** `internal/inspect/filecontentshape.go`.
   Compute selection summary: selector label, selected file count, directory
   count, extension histogram, readable count, unsupported count, parse-failure
   count, and skipped paths.
3. **File:** `internal/inspect/filecontentshape.go`.
   Markdown parser: use the existing document parser to produce text and tree
   view facets: frontmatter key frequencies, H1 count, H2+ section frequencies,
   and parse issues.
4. **File:** `internal/inspect/filecontentshape.go`.
   CSV parser: use `encoding/csv` to report common column names, optional
   columns, row-count min/median/max, and parse issues.
5. **File:** `internal/inspect/filecontentshape.go`.
   JSON parser: use `encoding/json` to report top-level shape frequencies and
   common keys for top-level objects.
6. **File:** `internal/inspect/filecontentshape.go`.
   Add a small coherence classifier (`coherent`, `partly_coherent`, `mixed`)
   based on high-frequency view facts. Keep it descriptive and count-backed.
7. **Tests:** `internal/inspect/filecontentshape_test.go`.
   Cover coherent Markdown, coherent CSV, partly coherent JSON, broad mixed
   selection, unsupported files, and parse/read failures.

### Phase 3 - Registry transition

**Goal:** `file_content_shape` is the public source inspector name; the old
Markdown-only inspector is removed from the default public registry.

1. **File:** `internal/inspect/inspectors_source.go`.
   Replace `FileTreeContent` in `SourceInspectors()` with
   `FileContentShape`. Remove old per-directory Markdown clustering helpers if
   no production code or tests still need them.
2. **File:** `internal/inspect/registry.go`.
   Replace the descriptor `file_tree_content` with `file_content_shape`,
   updating slug, title, family, and summary.
3. **Files:** `cmd/testdata/snapshots/inspectors/*`,
   `docs/content/reference/inspectors/source/*`.
   Regenerate inspector reference docs with `make docs-gen`; update snapshots
   for list/show output.
4. **Tests:** `internal/inspect/registry_test.go` and `cmd/inspectors_test.go`.
   Keep registry parity green and assert `file_content_shape` appears in source
   inspector listings.

### Phase 4 - Markdown rendering

**Goal:** default Markdown reads as a short content-shape report while JSON stays
complete.

1. **File:** `internal/inspect/render.go`.
   Add a `file_content_shape` renderer branch before generic `dataLines`.
2. **File:** `internal/inspect/render.go`.
   Render default Markdown sections:
   - selector and selection summary
   - coherence statement
   - common structure
   - variation
   - text, tree, and tabular summaries when present
   - read/parse issues
3. **File:** `internal/inspect/render.go`.
   Reuse the quiet inspect-report styling from `file_tree`: section dividers,
   lowercase labels, aligned key/value rows, and tabular headers.
4. **File:** `internal/inspect/render.go`.
   Treat `maxLines <= 0` as expanded output. Verbose output includes more
   examples, per-extension/per-directory breakdowns, and full frequency tables.
5. **Tests:** `internal/inspect/render_test.go` and
   `cmd/testdata/snapshots/inspect/source-report.txt`.
   Pin Markdown for Markdown/CSV/JSON examples and ensure generic rendering
   still handles other inspectors.

### Phase 5 - Docs and verification

**Goal:** documentation describes the two-step raw-source flow and the focused
suite verifies behavior.

1. **File:** `docs/content/deep-dives/inspectors.md`.
   Describe raw-source inspection as store map (`file_tree`) plus selected
   content shape (`file_content_shape`). Note clustering/suggestions as future
   work.
2. **File:** `internal/inspect/doc.go`.
   Align package-level wording if it names Markdown-only or clustering-specific
   behavior.
3. **File:** `docs/content/reference/cli.md`.
   Document `--select`, its valid pairing with
   `--inspector file_content_shape`, and supported selection syntax.
4. **File:** `docs/content/reference/glossary.md`.
   Add `content view` only if the implementation keeps that term visible.
5. **Validation:** run `go test ./internal/inspect ./cmd`, `make docs-gen`, and
   `go test ./...`.

## Key Files

| File | Role |
|---|---|
| `cmd/inspect.go` | adds and validates `--select`; passes selection through params |
| `internal/inspect/params.go` | carries selected-file state |
| `internal/inspect/selection.go` | path-derived selection resolver |
| `internal/inspect/source.go` | selected file helpers and relative file reads |
| `internal/inspect/filecontentshape.go` | summary builder and Markdown/CSV/JSON parsers |
| `internal/inspect/registry.go` | public inspector descriptor transition |
| `internal/inspect/render.go` | `file_content_shape` Markdown projection |
| `cmd/testdata/snapshots/inspect/` | CLI output contracts |
| `docs/content/deep-dives/inspectors.md` | raw-source model docs |
| `docs/content/reference/cli.md` | `--select` reference |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Command shape | regular source inspector plus `--select` | preserves the existing inspect grammar and registry flow |
| Selection scope | valid only for `file_content_shape` first cut | avoids implicit subset semantics for unrelated inspectors |
| Public name | replace `file_tree_content` with `file_content_shape` | avoids two names and keeps reports canonical |
| Parser scope | Markdown, CSV, JSON only | enough to prove text/tabular/tree views without dependency drift |
| Selection timing | path-only before content reads | deterministic, cheap, and matches the spec boundary |
| Output model | complete JSON, capped Markdown | machines get full evidence; humans get a short report |
| Clustering | retired | explicit selections are the primary workflow |

## Documentation updates

- `docs/content/deep-dives/inspectors.md`: store map plus content shape model.
- `docs/content/reference/inspectors/`: regenerate after descriptor rename.
- `docs/content/reference/cli.md`: `--select` syntax and constraints.
- `docs/content/reference/glossary.md`: add only surviving user-facing terms.

## Out of Scope

- HTML, XML, YAML, TOML, code AST, Markdown table, or JSON array-to-table
  parsers.
- Content predicates in selection syntax.
- Automatic selection suggestions.
- Automatic grouping or clustering.
- Alias compatibility for `file_tree_content`; add deliberately later if needed.
