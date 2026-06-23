+++
title = "The CLI"
weight = 20
+++

# The CLI

The `katalyst` CLI is **self-documenting**. Run `katalyst --help` for the full
command tree, and `katalyst <command> --help` (down to leaves like
`katalyst item list --help`) for a command's arguments, flags, and behavior.
That help is the source of truth, and it is snapshot-tested, so it does not
drift from the binary.

This page is deliberately thin. It records only the cross-cutting facts the
per-command help does not surface on its own: the two command grammars, the
shared exit codes, and the filter-predicate grammar that the configuration
`when:` clause reuses.

## Command grammars

The command tree is two grammars, shown as the **Verbs** and **Resources**
groups in `katalyst --help`:

- **Verbs** (`inspect`, `init`, `check`, `fix`) operate over content and take
  **selectors**: nothing (the whole project), `<collection>`, or
  `<collection>/<item>`.
- **Resource nouns** (`collection`, `item`, `schema`, `check-types`,
  `inspectors`) carry CRUD-style sub-verbs (`list`, `get`, ...).

See [Command organization]({{< relref "../deep-dives/command-organization.md" >}})
for why the tree is shaped this way.

## Exit codes

Shared across the validating commands (`check`, `fix --check`):

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | One or more items failed |
| `2` | Usage error |

## Filter predicates

The `--filter` flag of `katalyst item list` and the `when:` clause of a
[collection variant]({{< relref "configuration.md#variants" >}}) share one
predicate language, so it is documented here once. A predicate is
`field OP value`; the operator is the first one found scanning left to right:

| Operator | Meaning |
|---|---|
| `=` | equals (against an array field, "contains") |
| `!=` | not equals |
| `>` `>=` `<` `<=` | numeric / lexicographic comparison |
| `=~` | matches a Go regexp |
| `=` with comma RHS | equals any of (`year=1965,1937`) |
| `!=` with comma RHS | equals none of |
| bare `field` | key exists |
| `!field` | key absent |

`field` uses dot notation for nested keys (`author.name`). Values are typed as
YAML scalars (`year>=1965` is an integer compare). When predicates are listed
together they are ANDed; all must hold.

Run as `item list --filter`, a comparison against an incompatible type is
skipped by default (`--on-type-mismatch error` makes it exit `2`). The rest of
the query pipeline (`--grep`, `--sort`, `--skip`/`--limit`) is documented in
`katalyst item list --help`.

## See also

- [Configuration reference]({{< relref "configuration.md" >}})
- [Check types reference]({{< relref "check-types/_index.md" >}})
- [Inspectors reference]({{< relref "inspectors/_index.md" >}})
- [Command organization]({{< relref "../deep-dives/command-organization.md" >}})
