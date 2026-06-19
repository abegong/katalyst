+++
title = "Object Validation"
+++

## Rule ID

`kind: object`

## Purpose

Validate frontmatter metadata against a named schema from `schemas:`.

## Configuration

```yaml
schemas:
  book: ./schemas/book.json

rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: object
        schema: book
```

Required fields:

- `kind` must be `object`
- `schema` must name an entry in `schemas:`

## Behavior

`katalyst` validates frontmatter values as an object instance and reports schema
violations with JSON-pointer locations.

## Example validation failure

Input file:

```markdown
---
title: Dune
year: "not-a-number"
---
# Dune
```

Command:

```bash
katalyst validate notes/bad.md
```

Output:

```text
notes/bad.md:3: /year: got string, want integer
```
