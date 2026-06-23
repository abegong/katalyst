# Inspector layers — plan

> Spec: [Inspector layers — raw-source and collection inspectors over the storage seam](./inspector-layers-spec.md)
>
> **Status: implementing.** Phases 1–6 implemented; `make all` green and the
> inspector reference is generated. Graduation (deleting this spec/plan) waits on
> merge. Reorganizes `internal/inspect` from 11 `Corpus`-based inspectors into
> two layers (raw-source, collection) built from three reusable measurement
> primitives, with a shared profile-class summarizer and the first inspector
> parameter. Depends on the storage seam from
> [storage-layer-spec.md](./storage-layer-spec.md). Array/nested-object
> characterization is deferred to
> [#58](https://github.com/abegong/katalyst/issues/58).

## Current State

- **Inspectors are one interface over one input.** `internal/inspect/inspect.go`
  defines `Evidence{Inspector, Description, Scope, N, Data map[string]any}` and
  `Inspector interface { Name() string; Inspect(Corpus) Evidence }`.
  `internal/inspect/corpus.go` `Load(root)` walks **`*.md` only**, parses each
  via `frontmatter.Parse`, and stores `File{Rel, Doc, ParseErr}`; `meta(f)`
  reads a file's frontmatter map.
- **The 11 inspectors live in four family files.** `inspectors_structural.go`
  (`WalkParse`, `FrontmatterShape`), `inspectors_object.go` (the five
  `ObjectField*`), `inspectors_markdown.go` (`MarkdownHeadingShape`,
  `MarkdownSections`, `MarkdownCodeFences`), `inspectors_filesystem.go`
  (`FilesystemNaming`). Shared helpers in `helpers.go` (`eachValue`, `jsonType`,
  `toFloat`, `scalarString`, `sortedKeys`).
- **The registry is flat.** `internal/inspect/registry.go` holds
  `Descriptor{Name, Family, Slug, Title, Summary}`, `Families()` (structural /
  object / markdown / filesystem), `Descriptors()`, `All() []Inspector`,
  `ByName`, `Summary`. `registry_test.go` enforces `All()`↔`Descriptors()`
  parity and per-descriptor metadata.
- **Rendering is layer-agnostic already.** `render.go` `RenderMarkdown([]Evidence,
  maxLines)` groups by `Families()` order via `familyOf`; `RenderJSON([]Evidence)`
  emits the array. Both key off `Evidence`, so they survive the reorg.
- **The command builds a `Corpus` from a path.** `cmd/inspect.go` `newInspectCmd()`
  takes `inspect <path>`, `selectInspectors` resolves `--inspector` via
  `inspect.ByName`, runs each over `inspect.Load(path)`, renders. Flags: `--json`,
  `-o`, `--inspector`, `--max-lines`, `-v`. No project/config is loaded.
- **The noun command and docs read the registry.** `cmd/inspectors.go`
  (`inspectors list|show`, from #24) and `cmd/gendocs/main.go`
  (`inspectorsIndex` / `inspectorFamilyIndex` / `inspectorPage`) both render from
  `inspect.Descriptors()`/`Families()`. `make docs-gen-check` diffs
  `docs/content/reference/inspectors`.
- **The storage seam and project resolution exist.** `internal/storage` defines
  `CollectionDefinition` (`Collections`, `Items`, `Unmatched`, `Reference`),
  `Item{Collection, ID, Path}`, `Reference`, `StorageType`/`Known`, `Granularity`.
  `internal/project` `New(cfg)` exposes `Collection(name)`, `Items(c)`,
  `Resolve(selectors)`, and `ParseSelector`. `config.Load(wd)` returns
  `config.ErrNotFound` when no `.katalyst/` is present — the signal for "no
  project, run raw."
- **`internal/checks` substrate is unexported.** Field access and item iteration
  live inside `internal/checks`; a parallel branch is reworking them, so this
  plan touches the package as little as possible (spec Q3).

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Measurement primitives | `object_fields`, `markdown_body`, `file_metadata` as pure functions over generic inputs |
| 2 | Summarizer + parameters | profile-class dedup + outlier diff; `Params` tolerance (three mutually-exclusive forms, default `grouped`) |
| 3 | Collection layer | `CollectionView`, `CollectionInspector`, `object_fields` + `markdown_body` collection inspectors |
| 4 | Raw-source layer | `SourceView`, `SourceInspector`, `file_tree`, `file_tree_content`, `document_shape` |
| 5 | Registry + command cutover | `Layer` dimension, dual parity, retire the 11; rewrite `inspect` (layer from argument, `Params` flags), `inspectors list`, `gendocs` |
| 6 | Docs & graduation | deep-dives, glossary, regenerated reference; retire spec/plan |

Phases 1–4 are additive — new code beside the old, each gate green. Phase 5 is
the atomic cutover: it deletes the old inspectors and the `Corpus`-based
`Inspector` and rewrites every call site in one phase so the package never sits
half-migrated. Each phase is **tests-first internally**: write the failing test
sub-step, then the code that makes it pass.

## Phases

### Phase 1 — Measurement primitives

**Goal:** Three pure measurement engines, each a function of a generic input, so
both layers call the same code.

1. **File:** `internal/inspect/fields_test.go` *(new, failing first)* — over a
   fixed `[]map[string]any`, assert `objectFields` reports per-field presence
   over `n`, type histogram, cardinality, and common values; **string and
   numeric scalar value sets stay distinct** (a numeric `5` never merges with a
   string `"5"`); array/object values are typed but contribute no value set
   (per #58). `package inspect`, stdlib only.
2. **File:** `internal/inspect/fields.go` *(new)* — `objectFields(objs
   []map[string]any) map[string]any` building the data dictionary. Reuse
   `helpers.go` `jsonType`/`scalarString`/`toFloat`; key the value set by
   `(type, value)` so string/numeric stay separate. **Omit** the old
   `min/max` numeric-range and string-length columns (cull: dropped). This is
   the `object_fields` primitive; the five old `ObjectField*` collapse into it.
3. **File:** `internal/inspect/body_test.go` *(new, failing first)* — over a
   fixed `[][]byte` of markdown bodies, assert `markdownBody` reports the
   heading-shape facet (single-H1 / H1-matches-title / level-jump rates) and the
   recurring-sections facet (heading text → file count).
4. **File:** `internal/inspect/body.go` *(new)* — `markdownBody(bodies
   []mdInput) map[string]any` where `mdInput` carries body bytes + the `title`
   field (for H1-match). Move the `headings`/fence scan out of
   `inspectors_markdown.go` into here. Heading + sections facets only; **omit**
   the code-fences facet (cull: dropped). This is the `markdown_body` primitive.
5. **File:** `internal/inspect/filemeta_test.go` *(new, failing first)* — over a
   fixed `[]string` of relative paths, assert `fileMetadata` reports the
   extension histogram, naming-convention histogram (kebab/snake/other), space
   count, and max depth.
6. **File:** `internal/inspect/filemeta.go` *(new)* — `fileMetadata(refs
   []string) map[string]any`. Lift the casing/extension/depth logic from
   `inspectors_filesystem.go`. This is the `file_metadata` primitive; opens no
   files.
7. **Gate:** `go test ./internal/inspect/...` green; old inspectors untouched.

### Phase 2 — Summarizer + parameters

**Goal:** One "profile classes + outliers" summarizer and the tolerance
parameter both summarizing inspectors will take.

1. **File:** `internal/inspect/params_test.go` *(new, failing first)* — assert
   `ParseParams` accepts exactly one of `detail` / `similarity` / `maxClasses`,
   defaults to `detail=grouped` when none is set, and returns a usage error when
   two are set. Assert each maps to an internal tolerance.
2. **File:** `internal/inspect/params.go` *(new)* — `Params{ ... }` and
   `ParseParams(detail string, similarity float64, maxClasses int) (Params,
   error)`. The three forms are mutually exclusive (spec: usage error on >1);
   named levels `exact|grouped|coarse` map to internal proportions; `grouped`
   is the default. The collapse target is read by the summarizer.
3. **File:** `internal/inspect/summarize_test.go` *(new, failing first)* — over a
   list of `(label, profile)` pairs, assert the summarizer dedupes identical
   profiles into named classes, lists only outliers/diffs, and that **raising
   the tolerance reduces the class count**. Assert a 190-identical / 7 / 3-outlier
   input collapses to three classes + an outlier list.
4. **File:** `internal/inspect/summarize.go` *(new)* — `summarize(profiles
   []profile, p Params) map[string]any` producing `{classes, assignments,
   outliers}`. A profile is a comparable fingerprint; two profiles share a class
   when their similarity meets the `Params` tolerance. Render each member as a
   delta from its class representative so output is proportional to class count.
5. **Gate:** `go test ./internal/inspect/...` green.

### Phase 3 — Collection layer

**Goal:** A `CollectionInspector` over a resolved collection, addressed by domain
identity, running the primitives.

1. **File:** `internal/inspect/collection_test.go` *(new, failing first)* — build
   a `CollectionView` over a scaffolded collection (a `config.Collection` + a few
   `storage.Item`s in `t.TempDir()`), and assert `ObjectFields.Inspect` reports
   the dictionary over the items' frontmatter and `MarkdownBody.Inspect` reports
   the body facets. Assert items are addressed by `Item.ID`, never by raw path.
2. **File:** `internal/inspect/inspect.go` — add
   `CollectionInspector interface { Name() string; Inspect(CollectionView,
   Params) Evidence }`. Keep the old `Inspector` for now (removed in Phase 5).
3. **File:** `internal/inspect/collection.go` *(new)* — `CollectionView`: built
   from a `*project.Project` and a `config.Collection`. It resolves
   `[]storage.Item` via `project.Items(c)` and exposes, per item, its parsed
   frontmatter map and body — parsed with `frontmatter.Parse` through a **thin
   local adapter**, not by refactoring `internal/checks` (spec Q3: minimal
   touch). `Frontmatter() []map[string]any` and `Bodies() []mdInput` feed the
   primitives.
4. **File:** `internal/inspect/inspectors_collection.go` *(new)* — `ObjectFields`
   and `MarkdownBody` collection inspectors: each wraps its primitive over the
   `CollectionView`'s items and returns `Evidence` with `Scope` = collection
   name. Neither consumes `Params` (no summarizer).
5. **Gate:** `go test ./internal/inspect/...` green.

### Phase 4 — Raw-source layer

**Goal:** `SourceInspector`s over an unconfigured store, addressed by
backend-native reference, using primitives + summarizer.

1. **File:** `internal/inspect/source_test.go` *(new, failing first)* — build a
   `SourceView` over a scaffolded tree in `t.TempDir()` (mixed extensions, nested
   dirs, some markdown with frontmatter) and assert: `FileTree.Inspect` opens no
   files and reports a per-directory profile summarized into classes;
   `FileTreeContent.Inspect` parses markdown and reports parse/frontmatter/body
   stats per directory; `DocumentShape.Inspect` clusters files on the composite
   fingerprint (frontmatter key-set + body skeleton + file type/naming), not
   frontmatter alone. Assert `FileTree.AppliesTo(storage.Filesystem)` is true and
   a non-filesystem type is false.
2. **File:** `internal/inspect/inspect.go` — add `SourceInspector interface {
   Name() string; AppliesTo(storage.StorageType) bool; Inspect(SourceView,
   Params) Evidence }`.
3. **File:** `internal/inspect/source.go` *(new)* — `SourceView`: walks a
   filesystem path, collecting **all** files (not just `*.md`) with their
   relative `Reference`s grouped by directory, and lazily parses markdown for the
   deep/shape inspectors. Replaces `Corpus`/`Load` as the raw-layer input; keep
   `Load` only until Phase 5 removes its last caller. (Generalizing the walk to a
   storage-layer "enumerate units" API is out of scope — filesystem-only for now,
   gated by `AppliesTo`.)
4. **File:** `internal/inspect/inspectors_source.go` *(new)* — `FileTree`
   (`file_metadata` per directory → `summarize`), `FileTreeContent` (parse +
   `object_fields`/`markdown_body` per directory → `summarize`), and
   `DocumentShape` (composite fingerprint per file → `summarize`). All three
   consume `Params` for the collapse tolerance; all `AppliesTo(Filesystem)`.
5. **Gate:** `go test ./internal/inspect/...` green.

### Phase 5 — Registry + command cutover

**Goal:** One registry carrying both layers; the 11 old inspectors gone; the
command, noun command, and docs generator rewritten to the two-layer model. The
atomic flip — ends green.

1. **File:** `internal/inspect/registry_test.go` — replace the single-parity test
   with **two**: every `SourceInspectors()` entry has a `Descriptor` with
   `Layer == "source"` and vice versa; same for `CollectionInspectors()` /
   `Layer == "collection"`. Keep slug-uniqueness within a layer.
2. **File:** `internal/inspect/registry.go` — add `Layer string` to `Descriptor`
   (`"source"` | `"collection"`). Replace `All()`/`ByName()` with
   `SourceInspectors() []SourceInspector`, `CollectionInspectors()
   []CollectionInspector`, and layer-scoped name lookups. Rewrite `Descriptors()`
   for the five new inspectors (`file_tree`, `file_tree_content`,
   `document_shape`, `object_fields`, `markdown_body`) with their `Layer`, family,
   slug, title, summary.
3. **File:** delete `inspectors_structural.go`, `inspectors_object.go`,
   `inspectors_markdown.go`, `inspectors_filesystem.go`, `corpus.go`, and the old
   `Inspector` type and their tests. Fold any still-needed helper (`meta`,
   `headings`) into the primitives/views that use them.
4. **File:** `cmd/inspect_test.go` — assert layer selection: `inspect <rawpath>`
   (no `.katalyst/`) runs the source layer; inside a project, `inspect
   <collection-selector>` runs the collection layer; a path that isn't a
   configured collection runs source. Assert `--detail`/`--similarity`/
   `--max-classes` are mutually exclusive (exit 2) and default to `grouped`.
5. **File:** `cmd/inspect.go` — rewrite the run body. Resolve the argument:
   `config.Load(cwd)` → if `ErrNotFound`, or the argument doesn't resolve via
   `project.ParseSelector`/`Collection`, treat it as a path and run
   `SourceInspectors()` over a `SourceView`; otherwise build a `*project.Project`
   and run `CollectionInspectors()` over a `CollectionView`. Add `--detail`,
   `--similarity`, `--max-classes` → `inspect.ParseParams` → `Params`. Keep
   `--json`, `-o`, `--inspector` (now layer-scoped), `--max-lines`, `-v`. Render
   via existing `RenderMarkdown`/`RenderJSON`.
6. **File:** `cmd/inspectors.go` — group `list` output by `Layer` first, then
   family; `show` prints the inspector's layer. Drive off the new registry
   accessors.
7. **File:** `cmd/gendocs/main.go` — render the inspectors reference grouped by
   layer (two top-level sections), per-inspector pages under
   `reference/inspectors/<layer>/<slug>.md`. Run `make docs-gen`; the existing
   `docs-gen-check` (already covering `reference/inspectors`) guards drift.
8. **File:** `cmd/root_test.go` — refresh the help-fixture snapshot if any
   `Short` changed (it shouldn't; `inspectors`/`inspect` keep their summaries).
9. **Gate:** `make all` and `make docs-gen` clean.

### Phase 6 — Docs & graduation

**Goal:** Reconcile durable docs and retire the spec/plan.

1. **File:** `docs/content/deep-dives/core-concepts.md` — describe the two
   inspector layers (raw-source over a `StorageType`; collection over a
   configured collection, probing through checks) and the measurement primitives.
2. **File:** `docs/content/deep-dives/domain-model.md` — add the layer split and
   the `inspect` layer-selection flow; absorb the locked decisions (referencing
   machinery differs by layer; primitives; tolerance default `grouped`) into
   prose, per `how-we-plan.md` (no separate decisions log).
3. **File:** `docs/content/reference/glossary.md` — add *raw-source layer*,
   *collection layer*, *measurement primitive*, *profile class*; update
   *fingerprint* for the composite shape.
4. **File:** `docs/content/reference/commands.md`, `README.md` — document the two
   `inspect` modes and the `--detail` parameter.
5. **File:** `cmd/AGENTS.md` — note that `inspect` infers its layer from the
   argument (selector resolves → collection; else path → raw), an exception worth
   recording beside the verb/noun grammar.
6. **File:** `product/specs/inspect-spec.md` — confirm the supersession banner
   still points here; no design change.
7. **Graduation:** set this plan and the spec Status to **done**, run the
   `docs/contributing/how-we-plan.md` graduation checklist, delete spec + plan.
   Their locked rationale lands in `internal/inspect`'s package docs (a `doc.go`)
   and the deep-dives above.
8. **Gate:** `make all` and `make docs-gen` clean; no stale references to the old
   inspector names.

## Key Files

| File | Role |
|---|---|
| `internal/inspect/fields.go` | `object_fields` primitive — data dictionary over object maps (new) |
| `internal/inspect/body.go` | `markdown_body` primitive — heading + section facets (new) |
| `internal/inspect/filemeta.go` | `file_metadata` primitive — path-level attributes (new) |
| `internal/inspect/params.go` | `Params` + `ParseParams`; tolerance forms, default `grouped` (new) |
| `internal/inspect/summarize.go` | profile-class dedup + outlier diff (new) |
| `internal/inspect/collection.go` | `CollectionView` over `project.Project` + `config.Collection` (new) |
| `internal/inspect/inspectors_collection.go` | `ObjectFields`, `MarkdownBody` collection inspectors (new) |
| `internal/inspect/source.go` | `SourceView` — filesystem tree walk, all files (new) |
| `internal/inspect/inspectors_source.go` | `FileTree`, `FileTreeContent`, `DocumentShape` (new) |
| `internal/inspect/inspect.go` | `SourceInspector` / `CollectionInspector` / `Evidence` (edited) |
| `internal/inspect/registry.go` | `Descriptor.Layer`, dual accessors, five descriptors (edited) |
| `internal/inspect/registry_test.go` | dual parity (edited) |
| `internal/inspect/{inspectors_*,corpus}.go` | the 11 old inspectors + `Corpus` (deleted) |
| `cmd/inspect.go` | layer selection from argument; `Params` flags (edited) |
| `cmd/inspectors.go` | `list`/`show` grouped by layer (edited) |
| `cmd/gendocs/main.go` | inspectors reference grouped by layer (edited) |
| `docs/.../core-concepts.md`, `domain-model.md`, `glossary.md`, `commands.md`, `README.md`, `cmd/AGENTS.md` | graduation targets (edited) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Two interfaces | `SourceInspector` and `CollectionInspector`, not one `Inspector` | The layers reference data through different machinery (`Reference` vs `Collection`+`Item.ID`); one input type would re-hardcode the filesystem assumption the storage seam removes (spec) |
| Reuse via primitives | `object_fields` / `markdown_body` / `file_metadata` as pure functions both layers call | Five field inspectors are columns of one table; three markdown inspectors are facets of one walk; primitives remove the duplication |
| Substrate, minimal touch | `CollectionView` parses items via a thin local adapter over `frontmatter.Parse`, not by refactoring `internal/checks` | Spec Q3: a parallel branch is reworking the substrate; depend on the smallest surface and defer a shared package |
| Shared summarizer | One profile-class dedup + outlier diff for `file_tree*` and `document_shape` | Output proportional to distinct profiles, not directories; same mechanism clusters files→collections and dirs→layout |
| Tolerance parameter | Three mutually-exclusive forms; default `--detail grouped`; >1 is exit 2 | Spec Q2; mirrors the `--try`/`--schema` exclusion. First inspector parameter, superseding inspect-spec's parameterless-v1 |
| Layer from argument | Selector resolves against config → collection; else path → raw; no project → raw | Spec Q1: keeps onboarding (`inspect ./wiki`) flag-free; the niche foreclosed case gets an override only if needed |
| `object_fields` scope | Scalars only, string/numeric kept distinct; range/length columns dropped | Spec Q4 + #25 cull; arrays/nested deferred to #58 |
| Source walk is filesystem-only | `SourceView` walks the path directly, gated by `AppliesTo(Filesystem)` | No storage "enumerate units" API yet; `AppliesTo` scaffolds the seam without blocking on a second backend |

## Documentation updates

- **Generated inspector reference** — `cmd/gendocs/main.go` regrouped by layer
  (Phase 5); regenerate with `make docs-gen`, guarded by `docs-gen-check`.
- **Deep dives** — `core-concepts.md` and `domain-model.md` gain the two-layer
  model and primitives (Phase 6).
- **Glossary** — new terms (raw-source layer, collection layer, measurement
  primitive, profile class), updated *fingerprint* (Phase 6).
- **User docs** — `commands.md` and `README.md` document the two `inspect` modes
  and `--detail` (Phase 6).
- **Developer docs** — `cmd/AGENTS.md` records the layer-from-argument exception
  (Phase 6); `internal/inspect` gains a `doc.go` absorbing the spec rationale at
  graduation.

## Out of Scope

- **Array and nested-object characterization in `object_fields`.** Scalars only
  for this cut; deepening is [#58](https://github.com/abegong/katalyst/issues/58).
- **Deep check-substrate extraction.** No shared `internal/probe` package and no
  refactor of `internal/checks` — coordinate with the parallel substrate branch
  (spec Q3). A thin local adapter suffices for now.
- **Non-filesystem `StorageType`s.** The raw walk is filesystem-only; `AppliesTo`
  scaffolds the seam but `sqlite`/`mongodb` walks are future work.
- **A storage-layer "enumerate units" API.** `SourceView` walks the path
  directly; generalizing raw enumeration into `internal/storage` is deferred.
- **The final cull drop-list beyond the spec lean.** This plan drops the
  numeric-range/string-length columns and the code-fences facet (spec's #25
  lean); revisiting which optional columns/facets ship stays a maintainer call.
- **Agent orchestration.** Forming hypotheses, naming collections, drawing fuzzy
  cluster boundaries, and writing `.katalyst/` files remain the harness's job
  (carried from inspect-spec).

## Test checklist

The spec's [Test checklist](./inspector-layers-spec.md) is the contract. The
pending tests scaffold across phases: primitives (1), summarizer + params (2),
collection layer (3), raw-source layer (4), and the registry/command cutover (5).
