+++
title = "Bases"
weight = 30
+++

# Bases

A **base** is one configured backend store, plus the collections it maps onto
the domain model. Each file under `.katalyst/bases/` is one base, named for its
filename stem. There is no implicit base; `katalyst init` writes a default
`local` one.

| Key | Required | Default | Meaning |
|---|---|---|---|
| `type` | no | `filesystem` | Backend kind: `filesystem` or `sqlite`. |
| `root` | no | `.` | Base root directory, relative to the repo root. Collection paths resolve against it. |
| `path` | for `sqlite` | - | SQLite database path, relative to the repo root. Alias for `root` on SQLite bases. |
| `filesystemChecks` | no | - | Filesystem-attached checks for raw files under a filesystem base. Valid only when `type: filesystem`. |
| `collections` | no | - | Map of collection name -> definition. See [Collections]({{< relref "collections.md" >}}). |

```yaml
# .katalyst/bases/local.yaml
type: filesystem
root: .
collections:
  books:
    path: notes/books
    schema: book
    checks:
      - kind: markdown_title_matches_h1
```

Collection names are unique across the whole project (selectors are
`<collection>/<item>`, with no base qualifier).

Filesystem bases may also declare `filesystemChecks` for rules that run before
or beside collections. Each entry selects files under the base root, then runs
checks whose descriptors support the `filesystem` target:

```yaml
# .katalyst/bases/local.yaml
type: filesystem
root: .
filesystemChecks:
  - name: docs
    path: docs/content
    include: ["**/*.md"]
    exclude: ["**/_generated/**"]
    parseFailures: error
    checks:
      - kind: filesystem_name_case
        style: kebab
      - kind: filesystem_unmatched_files
collections: {}
```

| Key | Required | Default | Meaning |
|---|---|---|---|
| `name` | no | `path` | Diagnostic label and future selector handle. |
| `path` | no | `.` | Scope root, relative to the base root. |
| `include` | yes | - | Glob patterns relative to `path`; selected files match at least one. |
| `exclude` | no | `[]` | Glob patterns removed from the selected set and from unmatched-file reporting. |
| `parseFailures` | no | `error` | `error` fails the run; `warning` reports advisory parse failures when a check needs document data. |
| `checks` | yes | - | Check instances to run against selected files or the selected file set. |

SQLite bases use one table per collection. Each row is one item:

```yaml
# .katalyst/bases/db.yaml
type: sqlite
path: content.sqlite
collections:
  books:
    table: books
    id: slug
    attributes:
      title: title
      status: status
      author:
        columns:
          first: author_first
          last: author_last
    content:
      kind: markdown
      column: body
    checks:
      - kind: object_required_field
        field: title
```
