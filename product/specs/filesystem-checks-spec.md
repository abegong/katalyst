# Spec - filesystem checks and collection checks

> **Status: planning.** Adds filesystem-attached checks that run before
> collections exist, while keeping collection-attached checks and sharing check
> implementations across both attachment points.

## Overview

Katalyst checks are attached to collections today. That makes schemas, variants,
and item-level rules coherent, but it prevents basic filesystem policy from
running until the user has defined collections. Introduce a second attachment
point, **FilesystemChecks**, for path-selected files under filesystem storage
instances. Keep **CollectionChecks** as the checks attached to collection
definitions.

## Value

Users can enforce early, concrete rules while they are still profiling a
project:

- Markdown filenames use kebab-case.
- Markdown content does not live deeper than N directories.
- A subtree contains only allowed extensions.
- Every populated directory has an index file.
- Asset references point at existing files.

The same check type works under a collection and under a filesystem scope when
both contexts provide the data it needs. A rule does not get a duplicate
implementation or a duplicate docs page just because it has two attachment
points.

## Current State

`cmd/check.go` resolves selectors through `project.Resolve`. Selectors name
collections or collection items. With no selectors, Katalyst expands every
configured collection. A freshly initialized project has no collections, so
`katalyst check` succeeds because there is nothing to validate.

`internal/storage/collection/parse.go` gives collections one check attachment
site: `schema:` plus `checks:`. The parser folds `schema:` into a leading
`object` check and parses each configured check through the registry. It rejects
collections with no schema, checks, or variants.

`cmd/engine.go` builds runnable per-item checks from a collection's configured
checks and variants. It also builds collection-scoped checks in a second pass.
The runtime distinction already exists:

- `checks.Check` runs once per item with `checks.Context`.
- `checks.CollectionCheck` runs once per collection with
  `checks.CollectionContext`.

`internal/checks/registry.go` documents check types with one `Descriptor`. The
descriptor has `Family`, which says what data the check reads, and `Scope`,
which currently says whether the check is collection-scoped. It does not say
where the check is attachable.

The current terms collide. A "filesystem check" can mean either:

- a check type in the `fileSystem` family, such as
  `filesystem_name_case`; or
- a future check instance attached to the filesystem instead of a collection.

This spec separates those axes.

## Design

### Configuration Sites

Add a configuration-site axis to check descriptors:

```go
type Descriptor struct {
    // existing fields...
    ConfigurableIn []string // "collection", "filesystem"
}
```

`Family` keeps its current meaning: the source data the check reads.
`ConfigurableIn` says where a check instance may be configured.

During migration, an empty `ConfigurableIn` means `["collection"]`. That keeps every
existing check valid until each descriptor opts into filesystem attachment.

Use these concepts in product language:

- **CollectionCheck:** a check instance attached to a collection definition.
- **FilesystemCheck:** a check instance attached to a filesystem scope.
- **FileCheck:** a runtime check that runs once per file.
- **FileSetCheck:** a runtime check that runs once per selected file set.

The last two replace the overloaded internal names over time. Today's
`checks.Check` is a FileCheck. Today's `checks.CollectionCheck` is a
FileSetCheck.

### Config Shape

Keep collection checks where they are:

```yaml
collections:
  posts:
    path: content/posts
    pattern: "*.md"
    checks:
      - kind: filesystem_name_case
        style: kebab
      - kind: markdown_requires_h1
```

Do not rename collection `checks:` to `collectionChecks:`. Inside a collection,
the configuration site is already clear. A rename adds migration cost without
making the config easier to write.

Add `filesystemChecks` to filesystem storage instances:

```yaml
# .katalyst/storage/local.yaml
type: filesystem
root: .

filesystemChecks:
  - name: docs
    path: docs/content
    include: ["**/*.md"]
    exclude: ["**/_generated/**"]
    checks:
      - kind: filesystem_name_case
        style: kebab
      - kind: filesystem_path_depth
        max: 4
      - kind: filesystem_index_file_required
        name: _index.md

collections:
  pages:
    path: docs/content
    pattern: "**/*.md"
    schema: page
```

Each `filesystemChecks` entry defines a filesystem scope:

| Key | Required | Default | Meaning |
|---|---|---|---|
| `name` | no | `path` | Diagnostic label and future selector handle. |
| `path` | no | `.` | Scope root, relative to the storage instance root. |
| `include` | yes | - | Glob patterns relative to `path`. |
| `exclude` | no | `[]` | Glob patterns removed from the included set. |
| `parseFailures` | no | `error` | Severity for document parse failures: `error` or `warning`. |
| `checks` | yes | - | Check instances to run over selected files. |

Names are optional. A named scope gives diagnostics and future selective
execution a stable handle. An unnamed scope uses its `path` as its diagnostic
label.

`include` is required in the first cut. Filesystem scopes serve both path-only
checks and document-aware checks, and a default would surprise one of those
cases. A later release can add a default after real configs show whether
filesystem scopes are mostly broad path scans or Markdown-focused checks.

`parseFailures` applies only when at least one check in the scope needs
document data. The default is `error`, matching collection checks and CI
expectations. Set it to `warning` for onboarding a messy tree while still
surfacing skipped document-aware checks.

The same check type can appear in both attachment sites when its descriptor
supports both configuration sites:

```yaml
type: filesystem
root: .

filesystemChecks:
  - name: allMarkdown
    path: .
    include: ["**/*.md"]
    checks:
      - kind: filesystem_name_case
        style: kebab

collections:
  posts:
    path: content/posts
    pattern: "*.md"
    checks:
      - kind: filesystem_name_case
        style: kebab
      - kind: markdown_requires_h1
```

`filesystemChecks` is valid only on `type: filesystem` storage instances.
SQLite storage instances reject it.

### Why Storage-Level Config

Filesystem checks attach to filesystem storage because storage owns the root
path. This avoids a second project-level root and keeps path resolution beside
`collections:`.

The alternative is project-level `filesystemChecks` in `.katalyst/config.yaml`:

```yaml
filesystemChecks:
  - name: docs
    root: .
    path: docs/content
    include: ["**/*.md"]
    checks:
      - kind: filesystem_name_case
        style: kebab
```

That shape is visible, but it duplicates storage root semantics. It also raises
unclear questions for SQLite-backed projects. Storage-level config fits the
existing model: a filesystem storage instance declares both its domain mapping
and its raw filesystem policy.

### Shared Runtime Contexts

Avoid duplicate implementations by normalizing both attachment points into the
same contexts.

Rename or adapt `checks.Context` into a file-oriented context:

```go
type FileContext struct {
    FilePath string
    Root     string
    Doc      *markdownbodytext.Document
    Meta     map[string]any
}
```

For collection checks:

- `FilePath` is the item path.
- `Root` is the collection directory.
- `Doc` and `Meta` come from `Project.ReadItem`.

For filesystem checks:

- `FilePath` is the selected file path.
- `Root` is the filesystem check scope root.
- `Doc` and `Meta` are populated only when a check declares that it needs
  document data.

Unify collection-scoped checks and filesystem-scoped checks around a file-set
context:

```go
type FileSetContext struct {
    Root      string
    Files     []FileContext
    Unmatched []string
    Include   []string
    Exclude   []string
}
```

Then existing set-level checks become target-independent:

- `filesystem_unique_filename` groups `ctx.Files` by basename.
- `filesystem_index_file_required` groups `ctx.Files` by directory.
- `filesystem_unique_field` groups `ctx.Files` by metadata field.

This runtime shape also fixes the naming collision. Internal
`CollectionCheck` means "set-level check" today. Product language needs
"CollectionCheck" to mean "attached to a collection." Rename the internal
interface after the filesystem runner lands.

`Files` contains the files selected by the scope after `include` and `exclude`.
`Unmatched` contains regular files under the scope root that match neither
`include` nor `exclude`. Most checks ignore `Unmatched`; the
`filesystem_unmatched_files` check uses it to enforce raw subtree coverage.

### Document Access

FilesystemChecks are document-aware, but document parsing is lazy.

Path-only checks do not open files. Checks that need metadata or markdown body
content declare that need in the registry. The filesystem runner parses only
when at least one check in the scope needs document data.

Document parse failures use the filesystem scope's `parseFailures` severity.
With the default:

```yaml
filesystemChecks:
  - name: posts
    path: content/posts
    include: ["**/*.md"]
    checks:
      - kind: filesystem_name_matches_field
        field: slug
```

an invalid document fails the run:

```text
filesystem posts: content/posts/broken.md: /: cannot parse document: invalid frontmatter
```

With `parseFailures: warning`:

```yaml
filesystemChecks:
  - name: posts
    path: content/posts
    include: ["**/*.md"]
    parseFailures: warning
    checks:
      - kind: filesystem_name_matches_field
        field: slug
```

the same parse failure is advisory:

```text
filesystem posts: content/posts/broken.md: warning: /: cannot parse document: invalid frontmatter; skipped document-aware checks
```

The runner always reports parse failures. A passing run proves that every
selected file needed by a document-aware check was inspected.

This lets almost every current filesystem-related check run under both configuration sites:

| Check type | CollectionChecks | FilesystemChecks |
|---|---:|---:|
| `filesystem_extension_in` | yes | yes |
| `filesystem_parent_dir_in` | yes | yes |
| `filesystem_name_case` | yes | yes |
| `filesystem_name_affix` | yes | yes |
| `filesystem_path_charset` | yes | yes |
| `filesystem_name_regex` | yes | yes |
| `filesystem_name_length` | yes | yes |
| `filesystem_path_depth` | yes | yes |
| `filesystem_unique_filename` | yes | yes |
| `filesystem_index_file_required` | yes | yes |
| `filesystem_name_matches_field` | yes | yes, with metadata |
| `filesystem_parent_dir_matches_field` | yes | yes, with metadata |
| `filesystem_referenced_files_exist` | yes | yes, with metadata |
| `filesystem_unique_field` | yes | yes, with metadata |

`filesystem_unique_field` is in the `structuredObject` family today despite its
historical `filesystem_` prefix. Keep the kind for compatibility. A future
rename can add an alias.

### Filesystem Unmatched Files

Add a new filesystem-only FileSetCheck:

```yaml
filesystemChecks:
  - name: docsCoverage
    path: docs/content
    include: ["**/*.md"]
    exclude: ["assets/**", "_generated/**"]
    checks:
      - kind: filesystem_unmatched_files
```

`filesystem_unmatched_files` enforces scope coverage. Every regular file under
the scope root must either match `include` and enter the checked file set, or
match `exclude` and be intentionally ignored. A file that matches neither is an
unmatched file.

Example diagnostic:

```text
filesystem docsCoverage: docs/content/raw.txt: unmatched file (does not match include ["**/*.md"] or exclude ["assets/**", "_generated/**"])
```

This check is separate from collection unmatched-file detection. Collection
unmatched detection remains the existing invariant: files inside a collection
directory must match the collection's `pattern`. The new check applies before
collections exist and only runs when the user configures it.

### Command Behavior

`katalyst check` with no selectors runs:

1. all configured FilesystemChecks
2. all configured CollectionChecks

Existing collection selectors stay collection-only:

```text
katalyst check pages
katalyst check pages/home
```

Those commands do not run unrelated filesystem scopes. Selective filesystem
execution can come later through named scopes:

```text
katalyst check --filesystem docs
```

Do not add filesystem selectors in the first implementation unless the CLI work
needs them. Running all filesystem scopes under the empty selector covers the
main onboarding and CI cases.

### Diagnostics

Diagnostics name the configuration site:

```text
filesystem docs: docs/content/Old Note.md: /path: filename is not kebab-case
```

When a scope has no `name`, use its path:

```text
filesystem docs/content: docs/content/Old Note.md: /path: filename is not kebab-case
```

Collection diagnostics keep the existing item path format unless the new target
label is needed to disambiguate output when both attachment types run.

### Migration

Existing configs keep working:

- Collection `checks:` keeps its meaning.
- Check type `kind` names keep their meaning.
- Existing collection-scoped runtime interfaces can remain during the first
  implementation.

Implementation is additive:

1. Add descriptor configurableIn metadata, defaulting to `collection`.
2. Add file and file-set runtime contexts without changing behavior.
3. Add `filesystemChecks` parsing to filesystem storage instances.
4. Run path-only FilesystemChecks.
5. Add lazy document parsing and opt metadata-aware check types into the
   filesystem configuration site.
6. Rename internal set-level interfaces away from `CollectionCheck`.

## Open Questions

_None._

## Documentation Updates

- `docs/content/deep-dives/domain-model/checks.md`: split data family,
  library, configuration site, and runtime granularity.
- `docs/content/reference/configs/bases.md`: document storage-level
  `filesystemChecks`.
- `docs/content/reference/glossary.md`: add CollectionCheck, FilesystemCheck,
  FileCheck, FileSetCheck, and configuration site. Update Check instance and
  Collection-scoped check.
- `docs/content/how-to/configure-rules.md`: add a pre-collection filesystem
  check workflow.
- Generated check-type reference: render supported configuration sites and file vs.
  file-set granularity from descriptors. Run `make docs-gen`.
- `internal/checks/AGENTS.md`: document descriptor configurableIn metadata and the
  runtime naming distinction.
- `internal/storage/collection/AGENTS.md`: document that collection `checks:`
  remain collection-attached and filesystem checks live on filesystem storage
  instances.
- `.cursor/skills/add-katalyst-check-type/SKILL.md`: update the checklist so a
  new check type declares supported configuration sites.

## Rejected Alternatives

- **Rename collection `checks:` to `collectionChecks:`.** Rejected because the
  collection block already supplies the target. The rename creates migration
  work without changing behavior.
- **Put `filesystemChecks` in `.katalyst/config.yaml`.** Rejected because it
  duplicates storage roots and makes filesystem policy independent of the
  storage instance that owns path resolution.
- **Create separate check kinds for filesystem and collection attachment.**
  Rejected because it splits docs and implementations for the same rule.
- **Infer collections to run filesystem rules.** Rejected because a filesystem
  scope has no item identity, schema, variants, or selector namespace. It is not
  a collection.
