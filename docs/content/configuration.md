+++
title = "Configuration"
+++

Katalyst looks for the nearest `.katalyst/` directory by walking upward from
the current working directory; that directory's parent is the project root.
Schemas live in `.katalyst/schemas/<name>.yaml` and collections in
`.katalyst/collections/<name>.yaml`, each discovered by filename.

Example — a `books` collection:

```yaml
# .katalyst/collections/books.yaml
path: notes/books        # directory, relative to the project root
pattern: "*.md"           # optional; default "*.md"
checks:
  - kind: object
    schema: book          # a schema name from .katalyst/schemas/
  - kind: markdown_title_matches_h1
  - kind: filesystem_filename_matches_slug
```

A collection backed by a single object schema can use the `schema:`
shorthand instead of an explicit `object` check:

```yaml
# .katalyst/collections/people.yaml
path: notes/people
schema: person
```

Each collection runs one or more `checks`:

- `kind: object` with `schema: <name>` validates frontmatter against a schema.
- `kind: object_required_field` requires a field to exist.
- `kind: object_field_type` enforces a field type.
- `kind: object_field_enum` enforces an allowed value list.
- `kind: object_number_range` enforces numeric bounds.
- `kind: object_string_length` enforces string length bounds.
- `kind: markdown_title_matches_h1` checks that a field (default `title`) matches the first H1.
- `kind: markdown_requires_h1` requires at least one H1.
- `kind: markdown_single_h1` requires at most one H1.
- `kind: markdown_no_heading_level_jumps` disallows heading jumps like `H1 -> H3`.
- `kind: markdown_required_section` requires a heading text.
- `kind: markdown_code_fence_language_required` requires code fence language tags.
- `kind: filesystem_filename_matches_slug` checks that a field (default `slug`) matches the file basename.
- `kind: filesystem_extension_in` allows only specific extensions.
- `kind: filesystem_filename_kebab_case` requires lowercase kebab-case basenames.
- `kind: filesystem_no_spaces_in_path` disallows spaces in paths.
- `kind: filesystem_parent_dir_in` constrains parent directory names.
- `kind: filesystem_filename_prefix` requires a filename prefix.

Object-schema resolution precedence:

1. `--schema <path>`
2. Inline `schema: <name>` key in frontmatter
3. The collection's `kind: object` checks

Markdown/filesystem checks always come from the collection, even when
`--schema` is used.

Schema discovery (`convention` vs. an explicit `defs` map) and file format
(`yaml`, `json`, or `both`) are settable per kind in `.katalyst/config.yaml`;
both default to `convention` and `yaml`.

Rule references:

- [Object Validation]({{< relref "rules/objects/object.md" >}})
- [Title Matches H1]({{< relref "rules/markdown/title-matches-h1.md" >}})
- [Filename Matches Slug]({{< relref "rules/filesystem/filename-matches-slug.md" >}})
