+++
title = "Add a schema"
weight = 20
+++

# Add a schema

JSON Schema is how Katalyst validates the *shape* of an item's frontmatter —
required keys, types, ranges. This guide registers a schema and binds it to a
collection.

## 1. Write the schema file

Put a JSON Schema (draft 2020-12) under `.katalyst/schemas/`. Its **name** is
the filename stem — `book.json` registers a schema named `book`, with no
separate registration step:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "book",
  "type": "object",
  "required": ["title", "year"],
  "properties": {
    "title": { "type": "string", "minLength": 1 },
    "year":  { "type": "integer", "minimum": 0 }
  }
}
```

The name — not the path — is the stable handle the rest of the config uses, so
the file is free to move as long as its stem stays the same.

## 2. Bind it to a collection

The shortest way is the `schema:` shorthand, which adds a single `object`
check:

```yaml
# .katalyst/collections/books.yaml
path: notes/books
schema: book
```

Equivalently, add an explicit object check to `checks` — useful when you mix
it with markdown or filesystem checks:

```yaml
# .katalyst/collections/books.yaml
path: notes/books
checks:
  - kind: object
    schema: book
  - kind: markdown_title_matches_h1
```

## 3. Override per file or per run

A single document can opt into a different registered schema with an inline
key in its frontmatter:

```markdown
---
schema: strict-book
title: Dune
---
```

And `--schema <path>` overrides resolution for every selected item in one
run:

```bash
katalyst check books --schema ./schemas/strict-book.json
```

The precedence is `--schema` > inline `schema:` key > the collection's object
check. See the [configuration
reference]({{< relref "../reference/configuration.md" >}}) for the key surface,
or `internal/config/README.md` for why.

## See also

- [Object validation reference]({{< relref "../reference/check-types/objects/object.md" >}})
- [Inspect schemas]: `katalyst schema list` and `katalyst schema show <name>`.
