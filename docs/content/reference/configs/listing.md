+++
title = "Listing"
weight = 70
+++

# Listing

Two `item list` behaviors have configurable defaults. A `listing:` block sets
them project-wide in `.katalyst/config.yaml`, and a collection's file can
override either key for that collection.

| Key | Values | Default | Meaning |
|---|---|---|---|
| `filterTypeMismatch` | `skip` / `error` | `skip` | A `--filter` comparison against an incompatible type either skips the item or exits 2. |
| `sortMissing` | `last` / `lowest` | `last` | Where items lacking the `--sort` key land: at the end (both directions), or below any present value. |

```yaml
# .katalyst/config.yaml - project default
listing:
  filterTypeMismatch: skip
  sortMissing: last
```

```yaml
# under a base's collections: override for one collection
books:
  path: notes/books
  schema: book
  listing:
    filterTypeMismatch: error
```

Resolution is highest-precedence first: the `--on-type-mismatch` /
`--sort-missing` flags, then the collection's `listing:`, then the project
`listing:`, then the built-in default. An unset key falls through to the next
level.
