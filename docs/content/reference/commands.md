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

## `init`

```bash
katalyst init [--dir <path>]
```

Scaffold a starter `katalyst.yaml`, an example schema, and an example
document. Refuses to overwrite existing files.

## See also

- [Configuration reference]({{< relref "configuration.md" >}})
- [Rules reference]({{< relref "rules/_index.md" >}})
