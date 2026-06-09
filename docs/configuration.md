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
- `kind: markdown_title_matches_h1` checks that a field (default `title`) matches the first H1.
- `kind: filesystem_filename_matches_slug` checks that a field (default `slug`) matches the file basename.

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
