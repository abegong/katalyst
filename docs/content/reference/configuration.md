+++
title = "Configuration"
weight = 10
+++

# Configuration

Katalyst reads a single `katalyst.yaml`, found by walking upward from the
current working directory to the nearest ancestor that contains it. That
directory is the repo root; all relative paths resolve against it.

For *why* the config is shaped this way, see the [configuration
rationale]({{< relref "../explanation/configuration.md" >}}). To set one up
step by step, see [Configure checks for a
collection]({{< relref "../how-to/configure-rules.md" >}}).

## Top-level keys

| Key | Type | Meaning |
|---|---|---|
| `schemas` | map | Schema name → path to a JSON Schema file. |
| `collections` | map | Collection name → collection definition (below). |

```yaml
schemas:
  book: ./schemas/book.json
  person: ./schemas/person.json

collections:
  books:
    path: notes/books
    schema: book
  people:
    path: notes/people
    schema: person
    checks:
      - kind: markdown_title_matches_h1
      - kind: filesystem_filename_matches_slug
```

## `schemas`

A map from a **name** to a JSON Schema file path. The name is the stable
public handle used by `schema show <name>`, by an inline `schema: <name>`
key in a document's frontmatter, and by a collection's `schema:` shorthand.
Paths are resolved relative to the repo root.

## `collections`

A map from a **collection name** to its definition. Each collection is a
directory of items plus the checks every item must pass.

| Key | Required | Default | Meaning |
|---|---|---|---|
| `path` | no | the collection name | Directory, relative to the repo root. |
| `pattern` | no | `*.md` | Filename glob selecting items in the directory. |
| `schema` | no | — | Schema name; shorthand for a leading `object` check. |
| `checks` | no | — | List of checks (see below). |

A collection must configure at least one check: set `schema`, or provide a
non-empty `checks` list, or both. Files in the directory that do not match
`pattern` are reported as errors.

## `checks`

Each entry has a `kind` and the keys that kind requires. The 18 kinds are
documented one per page in the [rule reference]({{< relref "rules/_index.md" >}}):

```yaml
checks:
  - kind: object
    schema: book
  - kind: object_field_type
    field: year
    type: integer
  - kind: markdown_title_matches_h1
  - kind: filesystem_filename_matches_slug
```

## Object-schema resolution precedence

When an item is checked against an object schema, the schema is chosen
highest-precedence first:

1. `--schema <path>` flag (applies to every selected item).
2. Inline `schema: <name>` key in the item's frontmatter.
3. The collection's `object` check (from `schema:` or an explicit entry).

Markdown and filesystem checks always come from the collection, even when
`--schema` is used.

## See also

- [Rules reference]({{< relref "rules/_index.md" >}}) — every check kind.
- [Configuration rationale]({{< relref "../explanation/configuration.md" >}})
  — why named collections, three-tier resolution, unmatched-as-error.
