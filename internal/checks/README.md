# internal/checks

The check engine: the set of checks run against an item, and the result those
checks produce.

## Check

A single check run against an item — a type constraint, a heading requirement,
a filename convention. Each comes from one of the **18 check types** Katalyst
ships, in three families:

- **Object** (6): `object` (full JSON Schema), plus targeted
  `object_required_field`, `object_field_type`, `object_field_enum`,
  `object_number_range`, `object_string_length`.
- **Markdown** (6): `markdown_title_matches_h1`, `markdown_requires_h1`,
  `markdown_single_h1`, `markdown_no_heading_level_jumps`,
  `markdown_required_section`, `markdown_code_fence_language_required`.
- **Filesystem** (6): `filesystem_filename_matches_slug`,
  `filesystem_extension_in`, `filesystem_filename_kebab_case`,
  `filesystem_no_spaces_in_path`, `filesystem_parent_dir_in`,
  `filesystem_filename_prefix`.

Each check type implements one `checks.Check` interface (`Run(Context)
[]Violation`) and is documented, per check type, in the generated check-types
reference. The per-check-type descriptors in `registry.go` are the source of
truth for that reference, so a new check type cannot ship undocumented.

## Validation result

The product of running an item's checks. Two states:

- **Valid**: nothing to print except the conventional `path: OK`.
- **Invalid**: a flat list of violations, each with a JSON pointer `Path`
  and a `Message`. JSON Schema's raw error tree is nested and unhelpful for
  line-level reporting, so it is flattened.

When combined with `Document.Lines`, a violation becomes a
`path:line: /pointer: message` user-visible line. If the exact pointer has
no recorded line (e.g. for "missing required property" errors), the resolver
walks up to the nearest ancestor that does — pointing at the parent object
is better than pointing at nothing.

## Invariant

**Schema compilation happens once per process per absolute path.** The
resolver's compiled-schema cache (see `internal/config`) is the bottleneck,
not the JSON Schema library.
