# Spec - file content shape inspector

> **Status: planning.** Tracks [#111](https://github.com/abegong/katalyst/issues/111).

## Overview

`file_content_shape` summarizes shared content structure across a selected set
of files. It is the second raw-source inspection level: `file_tree` maps the
store without opening files, then `file_content_shape` opens selected files and
reports the text, tabular, and tree views Katalyst extracts from them.

## Value

Automatic clustering hides the question the reader is asking: "Does this slice
of files behave like a coherent set of items?" `file_content_shape` lets a human
or agent propose a directory, glob, extension filter, or path query, then inspect
the shared evidence for that candidate set.

The loop is explicit:

1. Run `katalyst inspect .` to see the store map.
2. Run `katalyst inspect . --inspector file_content_shape --select ...` with a
   candidate selection such as `content/books/*.md`, `data/*.csv`, or
   `path under docs/reference`.
3. Read common structure and variation.
4. Refine the selection or draft a collection from the evidence.

## Current State

- `cmd/inspect.go` accepts one `<path-or-collection>` argument. A configured
  collection name runs collection inspectors; a filesystem path runs every
  raw-source inspector over the whole directory.
- `cmd/inspect.go` already supports inspector narrowing through the repeatable
  `--inspector` flag. That is the natural user-facing hook for
  `file_content_shape`; adding an inspector-specific subcommand would create a
  second invocation grammar for the same registry.
- `internal/inspect/params.go` carries inspector parameters (`--detail`,
  `--similarity`, `--max-classes`) through `inspect.Params`. Inspectors that do
  not use a parameter ignore it. Selection can follow that pattern if validation
  keeps it scoped to `file_content_shape`.
- `internal/inspect/source.go` builds `SourceView.files` from path metadata and
  lazily parses only `.md` files through `SourceView.markdown`.
- `internal/inspect/inspectors_source.go` implements `FileTreeContent.Inspect`
  by grouping parsed Markdown documents per directory, reducing each directory
  to feature tokens (`parsed`, `frontmatter`, `fmkey:<name>`), and passing them
  to `summarize`.
- `internal/inspect/registry.go` describes `file_tree_content` as Markdown-only:
  "Parse markdown and profile each directory's content shape."
- `internal/inspect/render.go` renders every inspector through the same generic
  key/value Markdown renderer. The output exposes `classes` and `outliers`
  instead of a readable report.
- `docs/content/deep-dives/inspectors.md` describes `file_tree` and
  `document_shape` as summarizing inspectors that collapse profiles into classes.

That model is too narrow. Markdown is one content source, not the boundary of the
second raw-source inspector. Profiling the entire directory by default also mixes
unrelated files and makes the result less useful.

## Design

### Raw-source levels

The raw-source layer has two primary inspection levels:

1. **Store map.** `file_tree` opens no files. It reports paths, directories,
   extensions, naming, depth, and density.
2. **Content shape.** `file_content_shape` opens a selected set of files and
   performs light parsing. It reports shared content views, common structure,
   variation, and read or parse issues.

The earlier `document_shape` clustering idea is deferred from the primary path.
Katalyst can add suggestion and clustering features later, but the core CLI
should first let the reader test explicit selections.

### Command surface

`file_content_shape` is a regular source inspector registered in
`internal/inspect/registry.go`, not a Cobra subcommand. The existing `inspect`
shape stays intact: the positional argument selects the root, `--inspector`
selects the inspector, and a new `--select` parameter narrows the file set that
`file_content_shape` profiles.

```sh
katalyst inspect <path> --inspector file_content_shape --select <selection>
```

Examples:

```sh
katalyst inspect . --inspector file_content_shape --select 'content/books/*.md'
katalyst inspect . --inspector file_content_shape --select 'ext = ".csv"'
katalyst inspect . --inspector file_content_shape --select 'path under "docs/reference"'
```

The first cut should treat `--select` as a parameter owned by
`file_content_shape`: it is valid only when exactly one source inspector is
selected and that inspector is `file_content_shape`. Passing `--select` with a
collection-layer target, with no `--inspector`, with multiple `--inspector`
flags, or with another inspector is a usage error. This keeps the existing
inspect pipeline predictable and avoids making every inspector define selection
semantics.

The old `file_tree_content` name should not remain the long-term user-facing
name. For the first cut, replace it in the registry with `file_content_shape`
rather than shipping two public names. If callers need compatibility later, add
an alias deliberately with tests that prove JSON and Markdown render the
canonical `file_content_shape` name.

### Selection

A selection is the set of source files the profile uses as its denominator. The
default selection is every non-hidden file under the inspected directory. This
fallback is useful for discovery, but the intended workflow is to pass a narrower
selection.

The first cut supports:

- directory selection: `content/books/`
- glob selection: `content/books/*.md`
- path-level query: `ext = ".csv"` or `path under "docs/reference"`

Metadata and content predicates are out of scope for the first cut because they
require content reads before selection.

The profile output always prints the resolved selector label, file count,
directory count, extension mix, and skipped/unsupported count before reporting
content facts.

Selection is resolved after the `SourceView` walk and before any content parser
runs, so it is path-derived and opens no files. The resolved selection can be
stored on `inspect.Params` as a small value object (for example `Selection{
Label, Mode, Pattern}`) and applied by `FileContentShape.Inspect`. Other
inspectors should not see selected subsets in the first cut.

### Content views

The inspector extracts zero or more views from each selected file. A view is an
analysis result, not the file's identity. The same file can produce multiple
views.

View families:

- **Text.** Sequential text, paragraphs, lines, headings, section-like markers,
  or visible text.
- **Tabular.** Rows and columns, from CSV first. Markdown tables, HTML tables,
  and JSON arrays of objects are later parsers.
- **Tree.** Nested structure, from Markdown AST/frontmatter and JSON first.
  YAML, TOML, HTML DOM, XML, and code ASTs are later parsers.

Each view record carries:

- source path
- family: `text`, `tabular`, or `tree`
- parser id, such as `markdown-body`, `csv`, or `json-tree`
- status: `extracted`, `partial`, `failed`, or `unsupported`
- compact facets for that family

### First-cut parser scope

Start with three parser families:

- Markdown text and tree views from the existing Markdown parser.
- CSV tabular views through the standard library.
- JSON tree views through the standard library.

If any parser, facet, or selector form turns into a large dependency or a long
implementation detour, move it to a follow-up issue and keep the first cut
focused. The first cut is successful when the command surface, output shape, and
Markdown/JSON/CSV views work end to end.

### Output shape

Default Markdown should read like a short report, not an inventory dump. It
should state what was selected, whether the selection looks coherent, what
evidence supports that read, and which files differ.

For a coherent Markdown selection:

```markdown
### file_content_shape

selector: content/books/*.md

24 files selected from `content/books/`. All 24 are readable Markdown files.
Katalyst extracted text and tree views from every file, plus tabular views from
2 files.

This selection is coherent:

- 24/24 files have frontmatter keys: title, status.
- 22/24 have a Review section.
- 24/24 filenames are kebab-case Markdown files.

Variation:
- tags appears in 18/24 files.
- rating appears in 11/24 files.
- 2 files lack Review: books/foo.md, books/bar.md.

Text:
- H1 in 24/24 files.
- H2 sections in 22/24 files.

Tree:
- frontmatter object in 24/24 files.
- common keys: title 24/24, status 24/24, tags 18/24.

Tabular:
- Markdown tables in 2/24 files.

Read/parse issues:
- none
```

For a CSV selection:

```markdown
### file_content_shape

selector: data/*.csv

12 files selected from `data/`. All 12 parse as CSV.

This selection is coherent:

- 12/12 files have columns: id, title, status.
- row count ranges from 8 to 118, median 42.
- 10/12 files include an optional notes column.

Tabular:
- common columns: id 12/12, title 12/12, status 12/12.
- optional columns: notes 10/12, tags 4/12.

Read/parse issues:
- none
```

For a JSON selection:

```markdown
### file_content_shape

selector: ext = ".json"

9 files selected across 3 directories. All 9 parse as JSON tree views.

This selection is partly coherent:

- 7/9 files are top-level objects.
- 7/9 files share keys: id, title, status.
- 2/9 files are arrays and should be profiled separately.

Tree:
- top-level object: 7/9 files.
- top-level array: 2/9 files.
- common object keys: id 7/7, title 7/7, status 7/7.

Variation:
- 2 array files: fixtures/books.json, fixtures/movies.json.
```

For a broad selection:

```markdown
### file_content_shape

selector: docs/**

142 files selected across 18 directories. Katalyst extracted content views from
106 files and skipped 36 assets or unsupported files.

This selection is too mixed to profile as one item set:

- No content view appears across more than 42% of selected files.
- Extensions are mixed: .md 31, .png 18, .json 5, .css 4.

Variation:
- 18 files are assets or unsupported.
- 5 JSON files share object keys, but they are only 8% of the selection.
```

Verbose output expands examples, per-directory breakdowns, and full frequency
tables. JSON remains complete and parseable.

### Common structure and variation

The report's most important claim is whether the selection behaves like a
coherent item set.

Common structure reports high-frequency facts across the selected files:

- content view families present in most files
- common frontmatter or object keys
- common columns
- common headings or section labels
- common top-level tree shape

Variation reports meaningful differences:

- optional keys or columns
- missing sections
- parse failures
- unsupported files inside the selection
- subsets that look coherent but represent a small fraction of the selection

These sections use counts and denominators. They do not recommend a schema or
collection.

### Relationship to `document_shape`

`document_shape` should not be the primary automatic clustering path for this
workflow. A future clustering or suggestion command can propose likely
selections, but `file_content_shape` should profile an explicit selection and
report evidence.

This keeps Katalyst's primary raw-source flow deterministic and explainable:

1. map the store
2. choose or query a slice
3. profile the slice

## Backoff rule

This spec sets the direction, not a mandate to build every parser and query form
at once. During planning or implementation, any part that becomes expensive,
unclear, or dependency-heavy should move to a follow-up issue. Preserve the
content-shape model and ship the smallest coherent version first.

Examples of acceptable deferrals:

- HTML DOM parsing
- code AST parsing
- Markdown table extraction
- JSON array-to-table inference
- YAML parsing
- a full query language
- automatic selection suggestions

## Open Questions

_None._ The first cut is intentionally small: regular inspector registry
addition, `--select` as a scoped inspect parameter, directory/glob/path-query
selections, Markdown/JSON/CSV parsers, and the `file_content_shape` user-facing
name.

## Documentation updates

- `docs/content/deep-dives/inspectors.md`: update the raw-source model from
  profile clustering to store map plus content shape. Note that clustering and
  suggestions are follow-up features, not the primary flow.
- `internal/inspect/doc.go`: align the package summary if it names Markdown-only
  or clustering-specific behavior.
- `docs/content/reference/inspectors/`: regenerate with `make docs-gen` if the
  registry descriptor changes from `file_tree_content` to `file_content_shape`.
- `docs/content/reference/cli.md`: document
  `katalyst inspect <path> --inspector file_content_shape --select ...` and the
  supported selection syntax.
- `docs/content/reference/glossary.md`: add `content view` and
  `file_content_shape` if those terms survive implementation.

## Test checklist

- Selection summary reports selector label, file count, directory count,
  extension mix, readable count, and unsupported/skipped count.
- `--select` is accepted only with
  `--inspector file_content_shape` on a source-layer target; invalid combinations
  return usage errors.
- Markdown files produce text and tree views without making Markdown the
  top-level output category.
- JSON files produce tree-view common keys.
- CSV files produce tabular column and row-count summaries.
- Broad mixed selections report weak commonality and skipped unsupported files.
- Narrow coherent selections report common structure and variation with
  denominators.
- Parse failures are visible and grounded in paths.
- JSON output remains complete and parseable.
- Default Markdown is capped; verbose output expands examples and frequency
  tables.
