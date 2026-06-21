+++
title = "Configure checks for a collection"
weight = 10
+++

# Configure checks for a collection

You have a directory of markdown files and want Katalyst to enforce checks on
them. This guide adds a collection and attaches checks to it.

## 1. Point a collection at the directory

In `katalyst.yaml`, add an entry under `collections`. The key is the
collection name; `path` is the directory relative to the repo root:

```yaml
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
collections:
  posts:
    path: content/posts
    checks:
      - kind: markdown_requires_h1
      - kind: markdown_title_matches_h1
        field: title
      - kind: filesystem_filename_kebab_case
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

## See also

- [Add a schema]({{< relref "add-a-schema.md" >}}) to validate frontmatter
  shape.
- [Configuration reference]({{< relref "../reference/configuration.md" >}})
  for every key.
