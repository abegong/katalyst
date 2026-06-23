+++
title = "Configuration"
weight = 10
+++

# Configuration

Katalyst reads a `.katalyst/` directory, found by walking upward from the
current working directory to the nearest ancestor that contains one. That
ancestor is the repo root; all relative paths resolve against it.

For *why* the config is shaped this way, see `internal/config/README.md`. To
set one up step by step, see [Configure checks for a
collection]({{< relref "../how-to/configure-rules.md" >}}).

## Layout

```
.katalyst/
  config.yaml          # optional: query defaults and discovery settings
  schemas/             # one JSON Schema file per named schema
    book.json
  storage/             # one file per storage instance
    local.yaml         # an instance + the collections it declares
    local/             # optional: one file per collection (escape hatch)
      books.yaml
```

By default, schemas and storage instances are discovered by **convention**:
every file under `schemas/` is a schema whose name is its filename stem
(`book.json` → `book`), and every file under `storage/` is a
[storage instance](#storage-instances) named for its filename stem
(`local.yaml` → `local`). `config.yaml` is optional; it carries `query:`
defaults and can switch a kind to **explicit** discovery, listing definitions
inline instead of as files.

## Schemas

Each file under `.katalyst/schemas/` is a JSON Schema. Its **name**, the
filename stem, is the stable public handle used by `schema show <name>`, by
an inline `schema: <name>` key in a document's frontmatter, and by a
collection's `schema:` shorthand. The path can move; the name should not.

## Storage instances

A **storage instance** is one configured backend store, today always the local
filesystem, plus the collections it maps onto the domain model. Each file under
`.katalyst/storage/` is one instance, named for its filename stem. There is no
implicit instance; `katalyst init` writes a default `local` one.

| Key | Required | Default | Meaning |
|---|---|---|---|
| `type` | no | `filesystem` | Backend kind. `filesystem` is the only kind today. |
| `root` | no | `.` | Instance root directory, relative to the repo root. Collection paths resolve against it. |
| `collections` | no | - | Map of collection name → definition (see below). |

```yaml
# .katalyst/storage/local.yaml
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
`<collection>/<item>`, with no instance qualifier).

## Collections

A **collection** is a directory of items plus the checks every item must pass.
Collections are declared inside their storage instance, under `collections:`.

| Key | Required | Default | Meaning |
|---|---|---|---|
| `path` | no | the collection name | Directory, relative to the instance `root`. |
| `pattern` | no | `*.md` | Filename glob selecting items in the directory. |
| `schema` | no | - | Schema name; shorthand for a leading `object` check. |
| `checks` | no | - | List of checks (see below). |
| `query` | no | - | `item list` query behavior for this collection (see [`query`](#query)). |

A collection must configure at least one check: set `schema`, or provide a
non-empty `checks` list, or both. Files in the directory that do not match
`pattern` are reported as errors.

### Per-collection files

An instance whose `collections:` block grows unwieldy may split collections into
one file each under `.katalyst/storage/<instance>/<collection>.yaml`, named for
its filename stem. Inline and per-file collections coexist; a name declared both
inline and in a file is an error.

```yaml
# .katalyst/storage/local/books.yaml
path: notes/books
schema: book
```

## `checks`

Each entry has a `kind` and the keys that check type requires. Every check
type is documented one per page in the [check types reference]({{< relref "check-types/_index.md" >}}):

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

### Text rules

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
  anchors with `^…$`). `text_requires` also takes `match: any` (default, at
  least one span matches) or `match: all` (every span must match).
- `text_denylist` takes `values:`, a list of literal substrings; regex
  metacharacters are inert.
- `text_forbids` may declare a `fix:`: a replacement template (`$1`, `${name}`
  capture syntax) applied to the matched text by `katalyst fix`. The fix
  re-checks its own work and fails rather than writing a file the rule would
  still reject. `text_requires` and `text_denylist` are report-only.

## `variants`

A collection runs its base `schema`/`checks` against every item. **Variants**
let it run *extra* checks on a subset, chosen by the item's metadata. Each
entry in a collection's `variants:` list has a `when` discriminator and its own
`schema`/`checks`:

```yaml
pages:
  path: docs/content
  pattern: "**/*.md"
  schema: page                  # base: every page needs a title
  variants:
    - when: "bookCollapseSection"   # section landing pages have this flag
      schema: section_index
    - when: "!bookCollapseSection"  # every other page is a content page
      schema: content_page
      checks:
        - kind: object_required_field
          field: weight
        - kind: markdown_requires_h1
  useExhaustiveVariants: false   # default
```

**`when`** is a list of [`item list --filter`]({{< relref "commands.md" >}})
predicates (`field=value`, `field>=n`, `field=~regex`, `!field`, ...), evaluated
against the item's frontmatter. All entries must hold (AND). Three shapes are
accepted, the first two desugaring to the third:

```yaml
when: "kind=section"             # one predicate
when: ["kind=section", "w>1"]    # a list of predicates
when: { where: ["kind=section"] }
```

**Resolution.** An item runs the base checks plus the checks of the **first**
variant (in list order) whose `when` it satisfies, at most one variant
applies. A variant *adds* to the base, so a check belongs in a variant exactly
when some page type must skip it: in the example, `weight` and the H1
requirement apply to content pages but not section indexes. A variant may
declare no checks at all (a deliberate exemption).

An item that matches **no** variant runs the base checks alone. Set
**`useExhaustiveVariants: true`** to instead make an unmatched item a check
failure (`matches no variant`), so every item is provably accounted for.

Discrimination is by metadata only; selecting items by path or filename is not
supported yet (a page type distinguishable only by location needs a frontmatter
marker). `pattern` still governs collection **membership** and which files are
reported as [unmatched]({{< relref "../deep-dives/domain-model.md" >}}#invariants);
variants only route checks.

## `query`

Two `item list` behaviors have configurable defaults. A `query:` block sets
them project-wide in `.katalyst/config.yaml`, and a collection's file can
override either key for that collection.

| Key | Values | Default | Meaning |
|---|---|---|---|
| `filterTypeMismatch` | `skip` · `error` | `skip` | A `--filter` comparison against an incompatible type either skips the item or exits 2. |
| `sortMissing` | `last` · `lowest` | `last` | Where items lacking the `--sort` key land: at the end (both directions), or below any present value. |

```yaml
# .katalyst/config.yaml — project default
query:
  filterTypeMismatch: skip
  sortMissing: last
```

```yaml
# under a storage instance's collections: — override for one collection
books:
  path: notes/books
  schema: book
  query:
    filterTypeMismatch: error
```

Resolution is highest-precedence first: the `--on-type-mismatch` /
`--sort-missing` flags, then the collection's `query:`, then the project
`query:`, then the built-in default. An unset key falls through to the next
level.

## Object-schema resolution precedence

When an item is checked against an object schema, the schema is chosen
highest-precedence first:

1. `--schema <path>` flag (applies to every selected item).
2. Inline `schema: <name>` key in the item's frontmatter.
3. The collection's `object` check (from `schema:` or an explicit entry), plus
   the matched [variant](#variants)'s schema: both apply, additively.

Markdown and filesystem checks always come from the collection (and the matched
variant), even when `--schema` is used.

## See also

- [Check types reference]({{< relref "check-types/_index.md" >}}), every check type.
- [Storage layer]({{< relref "../deep-dives/storage.md" >}}), the storage
  instance / collection-definition model and its lineage.
- `internal/config/README.md`, configuration rationale: storage instances,
  three-tier resolution, unmatched-as-error.
