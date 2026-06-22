+++
title = "Configuration"
weight = 10
+++

# Configuration

Katalyst reads a `.katalyst/` directory, found by walking upward from the
current working directory to the nearest ancestor that contains one. That
ancestor is the repo root; all relative paths resolve against it.

For *why* the config is shaped this way, see `internal/config/README.md`. To
set one up step by step, see [Configure checks for a
collection]({{< relref "../how-to/configure-rules.md" >}}).

## Layout

```
.katalyst/
  config.yaml          # optional: query defaults and discovery settings
  schemas/             # one JSON Schema file per named schema
    book.json
  collections/         # one YAML file per named collection
    books.yaml
```

By default, schemas and collections are discovered by **convention**: every
file under `schemas/` is a schema whose name is its filename stem
(`book.json` → `book`), and every file under `collections/` is a collection
named for its filename stem (`books.yaml` → `books`). `config.yaml` is
optional; it carries `query:` defaults and can switch a kind to **explicit**
discovery, listing definitions inline instead of as files.

## Schemas

Each file under `.katalyst/schemas/` is a JSON Schema. Its **name** — the
filename stem — is the stable public handle used by `schema show <name>`, by
an inline `schema: <name>` key in a document's frontmatter, and by a
collection's `schema:` shorthand. The path can move; the name should not.

## Collections

Each file under `.katalyst/collections/` defines one collection: a directory
of items plus the checks every item must pass. The filename stem is the
collection name.

| Key | Required | Default | Meaning |
|---|---|---|---|
| `path` | no | the collection name | Directory, relative to the repo root. |
| `pattern` | no | `*.md` | Filename glob selecting items in the directory. |
| `schema` | no | — | Schema name; shorthand for a leading `object` check. |
| `checks` | no | — | List of checks (see below). |
| `query` | no | — | `item list` query behavior for this collection (see [`query`](#query)). |

```yaml
# .katalyst/collections/books.yaml
path: notes/books
schema: book
checks:
  - kind: markdown_title_matches_h1
  - kind: filesystem_name_matches_field
```

A collection must configure at least one check: set `schema`, or provide a
non-empty `checks` list, or both. Files in the directory that do not match
`pattern` are reported as errors.

## `checks`

Each entry has a `kind` and the keys that check type requires. The 18 check
types are documented one per page in the [check types reference]({{< relref "check-types/_index.md" >}}):

```yaml
checks:
  - kind: object
    schema: book
  - kind: object_field_type
    field: year
    type: integer
  - kind: markdown_title_matches_h1
  - kind: filesystem_name_matches_field
```

## `query`

Two `item list` behaviors have configurable defaults. A `query:` block sets
them project-wide in `.katalyst/config.yaml`, and a collection's file can
override either key for that collection.

| Key | Values | Default | Meaning |
|---|---|---|---|
| `filterTypeMismatch` | `skip` · `error` | `skip` | A `--filter` comparison against an incompatible type either skips the item or exits 2. |
| `sortMissing` | `last` · `lowest` | `last` | Where items lacking the `--sort` key land: at the end (both directions), or below any present value. |

```yaml
# .katalyst/config.yaml — project default
query:
  filterTypeMismatch: skip
  sortMissing: last
```

```yaml
# .katalyst/collections/books.yaml — override for one collection
path: notes/books
schema: book
query:
  filterTypeMismatch: error
```

Resolution is highest-precedence first: the `--on-type-mismatch` /
`--sort-missing` flags, then the collection's `query:`, then the project
`query:`, then the built-in default. An unset key falls through to the next
level.

## Object-schema resolution precedence

When an item is checked against an object schema, the schema is chosen
highest-precedence first:

1. `--schema <path>` flag (applies to every selected item).
2. Inline `schema: <name>` key in the item's frontmatter.
3. The collection's `object` check (from `schema:` or an explicit entry).

Markdown and filesystem checks always come from the collection, even when
`--schema` is used.

## See also

- [Check types reference]({{< relref "check-types/_index.md" >}}) — every check type.
- `internal/config/README.md` — configuration rationale: why named
  collections, three-tier resolution, unmatched-as-error.
