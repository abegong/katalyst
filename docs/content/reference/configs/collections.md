+++
title = "Collections"
weight = 40
+++

# Collections

A **collection** is a directory of items plus the checks every item must pass.
Collections are declared inside their base, under `collections:`.

| Key | Required | Default | Meaning |
|---|---|---|---|
| `path` | no | the collection name | Directory, relative to the base `root`. |
| `pattern` | no | `*.md` | Filename glob selecting items in the directory. |
| `table` | for `sqlite` | - | SQLite table backing the collection. |
| `id` | for `sqlite` | - | SQLite column that provides item identity. |
| `attributes` | no | all scalar columns except `id` and `content.column` | SQLite column captures exposed as item attributes. |
| `content` | no | - | Optional SQLite content mapping, with `kind: text` or `kind: markdown` and `column: <name>`. |
| `schema` | no | - | Schema name; shorthand for a leading `object` check. |
| `checks` | no | - | List of checks. See [Checks]({{< relref "checks.md" >}}). |
| `listing` | no | - | `item list` listing defaults for this collection. See [Listing]({{< relref "listing.md" >}}). |

A collection must configure at least one check: set `schema`, or provide a
non-empty `checks` list, or both. Files in the directory that do not match
`pattern` are reported as errors.

SQLite collections do not support filesystem check types. Configure
structured-object checks against captured attributes. Text and markdown
body-text checks require a compatible `content` mapping.

`attributes` accepts shorthand single-column captures and structured
multi-column captures:

```yaml
attributes:
  title: title
  author:
    columns:
      first: author_first
      last: author_last
```

The structured form above exposes `author.first` and `author.last` as fields
inside the `author` attribute object.

## Per-collection files

A base whose `collections:` block grows unwieldy may split collections into
one file each under `.katalyst/bases/<base>/<collection>.yaml`, named for
its filename stem. Inline and per-file collections coexist; a name declared both
inline and in a file is an error.

```yaml
# .katalyst/bases/local/books.yaml
path: notes/books
schema: book
```
