# Spec — SQLite attributes and content shapes

> **Status: planning.** Follow-up design for issue #101 and the draft SQLite
> storage PR. This spec captures the terminology and abstraction decisions that
> should be settled before SQLite support graduates from an experimental first
> pass.

## Overview

SQLite rows can be Katalyst items, but they are not Markdown files. The first
SQLite implementation proves the storage path by mapping a table row into the
current Markdown-shaped item model: non-ID scalar columns become metadata, and
an optional configured column becomes the item body. That is useful as a stress
test, but it smuggles Markdown terms into a relational backend.

This spec sharpens the model:

- SQLite collection definitions must describe how row values become item
  attributes.
- A captured attribute can come from one column or from a set of columns that
  form a structured object with one field per column.
- Textual or document content is a separate content shape, not an implicit
  "body" concern of every backend.
- Check-family compatibility depends on the shapes a collection provides
  (attributes, text content, Markdown document content, filesystem reference),
  not on the storage backend name alone.

## Value

The SQLite backend is the first serious test of whether Katalyst's new codec and
storage architecture can work for non-filesystem data. If SQLite rows must
pretend to be Markdown documents to participate in checks, the abstraction is
still too narrow. If the collection definition can explicitly map row data into
attributes and optional content shapes, then the same architecture can later
support sidecars, JSON records, databases, and mixed storage without forcing
their terms through frontmatter and body.

## Current State

The draft SQLite PR now has a working storage backend plus an initial
attribute/content mapping surface, but several Markdown-shaped assumptions
remain:

- `internal/storage/collection.Collection` and `RawCollection` carry
  SQLite-specific `Table`, `IDColumn`, `Attributes`, `ContentKind`, and
  `ContentColumn` fields. The old `body:` key is still accepted as a deprecated
  compatibility alias for Markdown content.
- `internal/storage/collection/sqlite.Definition` reads a row, applies
  configured attribute captures, optionally maps one column to content, and
  still synthesizes a `markdownbodytext.Document` for the rest of Katalyst to
  consume.
- `internal/project.ItemContent` currently returns raw bytes plus
  `*markdownbodytext.Document`, so project reads expose the Markdown content
  shape directly.
- `internal/checks.Context` carries `Doc *markdownbodytext.Document` and
  `Meta map[string]any`. Structured-object checks read `Meta`; plaintext and
  Markdown body-text checks read `Doc.Body`.
- `internal/inspect.CollectionView` stores parsed
  `*markdownbodytext.Document` values and exposes `Frontmatter()` and
  `Bodies()`.
- `cmd/item.go` exposes `item get --frontmatter`, `item get --body`, and
  `item list --grep-in frontmatter|body|all`.

These are reasonable inheritance points from the Markdown-first product, but a
SQLite row should not need frontmatter or body semantics unless the collection
configuration explicitly asks Katalyst to project the row into such a content
shape.

## Levels of Abstraction

This is the current stack, using the concrete directories and symbols in the
codebase.

### Storage backend

Code:

- `internal/storage/storage.go`
- `internal/storage/collection/filesystem`
- `internal/storage/collection/sqlite`
- `internal/project` backend resolution

The storage backend owns how Katalyst lists, reads, adds, updates, deletes, and
references stored items. It should know about tables, SQL queries, files, and
backend-specific write mechanics. It should not define whether a row is
"frontmatter" or "body"; those are content-shape concerns.

### Collection definition

Code:

- `internal/storage/collection.Collection`
- `internal/storage/collection.RawCollection`
- `internal/storage/collection.Build`

The collection definition is the contract between storage and checks. For
SQLite, it should include the table name, the item ID column, and the attribute
capture rules. It may also declare optional content-shape mappings. This is the
right layer for saying "attribute `author` comes from columns
`author_first_name` and `author_last_name`."

### Item identity and reference

Code:

- `internal/storage/collection.Item`
- `internal/project.Reference`
- `internal/project.ItemExists`

An item is the collection-relative unit Katalyst operates on. For SQLite, the
item ID is the configured ID-column value. `collection.Item.Path` currently
serves as a display/reference string across backends, but for SQLite it is not a
filesystem path. Long term, references should make that distinction explicit.

### Attribute object

Code today:

- `checks.Context.Meta`
- `markdownbodytext.Document.Meta`
- `internal/inspect.CollectionView.Frontmatter`
- `internal/storage/collection/predicate`
- structured-object checks under `internal/checks/structuredobject`

An attribute is the general term: any named characteristic of an item. A field
is the structured-object specialization: a key in the object passed to
structured-object checks. Every field is an attribute; not every attribute is a
field.

For SQLite, captured attributes should produce the structured object that these
checks and predicates read. The current `Meta` name is an implementation
artifact from Markdown frontmatter. It should eventually become an
attributes/object surface rather than a metadata/frontmatter surface.

### Content shape

Code today:

- `internal/codec/markdownbodytext.Document`
- plaintext checks under `internal/checks/plaintext`
- Markdown body-text checks under `internal/checks/markdownbodytext`
- `internal/fix`
- `internal/inspect.CollectionView.Bodies`

A content shape is the parsed or decoded representation that text-oriented
checks consume. `markdownbodytext.Document` is one content shape: a Markdown
document with frontmatter, body bytes, and line maps. SQLite does not have that
shape by default. A SQLite collection may opt into a text or Markdown document
shape by mapping a column to it, but the base row-to-item mapping should remain
attribute-oriented.

### CLI surface

Code:

- `cmd/item.go`
- `cmd/check.go`
- `cmd/inspect.go`
- `cmd/fix.go`
- `cmd/write_validation.go`

The CLI currently names Markdown parts directly: frontmatter and body. That is
accurate for Markdown files but not for SQLite. The CLI needs either generalized
flags (`--attributes`, `--content`) or backend-aware compatibility behavior
before SQLite feels native.

## Design

### Attribute capture in SQLite collections

A SQLite collection definition should explicitly describe how attributes are
captured from table columns. The shape should support both simple and structured
captures:

```yaml
storage:
  main:
    type: sqlite
    path: ./catalog.db
    collections:
      books:
        table: books
        id: slug
        attributes:
          title:
            column: title
          status:
            column: status
          author:
            columns:
              first: author_first_name
              last: author_last_name
```

In this example:

- `title` is an attribute backed by one column.
- `status` is an attribute backed by one column.
- `author` is an attribute backed by a set of columns. In the structured object,
  it becomes an object with fields `first` and `last`.

The exact YAML syntax is still open, but the key design point is not: SQLite
columns should not be automatically called metadata, and a multi-column
attribute should be representable as a nested structured value.

### Content mapping is optional and separate

If a SQLite table has a text column that should participate in text or Markdown
checks, the collection should declare that as a content mapping rather than a
body column:

```yaml
content:
  kind: markdown
  column: markdown_text
```

or:

```yaml
content:
  kind: text
  column: description
```

This keeps "body" scoped to Markdown document internals. A backend may provide
textual content, but that does not make the backend itself body-oriented.

### Check-family compatibility

Check support should be decided from the shapes a collection provides:

- Structured-object checks require an attribute object. SQLite can support these
  when the collection defines attribute capture rules.
- JSON Schema checks validate that same attribute object. They do not require
  Markdown frontmatter, even if the current implementation still routes through
  `Meta`.
- Plaintext checks require a text content shape. SQLite supports them only when
  the collection maps a column into text content.
- Markdown body-text checks require a Markdown document content shape. SQLite
  supports them only when the collection maps a column into Markdown document
  content and supplies the fields those checks reference.
- Filesystem checks require filesystem references and path-like attributes.
  SQLite should reject unsupported filesystem checks at config or load time
  unless a future collection explicitly supplies the needed filesystem shape.
- `fix` remains unsupported for SQLite until write-back semantics are defined
  for the relevant shapes.

This is more durable than a backend allowlist. A filesystem collection without
Markdown content should not get Markdown checks just because it is filesystem
backed; a SQLite collection with explicit text content should not be blocked
from text checks just because it is SQLite backed.

### Document terminology

`Document` remains a useful term when the concrete textual object is the
subject. The current `markdownbodytext.Document` type is specifically a
Markdown document content shape: frontmatter, body, line map, and source format.
Future text files that are not Markdown may also reasonably be described as
documents in user-facing prose, but they should not be represented by
`markdownbodytext.Document` unless they actually have that shape.

The immediate change is not to ban "document." It is to stop using a Markdown
document as the universal item content type.

## Decisions

- Use **attribute** as the general term for named item characteristics.
- Use **field** only for keys in a structured object.
- Do not use frontmatter, metadata, or body as SQLite collection concepts.
- SQLite collection definitions own attribute capture from columns.
- SQLite item identity comes from the configured item ID column.
- SQLite support can be documented as experimental while these abstractions
  settle.
- Unsupported check families should fail at config or load time.
- Split backend reference docs into distinct filesystem and SQLite subsections,
  and consider splitting the configuration reference into more than one file.

## Open Questions

1. What exact YAML shape should `attributes:` use? The design needs to support
   single-column attributes and multi-column structured attributes without being
   too verbose for common tables.
2. Should SQLite default to capturing every scalar column except the item ID
   when `attributes:` is omitted, or should explicit capture be required?
3. For a multi-column attribute, are the nested field names always user-defined,
   or can they default from column names?
4. Are composite attributes writable through `item add` and `item update` in
   v1, or are only single-column attributes writable at first?
5. What should the optional text/document mapping be called: `content`, `text`,
   `document`, or something else?
6. Should `item get --frontmatter` gain a generalized `--attributes` flag, and
   should `--frontmatter` become a Markdown-only alias?
7. Should `item list --grep-in frontmatter` gain `--grep-in attributes`, and how
   should compatibility behave for existing scripts?
8. Should `checks.Context.Meta` be renamed to `Attributes`, `Object`, or
   `Fields`, and should that happen before or after the SQLite PR merges?
9. Should `internal/inspect.CollectionView.Frontmatter()` become `Attributes()`
   or `Objects()`, and should the `object_fields` inspector description change
   from frontmatter-oriented language to attribute/object language?
10. Does `markdownbodytext.Document` stay as the check-facing content shape for
    Markdown, or should project reads return a more general item content
    structure with optional shapes?
11. Should SQLite content mapping support Markdown frontmatter inside the mapped
    text column, or should row attributes always be the only structured object
    for SQLite collections?
12. What should SQLite references display as in CLI output when there is no real
    filesystem path: `collection/id`, `table/id`, or a backend-specific URI?

## Documentation Updates

This work should update more than the glossary:

- `docs/content/reference/glossary.md`: attribute, field, item ID, content
  shape, document, and SQLite experimental status.
- `docs/content/deep-dives/core-concepts.md`: the backend-neutral distinction
  between attributes, fields, items, and content shapes.
- `docs/content/deep-dives/storage.md`: backend responsibilities, collection
  definition responsibilities, and how SQLite maps table rows to items.
- `docs/content/deep-dives/formatting.md`: keep Markdown frontmatter/body
  behavior scoped to the Markdown codec.
- `docs/content/reference/configuration.md`, or a split
  `docs/content/reference/configuration/` section: separate filesystem and
  SQLite collection examples.
- `docs/content/reference/check-types/` generated descriptions: replace
  frontmatter-only language where a check now applies to a structured attribute
  object.
- `docs/content/reference/inspectors/`: update `object_fields` and related
  inspector descriptions if the API moves from frontmatter to attributes.
- `docs/content/how-to/configure-rules.md`: show which checks work with which
  collection shapes.
- Root and per-package `AGENTS.md`: capture implementation conventions and point
  to the storage deep-dive, without duplicating the architecture narrative.

## Test Checklist

- Loading a SQLite collection with explicit single-column attributes succeeds.
- Loading a SQLite collection with a multi-column structured attribute succeeds
  and exposes nested fields to structured-object checks.
- Structured-object checks and JSON Schema checks pass/fail against SQLite
  attributes without requiring Markdown frontmatter.
- Unsupported filesystem checks fail at config or load time for SQLite
  collections.
- Plaintext and Markdown body-text checks fail at config or load time unless the
  SQLite collection declares a compatible content mapping.
- `item add`, `item update`, and `item delete` work for writable SQLite
  attributes, with clear errors for unsupported composite writes.
- `item get` can print SQLite attributes without calling them frontmatter.
- Inspectors can summarize SQLite attribute objects without needing a synthesized
  Markdown document.

## Rejected Alternatives

- **Treat every non-ID SQLite column as metadata.** This is convenient for a
  prototype, but "metadata" is too tied to parsed Markdown frontmatter in the
  current docs and implementation. SQLite needs an attribute capture model.
- **Keep `body:` as the SQLite text-column config key.** Rejected as the long
  term shape because body is a Markdown document part. A relational table can
  expose text content, but the collection should say that explicitly.
- **Gate checks only by backend type.** Rejected because the real compatibility
  boundary is the content/attribute shape a collection provides.
- **Make SQLite rows universal Markdown documents.** Rejected because it hides
  the architecture problem the SQLite backend is meant to surface.
