# Spec - file tree inspector output

> **Status: planning.** Tracks [#110](https://github.com/abegong/katalyst/issues/110).

## Problem

`katalyst inspect .` starts with the raw-source inspectors. `file_tree` is the
first layer of that read: it sees paths, names, extensions, depth, and directory
shape before Katalyst opens any file.

The current `file_tree` output answers a clustering question. It groups
directory profiles into `classes` and `outliers`. That hides the filesystem map
the reader needs first: what files are present, where they live, which regions
dominate, and which path-level conventions are visible.

## Goal

`file_tree` should render a deterministic filesystem map for humans and agents.
It should help the reader answer these questions at a glance:

- How many files and directories are present?
- How deep and wide is the tree?
- What is the actual tree for a small directory?
- What are the top-level regions for a larger directory?
- Which extensions dominate?
- Which directories are dense or Markdown-heavy by extension only?
- What filename conventions are visible from paths?
- Which concrete paths support the summary?

## Boundary

`file_tree` inspects only filesystem metadata:

- relative paths
- directory structure
- filenames
- extensions
- path depth
- counts and ratios derived from paths

It must not report:

- Markdown parse status
- frontmatter presence or keys
- body sections or headings
- valid content
- candidate collections
- schemas
- framework or project type

Good output says `content/books/ contains 24 Markdown files, mostly
kebab-case.` It does not say `content/books/ is a book review collection.`

## Design

Replace the current `classes` / `outliers` evidence for `file_tree` with a
structured filesystem summary. Keep the inspector name, registry entry, and
"opens no files" contract.

The summary should include:

- `file_count`
- `dir_count`
- `max_depth`
- extension histogram
- top-level regions with descendant file counts and dominant extensions
- directory summaries for verbose output and JSON
- naming buckets with counts
- representative paths
- small-tree entries when the tree is compact
- structural exception candidates tied to a stated dominant pattern

All observations must be backed by deterministic rules and visible counts.

Suggested thresholds:

- Small tree: `file_count <= 30` and `dir_count <= 12`.
- Major region: a top-level directory in the top 8 by descendant file count, or
  any top-level directory with at least 10% of files.
- Dominant extension: one extension has at least 60% of files and at least 3
  files.
- Markdown-heavy directory: a directory has at least 3 `.md` files and `.md` is
  at least 60% of files under that directory.
- Dominant naming bucket: one bucket has at least 80% of comparable files and at
  least 3 files.
- Naming exception: a file outside a dominant naming bucket, capped in default
  output.
- Deep path: path depth greater than the 90th percentile or greater than a fixed
  threshold such as 4. Pick the simpler stable rule during implementation.

These thresholds are conservative. The inspector should say less rather than
overclaim.

## Naming buckets

Classify filename stems into deterministic buckets:

- `kebab-case`
- `snake_case`
- `camelCase`
- `PascalCase`
- `title/spaces`
- `lowercase`
- `uppercase`
- `numeric`
- `mixed/other`

Only compare regular files with non-empty stems. Directory naming is a separate
classifier if implementation needs it; file naming is the default contract.

## Verbosity

Default Markdown output is a skimmable map:

- overview sentence
- actual tree for small trees
- top-level regions for larger trees
- top 5 extensions
- naming summary only when a dominant pattern or clear exception exists
- capped representative paths or exception examples
- explicit hidden-data notices

Default output should not include exhaustive directory tables, full histograms,
or long exception lists.

Verbose Markdown shows more evidence:

- more regions
- full extension histogram
- directory density table
- naming buckets with counts
- more representative paths
- deepest paths
- more exception examples

JSON remains complete and parseable. It should not be truncated by Markdown
verbosity.

It should include the evidence downstream tools need even when Markdown output is
capped:

- file count
- directory count
- max depth
- extension histogram
- top-level regions and counts
- directory summaries
- naming buckets
- representative paths
- exception candidates
- tree entries or enough path data to reconstruct the tree

## Default Markdown shape

For a small tree:

```markdown
### file_tree (n=7)

Filesystem map: 7 files in 4 directories, max depth 2. Most files are Markdown.

Tree:
.
├── README.md
├── books/
│   ├── dune.md
│   └── it.md
└── notes/
    └── meeting-2026-06-24.md

File types:
- .md: 5
- .png: 1
- no extension: 1

Naming:
- Markdown filenames are mostly kebab-case: 4 of 5 files.
```

For a larger tree:

```markdown
### file_tree (n=184)

Filesystem map: 184 files in 26 directories, max depth 5. Markdown is the dominant extension: 128 of 184 files.

Top-level regions:
- docs/ - 54 files, mostly .md
- content/ - 86 files, mostly .md
- static/ - 31 files, mostly .png, .css
- scripts/ - 6 files, mostly .sh
- ... 4 more top-level entries hidden; pass -v to show all

File types:
- .md: 128
- .png: 18
- .yml: 9
- .css: 6
- .sh: 5
- ... 7 more extensions hidden; pass -v to show all

Naming:
- Markdown filenames are mostly kebab-case: 113 of 128 files.
- 6 Markdown files contain spaces, for example `docs/Old Notes.md`.
```

Use ASCII-only tree rendering unless the project accepts Unicode tree characters
in CLI snapshots. If Unicode tree characters ship, keep them stable and covered
by snapshots.

## Exceptions

Exceptions appear only when there is a stated pattern to be an exception to.

Good:

```markdown
Naming:
- Markdown filenames are mostly kebab-case: 113 of 128 files.
- Exceptions include `docs/Old Notes.md` and `content/books/Dune.md`.
```

Avoid:

```markdown
Outliers:
- docs/Old Notes.md
- content/books/Dune.md
```

Exception lists are capped in default output and expanded in verbose output.

## Representative paths

Representative path selection is deterministic:

1. Prefer paths from different top-level regions.
2. Within each region, sort lexicographically.
3. Cap the total number shown.
4. State when additional paths are hidden.

Representative paths ground summary claims without becoming a full file listing.

## Relationship to other raw-source inspectors

`file_tree` reports the map: paths, names, extensions, counts, depth, and
density.

`file_tree_content` reports content facts: parse status, frontmatter presence,
and directory-level content shape.

`document_shape` reports document grouping: candidate document groups, shared
fingerprints, and document-level exceptions.

When a claim requires opening a file, leave it to `file_tree_content` or
`document_shape`.

## Tests

Cover the behavior with failing tests first:

- tiny tree renders a tree-like listing
- medium tree renders top-level regions
- dominant extension and dominant naming pattern are reported with counts
- naming exceptions are grounded in paths
- default output is capped and verbose output expands it
- JSON contains complete structured evidence
- `file_tree` still opens no files

## Acceptance criteria

- `katalyst inspect . --inspector file_tree` reads as a coherent filesystem map,
  not a dump of generic summarizer internals.
- Small directories render an actual tree or tree-like listing.
- Larger directories render top-level regions and capped summaries.
- Interpretive phrases are backed by deterministic thresholds and visible
  counts.
- The inspector does not claim parse status, frontmatter, body structure,
  schemas, or collections.
- Default Markdown is concise; verbose Markdown shows more evidence; JSON remains
  complete.
- Truncation is explicit and tells the user how to see more.
- Snapshot tests cover a tiny tree, a medium tree with multiple top-level
  regions, dominant extension and naming patterns, naming exceptions, and a tree
  large enough to trigger truncation.
- Unit tests cover extension histograms, directory counts, depth calculation,
  naming bucket classification, and deterministic representative path selection.

## Documentation updates

Update the inspector deep-dive after the implementation lands:

- `docs/content/deep-dives/inspectors.md`: describe the raw-source layering as
  filesystem map, content facts, document grouping.
- `internal/inspect/doc.go`: keep the package summary aligned with the new
  `file_tree` evidence model if it names the old clustering shape.
- Generated inspector reference only changes if registry descriptor wording
  changes.

## Out of Scope

- Changing `file_tree_content`.
- Changing `document_shape`.
- Adding semantic project or framework labels.
- Adding schema recommendations.
- Adding a new CLI verbosity flag.
- Changing collection-layer inspectors.
