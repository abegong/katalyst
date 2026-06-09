+++
title = "Commands"
+++

## Validate

```bash
katalyst validate [paths...]
```

Validate markdown frontmatter against a resolved schema.
Run object, markdown, and filesystem checks resolved from config.

Example output:

```text
notes/dune.md: OK
notes/bad.md:3: /year: got string, want integer
```

Exit codes:

| Code | Meaning                         |
|-----:|---------------------------------|
| `0`  | All files valid                 |
| `1`  | One or more validation failures |
| `2`  | Usage error                     |

See [Rules Reference]({{< relref "rules/_index.md" >}}) for rule definitions and behavior.

## Format

```bash
katalyst fmt [paths...]
katalyst fmt --check [paths...]
```

Normalize frontmatter formatting.

## Schema

```bash
katalyst schema list
katalyst schema show <name>
```

Inspect configured schemas.

## CRUD

```bash
katalyst create <path> [key=value ...]
katalyst read <path>
katalyst update <path> key=value [key=value...]
katalyst delete <path> [path...]
```

Create, read, update, and delete markdown items with frontmatter.

## Init

```bash
katalyst init [--dir <path>]
```

Scaffold a starter project layout.
