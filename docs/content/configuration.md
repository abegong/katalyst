+++
title = "Configuration"
+++

Katalyst looks for the nearest `katalyst.yaml` by walking upward from the
current working directory.

Example:

```yaml
schemas:
  book: ./schemas/book.json
  person: ./schemas/person.json

rules:
  - paths: "notes/books/**/*.md"
    checks:
      - kind: object
        schema: book
      - kind: markdown_title_matches_h1
      - kind: filesystem_filename_matches_slug
  - paths: "notes/people/**/*.md"
    schema: person # legacy shorthand for object check
```

`rules` are evaluated in source order; first matching `paths` wins.

Each rule can contain one or more `checks`:

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
3. `kind: object` checks in the first matching rule

Markdown/filesystem checks always come from the first matching rule, even when
`--schema` is used.

Rule references:

- [Object Validation]({{< relref "rules/objects/object.md" >}})
- [Title Matches H1]({{< relref "rules/markdown/title-matches-h1.md" >}})
- [Filename Matches Slug]({{< relref "rules/filesystem/filename-matches-slug.md" >}})
