+++
title = "Configure checks for a collection"
weight = 10
+++

# Configure checks for a collection

You have a directory of markdown files and want Katalyst to enforce checks on
them. This guide adds a collection and attaches checks to it.

## 1. Point a collection at the directory

Collections are declared inside a storage instance. In a fresh project that is
`.katalyst/storage/local.yaml` (the default filesystem instance). Add the
collection under `collections:`, keyed by its name; `path` is the directory
relative to the instance root:

```yaml
# .katalyst/storage/local.yaml
type: filesystem
root: .
collections:
  posts:
    path: content/posts
```

If you omit `path`, the directory defaults to the collection name. If you
omit `pattern`, it defaults to `*.md`.

## 2. Attach checks

Add a `checks` list. Each entry names a `kind` and its required keys — see
the [check types reference]({{< relref "../reference/check-types/_index.md" >}})
for every check type:

```yaml
# .katalyst/storage/local.yaml
type: filesystem
root: .
collections:
  posts:
    path: content/posts
    checks:
      - kind: markdown_requires_h1
      - kind: markdown_title_matches_h1
        field: title
      - kind: filesystem_name_case
        style: kebab
```

A collection must have at least one check — either a `schema` (see [Add a
schema]({{< relref "add-a-schema.md" >}})) or a non-empty `checks` list.

## 3. Run it

```bash
katalyst check posts
```

Each item prints `OK` or a `path:line: /pointer: message` violation. Files
in `content/posts` that do not match the pattern are reported as errors, so
nothing is silently skipped.

## Lint the body as text

The checks above read frontmatter and filenames. To lint the **body** itself —
as raw text, regardless of markdown structure — use the `text_*` rules. Each
takes a regex `pattern` (or a list of literal `values`) and an optional `target`
selecting which slice of the body to test (`body`, `line`, `first-line`,
`matched-lines`):

```yaml
checks:
  # No line may contain "TODO".
  - kind: text_forbids
    target: line
    pattern: '\bTODO\b'
  # The body must mention "Sources" somewhere.
  - kind: text_requires
    pattern: Sources
  # Ban a set of literal markers (regex metacharacters are inert).
  - kind: text_denylist
    values: [FIXME, XXX]
```

Because text rules read only the body, they also lint **plain-text items** — a
`.txt` file, or a markdown file with no frontmatter — so a collection with
`pattern: "*.txt"` works the same way.

A `text_forbids` rule may declare a `fix`: a replacement template (`$1`,
`${name}` capture syntax) applied to the matched text by `katalyst fix`. This
one drops a trailing period from the first body line:

```yaml
checks:
  - kind: text_forbids
    target: first-line
    pattern: '\.(\s*)$'
    fix: '$1'
```

## See also

- [Add a schema]({{< relref "add-a-schema.md" >}}) to validate frontmatter
  shape.
- [Configuration reference]({{< relref "../reference/configuration.md" >}})
  for every key.
