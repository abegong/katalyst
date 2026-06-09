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
    schema: book
  - paths: "notes/people/**/*.md"
    schema: person
```

Schema resolution precedence:

1. `--schema <path>`
2. Inline `schema: <name>` key in frontmatter
3. First matching rule in `rules`
