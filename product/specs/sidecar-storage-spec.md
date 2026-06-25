# Spec — sidecar storage

> **Status: planning.** Covers issue #102: stress-test the codec architecture by
> adding JSON and SQLite sidecars to filesystem-backed collections.

## Overview

Standalone SQLite proved that a Katalyst collection can be backed by something
other than a directory of Markdown files. Sidecars test a different pressure
point: one Katalyst item assembled from a primary store plus auxiliary storage.

The first sidecar target is a filesystem Markdown collection where the Markdown
file remains the primary item, and JSON files or SQLite rows provide additional
attributes for the same item:

```text
notes/dune.md
notes/dune.json
```

or:

```text
notes/dune.md
.katalyst/enrichments.sqlite  # row keyed by "dune"
```

Checks and inspectors should still receive the item content shapes they already
understand: a Markdown document with an attribute object and body text. They
should not know whether a field came from frontmatter, a JSON file, or a SQLite
row unless a diagnostic or write path explicitly exposes provenance.

## Value

Many knowledge bases keep authored prose in Markdown while keeping structured
data nearby: generated classification labels, review status, catalog metadata,
API enrichment, or migration state. Forcing all of that into frontmatter is
awkward and sometimes wrong. Sidecars let users validate the whole item without
collapsing every storage concern into the Markdown file.

Architecturally, sidecars are a stronger codec/storage stress test than
standalone SQLite:

- The primary collection still owns item membership and item identity.
- Auxiliary storage must be resolved from that identity.
- Auxiliary payloads must be decoded and merged without making checks
  backend-aware.
- Writes become source-specific: updating a sidecar field is not the same as
  updating Markdown frontmatter.

## Current State

The post-SQLite `main` branch has these relevant pieces:

- `internal/storage/collection.CollectionDefinition` owns collection/item
  discovery and reverse references: `Collections`, `Items`, `Unmatched`, and
  `Reference`.
- `internal/storage/collection.Collection` is the loaded collection
  configuration. It currently contains filesystem fields (`Path`, `Dir`,
  `Pattern`) and SQLite fields (`Table`, `IDColumn`, `Attributes`,
  `ContentKind`, `ContentColumn`).
- `internal/project.Project.ReadItem` is where item bytes are actually read and
  decoded. Filesystem items are read directly with `os.ReadFile` and
  `markdownbodytext.Parse`; SQLite items call `sqlite.Definition.Read`.
- `internal/project.ItemContent` is still Markdown-shaped: `Raw []byte` plus
  `Doc *markdownbodytext.Document`.
- `internal/checks.Context` and collection inspectors consume `Doc.Meta` /
  `Doc.Body`, even though user-facing docs now prefer attributes/content where
  the backend is not specifically Markdown.
- `internal/storage/collection/sqlite` already knows how to map SQLite columns
  into attributes and optional content for standalone SQLite collections.
- There is no concept of an item being composed from more than one storage
  reference.

The main architectural gap is that the collection seam maps identity, while the
project layer still owns primary content reading for filesystem collections.
Sidecar composition will start there unless we first move content reading behind
the collection definition interface.

## Goals

- Support JSON sidecars attached to filesystem Markdown collections.
- Support SQLite sidecars attached to filesystem Markdown collections.
- Let `check`, `item list --filter`, and `inspect object_fields` see
  sidecar-supplied attributes.
- Keep checks and inspectors unaware of sidecar backends.
- Define deterministic merge behavior, including collisions and missing
  sidecars.
- Make sidecars read-only in the first cut.

## Non-Goals

- Do not make sidecars independent collections.
- Do not support sidecars attached to standalone SQLite collections in the first
  cut.
- Do not make `fix` or `item update` write sidecars in the first cut.
- Do not expose provenance in check APIs until a concrete diagnostic or write
  path requires it.
- Do not implement general multi-coordinate templates beyond the item ID unless
  the implementation needs them for the first JSON/SQLite sidecar examples.

## Domain Model

### Primary collection

The primary collection owns membership, item identity, and selector behavior.
In the first cut, this is a filesystem Markdown collection:

```yaml
collections:
  notes:
    path: notes
    checks:
      - kind: object_required_field
        field: catalog.status
```

`notes/dune.md` is still the primary item. `notes/dune.json` or a SQLite row may
augment it, but neither creates a second item.

### Sidecar

A sidecar is auxiliary storage attached to each item in a primary collection. A
sidecar definition answers:

1. How to find the auxiliary reference for an item.
2. How to decode that auxiliary reference into attributes.
3. How to merge those attributes into the primary item.
4. Whether the sidecar is required.
5. Whether any write operation may modify it.

### Sidecar source

A sidecar source is one configured auxiliary backend:

- JSON sidecar: one JSON object file per primary item.
- SQLite sidecar: one row per primary item, keyed by item ID.

The term "source" is intentionally narrower than `StorageInstance`. A sidecar
source does not declare collections or item membership; it hangs off a primary
collection.

### Composed item content

The composed item content is what checks and inspectors see after primary
content and sidecars have been decoded and merged. Today this will still be
represented as `markdownbodytext.Document.Meta` plus `Body`, but the durable
concept is:

- an attribute object;
- optional content shapes, starting with Markdown body text;
- optional internal provenance.

## Design

### Config shape

Add `sidecars:` under a collection:

```yaml
type: filesystem
root: .
collections:
  notes:
    path: notes
    sidecars:
      catalog:
        type: json
        path: "{id}.json"
        merge:
          namespace: catalog
        required: false
      enrichments:
        type: sqlite
        path: .katalyst/enrichments.sqlite
        table: note_enrichments
        key: slug
        attributes:
          sentiment: sentiment
          reviewer:
            columns:
              name: reviewer_name
              team: reviewer_team
        merge:
          namespace: enrichment
        required: true
    checks:
      - kind: object_required_field
        field: catalog.status
      - kind: object_required_field
        field: enrichment.sentiment
```

Open syntax questions remain, but the first design should preserve these
properties:

- sidecar names are stable and appear in diagnostics;
- JSON sidecars use a path template based on item identity;
- SQLite sidecars use the same `attributes:` capture model as standalone
  SQLite;
- merge behavior is explicit;
- `required` is explicit.

### JSON sidecars

A JSON sidecar decodes one JSON object per item. The first supported lookup is a
path template relative to the primary item's directory:

```yaml
sidecars:
  catalog:
    type: json
    path: "{id}.json"
```

For `notes/dune.md`, this resolves to `notes/dune.json`.

A second supported lookup can be a sidecar root plus template:

```yaml
sidecars:
  catalog:
    type: json
    root: .katalyst/catalog
    path: "notes/{id}.json"
```

JSON sidecars must decode to an object. Arrays, strings, and scalar JSON values
are invalid unless a future config explicitly maps them into an attribute.

### SQLite sidecars

A SQLite sidecar decodes one row per item from a configured table:

```yaml
sidecars:
  enrichments:
    type: sqlite
    path: .katalyst/enrichments.sqlite
    table: note_enrichments
    key: slug
    attributes:
      sentiment: sentiment
      confidence: confidence
```

For item `notes/dune`, Katalyst queries:

```sql
SELECT * FROM note_enrichments WHERE slug = ?
```

with `dune` as the key. The row is decoded with the same attribute capture rules
as standalone SQLite. The first cut should reject duplicate matching rows and
report a clear diagnostic.

### Merge policy

Sidecars merge decoded attributes into the primary attribute object.

Support two policies:

```yaml
merge:
  namespace: catalog
```

and:

```yaml
merge:
  topLevel: true
```

Namespaced merge is the default recommendation. It puts the whole decoded
sidecar object under one attribute:

```yaml
catalog:
  status: published
  source: library
```

Top-level merge is allowed only when it has deterministic collision behavior.
If any sidecar field collides with an existing primary or earlier sidecar field,
loading or reading the item fails with a message that names both sources. No
sidecar silently overwrites primary frontmatter.

Sidecars are merged in config order. If YAML map ordering is not reliable enough
for this contract, make `sidecars:` a list instead of a map.

### Missing sidecars

Missing behavior is explicit:

```yaml
required: true
```

- Missing required sidecar: item read fails with a storage diagnostic. `check`
  reports the item as an error.
- Missing optional sidecar: item read succeeds with only primary content and any
  present sidecars.

Invalid sidecar content is always an error when the sidecar file/row exists,
even when `required: false`.

### Read path

The first implementation can compose sidecars in `internal/project.ReadItem`:

1. Resolve and read the primary item exactly as today.
2. Decode primary content into `ItemContent`.
3. For each sidecar definition on the collection, resolve the sidecar reference
   using the item ID and primary reference.
4. Decode sidecar attributes.
5. Merge into the document attribute object.
6. Return the composed `ItemContent` to callers.

This is a pragmatic starting point because `ReadItem` is already where
filesystem content is read. It should be treated as a bridge, not necessarily
the final architecture. If the implementation starts duplicating backend logic,
extract a `sidecar` package or move content reads behind a richer collection
interface.

### Write path

Sidecars are read-only in the first cut. This affects commands:

- `check`: reads sidecars and validates composed attributes.
- `inspect`: reads sidecars and includes composed attributes in evidence.
- `item list --filter`: filters against composed attributes.
- `item get --attributes`: prints composed attributes.
- `item update`: may update primary frontmatter fields only. If the user tries
  to update a namespaced sidecar field, return a clear read-only sidecar error.
- `fix`: does not modify sidecar fields. If a future fix would need to modify a
  sidecar-owned field, report that the field is read-only.

The first implementation may avoid provenance by making sidecars read-only only
at namespace boundaries: any assignment under a configured sidecar namespace is
rejected. Top-level sidecar merge probably requires provenance to reject writes
correctly.

### Provenance

Preserve provenance internally if it is cheap:

```text
/catalog/status -> sidecar catalog (json notes/dune.json)
/title          -> primary markdown frontmatter
```

Do not expose provenance to checks yet. It is mainly useful for diagnostics:
collision errors, missing sidecars, and write attempts against read-only
sidecar fields.

## Architecture Impact

### `internal/storage/collection`

Add sidecar config types near `Collection` / `RawCollection`, because sidecars
are declared on a collection. This package should validate config shape and
check-family compatibility, but should not import concrete sidecar IO packages.

### `internal/project`

`Project.ReadItem` is the likely first composition point. It already selects the
backend read path and returns the `ItemContent` checks consume. The spec should
not hide that this is a mixed concern today: project is orchestrating storage,
codec decode, and now composition.

### `internal/codec`

Add a structured JSON object decoder only if it will be reused outside sidecars.
Otherwise JSON sidecar decoding can start in the sidecar reader and move later.

SQLite row-to-attribute capture can reuse the concepts introduced for
standalone SQLite, but probably should not import the standalone
`sqlite.Definition` directly if that drags in collection-level semantics that a
sidecar does not need.

### `internal/checks` and `internal/inspect`

No direct sidecar imports. They consume composed attributes through the existing
item content path.

## Open Questions

1. **Should `sidecars:` be a map or a list?** A map is ergonomic; a list gives
   deterministic merge order without relying on YAML node ordering.
2. **Is `Project.ReadItem` the right first composition point?** It is practical
   today, but it keeps content reading outside `CollectionDefinition`. Should
   this spec first introduce a richer storage read interface?
3. **Should JSON sidecar path templates support only `{id}` or also future
   coordinates?** Starting with `{id}` is enough for flat filesystem items, but
   richer layouts will want coordinate templates.
4. **Should missing optional SQLite rows be equivalent to missing optional JSON
   files?** The likely answer is yes, but diagnostics may need to distinguish
   "database/table missing" from "row missing."
5. **Should `required` default to false or true?** Optional is friendlier for
   enrichment sidecars; required is safer for validation sidecars.
6. **Is namespaced merge required, or merely recommended?** Requiring namespace
   first avoids provenance and write-path ambiguity. Allowing top-level merge
   sooner improves ergonomics.
7. **How should collisions be reported with multiple sidecars?** The message
   should name the field, the earlier source, and the later source.
8. **What does `item get` print by default?** Today it prints raw primary
   content. Should composed attributes appear only under `--attributes`, or
   should default output remain the primary Markdown document?
9. **Should `item list --grep-in attributes` search sidecar JSON text or the
   merged YAML rendering?** Filtering should use structured attributes; grep
   needs a stable byte representation.
10. **Do sidecar-supplied attributes participate in variant routing?** Likely
    yes for read-time checks, but this means variants cannot be resolved until
    after sidecar composition.
11. **Do sidecars have `Unmatched` equivalents?** JSON directories may contain
    stray sidecar files; SQLite tables may contain rows for missing primary
    items. The first cut can defer this, but it is a natural `doctor` concern.
12. **How much provenance should be stored now?** Namespaced read-only behavior
    can work without full per-field provenance; top-level merge and good write
    diagnostics probably need it.

## Documentation Updates

- `docs/content/deep-dives/storage.md`: define sidecars as auxiliary storage
  attached to a primary collection; explain identity, merge policy, and
  read-only first cut.
- `docs/content/reference/configuration.md`: document `sidecars:` config for
  JSON and SQLite.
- `docs/content/deep-dives/core-concepts.md`: clarify primary item, auxiliary
  attributes, and composed item content if user-visible.
- `docs/content/reference/glossary.md`: add sidecar, primary collection, merge
  policy, and provenance if those terms ship.
- `docs/content/how-to/configure-rules.md`: show checks over sidecar-supplied
  attributes.
- `internal/storage/collection/AGENTS.md`: add conventions for sidecar config
  and composition ownership.
- Add package `AGENTS.md` files if JSON or SQLite sidecar readers get their own
  package.

## Test Checklist

- [ ] Filesystem Markdown collection can configure a required JSON sidecar.
- [ ] Filesystem Markdown collection can configure an optional JSON sidecar.
- [ ] Filesystem Markdown collection can configure a required SQLite sidecar.
- [ ] Filesystem Markdown collection can configure an optional SQLite sidecar.
- [ ] `check` can validate a namespaced field supplied by a JSON sidecar.
- [ ] `check` can validate a namespaced field supplied by a SQLite sidecar.
- [ ] `item list --filter` can filter on a sidecar-supplied field.
- [ ] `inspect object_fields` includes sidecar-supplied fields.
- [ ] `item get --attributes` prints composed attributes.
- [ ] Missing required sidecars fail with clear diagnostics.
- [ ] Missing optional sidecars do not fail item reads.
- [ ] Invalid JSON sidecars fail clearly.
- [ ] SQLite sidecars with zero matching rows obey `required`.
- [ ] SQLite sidecars with more than one matching row fail clearly.
- [ ] Top-level merge collisions fail deterministically.
- [ ] Sidecar read-only behavior is explicit for `item update` and `fix`.
- [ ] Checks and inspectors do not import sidecar backend packages.
- [ ] `go test ./...` passes.

## Rejected Alternatives

- **Treat every sidecar as its own collection.** Rejected because a sidecar is
  auxiliary data for an existing item identity, not a second set of items.
- **Inline sidecar data into Markdown frontmatter before checks run.** Rejected
  as a user-visible model because it erases source, collisions, and write-path
  ownership. Synthesizing `Doc.Meta` may remain an implementation bridge.
- **Let checks read sidecars directly.** Rejected because it would make check
  families backend-aware and punch through the storage/codec boundary.
- **Support only JSON sidecars first.** Rejected as the whole design because
  SQLite is the better architecture stress test: keyed lookup, row decoding,
  missing-row behavior, and non-file references.
- **Silently overwrite primary fields with sidecar fields.** Rejected because
  it makes validation nondeterministic and hides source-of-truth mistakes.
