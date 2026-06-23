# internal/checks

The check engine: the set of checks run against an item, and the result those
checks produce.

## Check

A single check run against an item: a type constraint, a heading requirement,
a filename convention, a forbidden phrase. Each comes from one of the check
types Katalyst ships, grouped into four **families** by the kind of source data
they read. Each family is its own package under `internal/checks/`, with one
file per check type:

- **structuredObject** (`structuredobject/`): the schema checks over
  frontmatter: `object` (full JSON Schema) plus targeted `object_required_field`,
  `object_field_type`, `object_field_enum`, `object_number_range`,
  `object_string_length`, and the collection-scoped `unique_field` (kind
  `filesystem_unique_field`).
- **markdownBodyText** (`markdownbodytext/`): `markdown_title_matches_h1`,
  `markdown_requires_h1`, `markdown_single_h1`,
  `markdown_no_heading_level_jumps`, `markdown_required_section`,
  `markdown_code_fence_language_required`.
- **fileSystem** (`filesystem/`): the name/path conventions
  (`filesystem_name_case`, `filesystem_name_regex`, `filesystem_path_depth`, ...)
  plus the collection-scoped `filesystem_unique_filename` and
  `filesystem_index_file_required`.
- **plainText** (`plaintext/`): `text_requires`, `text_forbids`,
  `text_denylist`: regex and literal-substring policy over the body, evaluated
  against a **span** selected by `target`. Reading only the body, they also lint
  plain-text and frontmatter-less items.

Family and granularity are orthogonal: a collection-scoped check is grouped by
the data it reads, not by its scope, so `unique_field` lives in `structuredObject`
while `unique_filename` lives in `fileSystem`. The `kind` ids are the wire
contract and never change, even when a check's family does.

Each check type implements `checks.Check` (`Run(Context) []Violation`), or
`checks.CollectionCheck` for collection-scoped checks, and lives in its own
file with its `Descriptor` and an `init()` that registers it through its
package's `register` helper (in `library.go`), which stamps `Descriptor.Library`
with the family's [CheckLibrary](#checklibrary). The core `checks` package owns
the shared types (`Context`, `Violation`, `MarkdownLines`, `LookupLine`) and the
registry; the family packages import the core, never the reverse. Callers wire
every family in by blank-importing `internal/checks/all`.

The registry (`registry.go`, populated by those `Register` calls) is the single
source of truth: `cmd/engine` builds the runnable check list by registry lookup
(`Build` / `BuildCollection`), and `cmd/gendocs` and `katalyst check-types`
render the catalog from `Descriptors()` / `Families()`. `registry_test.go`
enforces parity with `config.normalizeCheck`, so a new check type cannot ship
undocumented, and that every check type names a registered library.

## CheckLibrary

A **CheckLibrary** is the provider behind a check type. Each of the four native
families registers one in its `library.go` (always available, in-process); the
`internal/checks/jsonschema` library additionally implements `SchemaLibrary` to
compile a named schema. A library is *provenance*, orthogonal to a check type's
*family* (source-data kind): the `structuredObject` family holds both `object`
(the `json-schema` library) and `object_required_field` (the `structuredobject`
library). The engine resolves a kind's library via `checks.LibraryFor` and
fails the run if a library's `Available()` errors.

## Validation result

The product of running an item's checks. Two states:

- **Valid**: nothing to print except the conventional `path: OK`.
- **Invalid**: a flat list of violations, each with a JSON pointer `Path`
  and a `Message`. JSON Schema's raw error tree is nested and unhelpful for
  line-level reporting, so it is flattened.

When combined with `Document.Lines`, a violation becomes a
`path:line: /pointer: message` user-visible line. If the exact pointer has
no recorded line (e.g. for "missing required property" errors), the resolver
walks up to the nearest ancestor that does, pointing at the parent object
is better than pointing at nothing.

## Invariant

**Schema compilation happens once per process per absolute path.** The
resolver's compiled-schema cache (see `internal/config`) is the bottleneck,
not the JSON Schema library.
