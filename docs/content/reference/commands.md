+++
title = "Commands"
weight = 20
+++

# Commands

The `katalyst` CLI. Most commands take **selectors** — nothing (the whole
project), `<collection>`, or `<collection>/<item>`.

Exit codes are shared across the validating commands:

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | One or more items failed |
| `2` | Usage error |

## `check`

```bash
katalyst check [selector ...]
katalyst check --schema <path> [selector ...]
```

Run each selected item's configured checks. Prints `path: OK` for valid
items and `path:line: /pointer: message` for violations. `--schema/-s`
overrides object-schema resolution for every selected item. Files inside a
collection directory that do not match its pattern are reported as unmatched
references (errors).

```text
notes/example.md: OK
notes/bad.md:6: /year: got string, want integer
```

## `fix`

```bash
katalyst fix [selector ...]
katalyst fix --check [selector ...]
```

Rewrite each selected item's frontmatter in canonical form (keys sorted,
block style, one trailing newline; body preserved verbatim). With `--check`,
write nothing, print items that would change, and exit `1` — the CI form.

For why `fix` is deliberately opinionated and never injects missing values,
see the `internal/frontmatter` package docs (`internal/frontmatter/README.md`).

## `inspect`

```bash
katalyst inspect <path>
katalyst inspect <path> --json
katalyst inspect <path> --inspector <name> [--inspector <name> ...]
katalyst inspect <path> --max-lines <n> | -v
katalyst inspect <path> -o report.md
```

Profile the markdown files under `<path>` and report the corpus's shape:
frontmatter field frequency and types, markdown heading and section
conventions, and filename conventions. The output is **evidence** — counts and
distributions with the file count as denominator — not recommendations.
`inspect` is read-only: it writes no schema and mutates nothing under `<path>`.

Each inspector's results are prefixed with a one-line description of what they
mean. Output is Markdown by default; `--json` emits the same evidence as JSON.

- `--inspector` narrows the run to named inspectors (see the [inspectors
  reference]({{< relref "inspectors/_index.md" >}})).
- `--max-lines <n>` truncates each inspector's Markdown output to `n` lines
  (default `20`, `0` for no limit) so one wide field can't drown the report;
  `-v`/`--verbose` is the same as `--max-lines 0`. Truncation is per inspector
  and Markdown-only — `--json` is always complete and parseable.
- `-o` writes the report to a file instead of stdout.

Unlike the selector-based commands, `inspect` takes a plain directory path and
needs no `.katalyst/` project.

## `schema`

```bash
katalyst schema list
katalyst schema show <name>
```

Inspect the schemas registered in `katalyst.yaml`.

## `collection`

```bash
katalyst collection list
katalyst collection get <collection>
```

List collections (name, directory, item count, schema) or show one
collection's detail.

## `item`

```bash
katalyst item list <collection>
katalyst item get <collection>/<item> [--frontmatter | --body]
katalyst item add <collection>/<item> [key=value ...]
katalyst item update <collection>/<item> key=value [key=value ...]
katalyst item delete <collection>/<item> [<collection>/<item> ...]
```

List, inspect, and mutate items. `add` creates an item with the given
frontmatter and an empty body; `update` merges keys into an existing item
without touching the body; `delete` removes one or more items.

### `item list` query flags

`item list` narrows, searches, and orders its output with a
MongoDB-`find`-inspired pipeline: **filter → grep → sort → skip → limit**.

```bash
katalyst item list <collection>
  [--filter EXPR ]...        # field predicate; repeatable, ANDed
  [--grep PATTERN ]...       # regexp text search; repeatable, ANDed
  [--grep-in all|body|frontmatter]   # region --grep searches (default all)
  [-i | --ignore-case]       # case-insensitive --grep
  [--sort KEY ]...           # KEY or -KEY (descending); comma-joinable
  [--skip N] [--limit N]     # pagination, applied after sorting
  [--on-type-mismatch skip|error]    # override config
  [--sort-missing last|lowest]       # override config
```

`--filter` takes `field OP value`. The operator is the first one found
scanning left to right:

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

`field` uses dot notation for nested keys (`author.name`). Values are typed
as YAML scalars, the same as `item add` (`year>=1965` is an integer
compare). A comparison against an incompatible type is skipped by default;
`--on-type-mismatch error` makes it exit 2.

`--sort` keys are `id`, `status`, or any frontmatter field. Missing-field
items sort last by default (`--sort-missing lowest` treats them as below any
value). An empty result is a success (exit 0).

```bash
katalyst item list books --filter 'year>=1965' --filter 'status=draft'
katalyst item list books --grep TODO --grep-in body -i
katalyst item list books --sort -year --limit 10        # 10 newest
```

The `--on-type-mismatch` and `--sort-missing` defaults are configurable; see
[`query`]({{< relref "configuration.md#query" >}}).

## `rules`

`rules` is a resource noun (like `collection`, `item`, `schema`): a read-only
view of the check kinds the engine can enforce, read from its registry — the
same catalog the [rules reference]({{< relref "rules/_index.md" >}}) is
generated from. It takes no selector and reads no project, so it runs in any
directory. Invoked bare it prints help; the work happens under its sub-verbs.

```bash
katalyst rules list [--family <family>] [--json]
katalyst rules show <kind> [--json]
```

- `katalyst rules list` — every kind grouped by family (objects, markdown,
  filesystem): kind, purpose, required keys, optional keys.
- `katalyst rules list --family <family>` — narrow the list to one family.
- `katalyst rules show <kind>` — a docs-style readout for one kind: family
  context, purpose, full key table, example config, and the other kinds in
  its family.
- `--json` — on either sub-verb, a machine-readable descriptor array (`list`)
  or a single object (`show`), so an editor or skill reads the catalog
  instead of hardcoding it.

Exits `0`, or `2` for an unknown family or kind.

## `init`

```bash
katalyst init [--dir <path>]
```

Scaffold a starter `katalyst.yaml`, an example schema, and an example
document. Refuses to overwrite existing files.

## See also

- [Configuration reference]({{< relref "configuration.md" >}})
- [Rules reference]({{< relref "rules/_index.md" >}})
- [Inspectors reference]({{< relref "inspectors/_index.md" >}})
