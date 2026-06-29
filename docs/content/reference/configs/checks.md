+++
title = "Checks"
weight = 50
+++

# Checks

Each check instance has a `kind` and the keys that check type requires. A check
instance can be attached to a collection under `collections.<name>.checks`, or
to a raw filesystem scope under `filesystemChecks[].checks` when the check
type supports the `filesystem` target. Every check type is documented one per
page in the
[check types reference]({{< relref "../check-types/_index.md" >}}):

```yaml
checks:
  - kind: object
    schema: book
  - kind: object_field_type
    field: year
    type: integer
  - kind: markdown_title_matches_h1
  - kind: filesystem_name_matches_field
```

## Configuration Sites

Collection-attached checks run after a file belongs to a collection. They can
use collection schemas, variants, item selectors, and collection-wide sibling
sets:

```yaml
collections:
  posts:
    path: content/posts
    checks:
      - kind: markdown_requires_h1
      - kind: filesystem_name_case
        style: kebab
```

Filesystem-attached checks run from filesystem base config. They select files
with `include` and `exclude` globs and do not require collections to exist:

```yaml
filesystemChecks:
  - name: docs
    path: docs/content
    include: ["**/*.md"]
    parseFailures: warning
    checks:
      - kind: filesystem_name_case
        style: kebab
      - kind: filesystem_name_matches_field
        field: title
```

`katalyst check` with no selector runs filesystem scopes first, then collection
checks. `katalyst check <collection>` and `katalyst check <collection>/<item>`
run collection checks only. Filesystem scopes reject check types that do not
list `filesystem` in `configurableIn`.

Document-aware filesystem checks parse selected files only when needed.
`parseFailures: error` is the default and fails the run on parse errors.
`parseFailures: warning` reports the parse error as advisory and skips
document-aware checks for that file.

## Text rules

The `text_*` check types lint the item **body** as raw text, independent of
markdown structure, and also apply to plain-text items (a `.txt` file or a
markdown file with no frontmatter). Each is evaluated against a set of **spans**
chosen by `target`:

| `target` | Spans |
|---|---|
| `body` (default) | the entire body as one multiline string |
| `line` | each body line |
| `first-line` | the first non-blank body line |
| `matched-lines` | each body line matching `select: <regex>` |

- `text_requires` and `text_forbids` take a Go `pattern`, matched **unanchored**
  (it must appear *somewhere* in a span: unlike `filesystem_name_regex`, which
  anchors with `^...$`). `text_requires` also takes `match: any` (default, at
  least one span matches) or `match: all` (every span must match).
- `text_denylist` takes `values:`, a list of literal substrings; regex
  metacharacters are inert.
- `text_forbids` may declare a `fix:`: a replacement template (`$1`, `${name}`
  capture syntax) applied to the matched text by `katalyst fix`. The fix
  re-checks its own work and fails rather than writing a file the rule would
  still reject. `text_requires` and `text_denylist` are report-only.

## Object-schema resolution precedence

When an item is checked against an object schema, the schema is chosen
highest-precedence first:

1. `--schema <path>` flag (applies to every selected item).
2. Inline `schema: <name>` key in the item's frontmatter.
3. The collection's `object` check (from `schema:` or an explicit entry), plus
   the matched [variant]({{< relref "variants.md" >}})'s schema: both apply, additively.

Markdown and filesystem checks always come from the collection (and the matched
variant), even when `--schema` is used.
