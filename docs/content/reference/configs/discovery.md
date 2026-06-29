+++
title = "Discovery"
weight = 10
+++

# Discovery

Katalyst reads a `.katalyst/` directory, found by walking upward from the
current working directory to the nearest ancestor that contains one. That
ancestor is the repo root; all relative paths resolve against it.

Discovery resolves symlinks on both the root and the input path, because on
macOS `$TMPDIR` lives behind `/var` to `/private/var` and relative-path
resolution would otherwise produce garbage.

## Layout

```
.katalyst/
  config.yaml          # optional: listing defaults and discovery settings
  schemas/             # one JSON Schema file per named schema
    book.json
  bases/               # one file per base
    local.yaml         # a base + the collections it declares
    local/             # optional: one file per collection (escape hatch)
      books.yaml
```

By default, schemas and bases are discovered by **convention**:
every file under `schemas/` is a schema whose name is its filename stem
(`book.json` -> `book`), and every file under `bases/` is a
[base]({{< relref "bases.md" >}}) named for its filename stem (`local.yaml` -> `local`).
`config.yaml` is optional; it carries [listing]({{< relref "listing.md" >}}) defaults and can
switch a kind to **explicit** discovery, listing definitions inline instead of
as files.

`config.yaml` is YAML; schema and base files default to YAML/JSON, and the
accepted format is set per kind there.

Legacy projects that still use `storage:` in `config.yaml` or
`.katalyst/storage/` continue to load. Do not mix legacy and new forms in the
same project; move legacy base files to `.katalyst/bases/` when you edit them.
