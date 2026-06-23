# Collection variants spec

> **Status: planning.** Implements issue #41. Lets one collection apply
> different checks to different items, selected by a **discriminator** — a
> predicate over an item's properties, not just its path. Builds on the storage
> layer (`product/specs/storage-layer-spec.md`, #31): collections live inside a
> StorageInstance's `collections:` block, and this spec adds an optional
> `variants:` layer whose discriminator is the portable `item list --filter`
> predicate grammar, with filesystem globs as a storage-type-scoped special
> case.

## Overview

A collection runs one check list against every item under it. There is no way to
require `weight` on content pages while exempting `_index.md`, or to demand a
kebab-case filename on content pages while letting `_index.md` keep its
underscore. This spec adds **variants**: an ordered list of discriminated check
groups inside a collection. An item runs the collection's base checks plus the
checks of the first variant whose discriminator it satisfies.

The discriminator is a predicate over the item's **properties** — primarily its
metadata (frontmatter fields for a file, column values for a future tabular
row), reusing the predicate grammar `item list --filter` already speaks. A path
glob is offered only as a filesystem-specific convenience, because a path is one
storage type's property, not a universal one. The item still belongs to exactly
one collection; only its check profile varies.

## Value

Strict, page-type-aware enforcement is the difference between "the docs have *a*
title" and "content pages are correctly ordered and section indexes are
correctly structured." Today the dogfooding config
(`.katalyst/storage/local.yaml`) ships one permissive `page` schema for the
whole `docs/content/` tree precisely because it cannot say "content pages
require `weight`, `_index.md` files do not, and the generated reference pages are
exempt from `requires_h1`." Those comments name #41 as the blocker. Closing the
gap lets the corpus be validated strictly against a green tree.

Choosing a property-predicate discriminator over a path glob also keeps the
feature aligned with the storage layer's reason for existing: a SQLite or
tabular backend (granularity `UnitIsCollection`, row = item) has no path to
glob, but its rows still carry discriminable column values. A glob-only design
would be dead on arrival the moment a second StorageType lands.

## Current State

After the storage refactor (#31), a collection is declared inside a storage
instance and carries one check profile:

- `internal/config/config.go:171` — `Collection{Name, Path, Dir, Pattern,
  Schema, Checks, Query, Storage}`. `Pattern` selects membership; `Schema`/
  `Checks` are the single profile applied to every item.
- `internal/config/config.go:253` — `rawCollection{Path, Pattern, Schema,
  Checks, Query}`, the YAML shape inside an instance's `collections:` block.
- `cmd/engine.go:73` — `checksFor(c, meta)` builds an item's runnable check list
  from `c.Checks`. The list is a function of the collection alone, not of the
  individual item.
- `internal/storage/filesystem.go:64` — `Unmatched(c)` walks `c.Dir` and flags
  every file failing `c.Pattern`.

Two reusable pieces already exist and shape this design:

- **A metadata predicate grammar.** `internal/query/filter.go` (`ParseFilter`,
  `Predicate.match`) parses and evaluates `item list --filter` expressions
  against an item's frontmatter map: `field=value`, `field!=value`,
  `field>=n`, `field=~regex`, `field=a,b` (in), bare `field` (exists), `!field`
  (absent), dot-paths into nested maps, array-contains semantics. It already
  operates on a plain `map[string]any`, so it is backend-agnostic. This is the
  discriminator grammar variants should reuse rather than inventing a second
  one. Its `filterTypeMismatch` setting (`config.QuerySettings`, the `query:`
  block) already defines what a type-incompatible comparison does (`skip` or
  `error`).
- **The storage seam and granularity.** `internal/storage/storage.go` defines
  `Granularity` (`FileIsItem` vs `UnitIsCollection`) and the closed
  `knownStorageTypes` allowlist (`config.go:61`). A path exists only for
  path-addressable types; the seam is where filesystem assumptions are supposed
  to live (the storage spec's AGENTS.md convention: "do not inline filesystem
  assumptions elsewhere").

The relevant invariant is `docs/content/deep-dives/domain-model.md:324`
(invariant #4): *"A collection owns its checks; an item belongs to one
collection. There is no glob-ordering 'first match wins'."* This spec relaxes
the **second clause** while keeping the first; see Design. The storage spec
itself retains *"a file belongs to exactly one collection"* and keeps a
file-in-many-collections out of scope — so the #41 gap must be closed *within*
one collection, not by overlapping two.

## Design

### Variants: discriminated check groups inside a collection

A collection gains an optional, ordered `variants:` list. Each variant has a
`when` discriminator and its own optional `schema`/`checks`.

```yaml
collections:
  pages:
    path: docs/content
    pattern: "**/*.md"        # membership: which items belong (per storage type)
    schema: page              # base: the title contract, every page type
    checks:
      - kind: filesystem_extension_in
        values: [.md]
    variants:
      - when:                 # section landing pages, by frontmatter
          where: ["kind=section"]
        schema: section_index
      - when:                 # generated reference pages, by path (fs-scoped)
          path: "reference/check-types/**/*.md"
        # exempt from requires_h1: adds nothing
      - when:                 # every other page is a content page
          where: ["kind!=section"]
        schema: content_page
        checks:
          - kind: object_required_field
            field: weight
          - kind: filesystem_name_case
            style: kebab
          - kind: markdown_requires_h1
```

### The discriminator: a property predicate, not a path

`when` holds one or more **conditions**, combined with AND (the variant matches
only when all pass). Two condition kinds:

- **`where:`** — a list of metadata predicates in the `item list --filter`
  grammar (`internal/query/filter.go`), evaluated against the item's metadata
  map. This is the **primary, portable** discriminator: every item, on every
  StorageType, yields a metadata map (frontmatter for a file, columns for a
  row). Multiple entries AND together. Type-incompatible comparisons follow the
  collection's resolved `filterTypeMismatch` (`skip` means the condition simply
  fails to match; `error` aborts the run), reusing the exact semantics
  `item list --filter` already documents.
- **`path:`** — a doublestar glob over the item's backend reference, **scoped to
  path-addressable StorageTypes** (filesystem today). It is sugar for the common
  "by location" case that a metadata predicate expresses clumsily. On a backend
  with no path (e.g. tabular `UnitIsCollection`), a `path:` condition is a config
  error, caught at load time against the instance's declared `type`.

A bare-glob convenience shorthand — `when: "reference/**/*.md"` desugaring to
`when: { path: ... }` — is available so the common filesystem case stays terse.

Keeping the path selector in its own `path:` key (rather than as a reserved
field inside the metadata namespace) avoids colliding with a real frontmatter
key named `path`, and fences every storage-type-specific selector into one
clearly backend-scoped spot. See Rejected alternatives for the "merge path into
the metadata grammar as an intrinsic field" option and why it is deferred.

### Resolution semantics

For each item:

1. **Membership is unchanged.** A file is an item iff it matches the
   collection's `pattern` (or, per granularity, the backend's unit rule). Items
   that miss membership are still `Unmatched` (invariant #6 holds). Variants
   never widen or narrow membership; they only route checks.
2. **First match wins among variants.** The item is routed to the *first*
   variant whose `when` matches, in declaration order. At most one variant
   applies. An item matching no variant runs the base profile only.
3. **Base + variant compose additively.** Effective checks = the collection's
   base `schema`/`checks` (always) followed by the matched variant's
   `schema`/`checks`. Object schemas are additive: an item validated against
   both a base `page` schema and a variant `content_page` schema must satisfy
   both, so a variant *adds* required structure rather than restating the base.

Additive base-plus-variant is what makes **exemption** work without negation: a
check lives in a variant, not the base, exactly when some item type must *not*
run it. `_index.md` is exempt from `weight`/kebab/`requires_h1` because those
checks live in the `content_page` variant and `_index.md` is routed to the
`section_index` variant first.

### Discriminators are storage-type-scoped

Which condition kinds a variant may use depends on the StorageType, exactly as
membership and granularity already do:

| Condition | Availability |
|---|---|
| `where:` (metadata predicate) | **All** StorageTypes — every item yields a metadata map. |
| `path:` (glob over the reference) | **Path-addressable** types only (filesystem). |

`config` already carries a closed `knownStorageTypes` allowlist (`config.go:61`)
because it cannot import `internal/storage`. This spec adds a parallel,
config-local set of path-addressable types (currently `{filesystem}`) used to
validate `path:` conditions at load time. When a non-filesystem backend lands,
it declares its capability in the storage registry and the config mirror, the
same one-line extension the storage spec established.

### Where the routing lives, and the seam

Variant routing decides *which checks an item runs*, so it lives in the check
engine (`cmd/engine.go`), not the storage mapping. But the two condition kinds
evaluate in different places to keep the seam closed:

- **`where:` conditions** are evaluated by the engine over `Item` metadata. The
  engine already parses each item's frontmatter (`checkItem`), and `query`
  exposes predicate matching over a `map[string]any` — no filesystem assumption
  enters the engine.
- **`path:` conditions** are evaluated by the storage `CollectionDefinition`,
  which already performs doublestar matching in `Items`/`Unmatched`. Evaluating
  a path glob there (rather than `doublestar.Match` on `Item.ID` in the engine)
  honors the storage spec's convention that path semantics stay behind the seam.
  The exact touchpoint — a `Matches(c, ref, glob)` method versus the definition
  pre-tagging each `Item` with the variant indices its reference satisfies — is
  Open Question 3.

Config changes:

- `internal/config`: add `Variants []CollectionVariant` to `Collection` and
  `variants:` to `rawCollection`. `CollectionVariant` is
  `{When Discriminator, Schema string, Checks []CheckInstance}`, where
  `Discriminator` is `{Where []query.Predicate, Path string}` (or the parsed
  forms). Validate each `Where` entry through `query.ParseFilter`, each `Path`
  as a compilable doublestar glob *and* only on a path-addressable instance
  type, and `Schema`/`Checks` through the existing `normalizeCheck` path.
- `cmd/engine.go`: `checksFor` gains the item (for metadata and reference) so it
  can select the matched variant and append its checks to the same compile
  `switch` that handles `c.Checks`.
- `internal/storage` is otherwise untouched: membership, `Unmatched`,
  `Reference`, granularity are unaffected.

### Granularity makes the case concrete

For `FileIsItem` (filesystem markdown), an item's properties are its frontmatter
plus a path; both `where:` and `path:` apply. For a future `UnitIsCollection`
(tabular) backend, a row item has column values but no path: `where:`
discriminates naturally (`where: ["status=published"]`), and `path:` is rejected
at config time. This is the "Item and Collection are roles, not file counts"
principle from the domain model, now extended to discrimination: a variant
selects items by *what they are*, and only optionally by *where they sit*.

### Invariant #4 is revised, not broken

Invariant #4 (`domain-model.md:324`) gets reworded. The first clause holds: an
item belongs to exactly one collection (variants live inside one collection;
nothing overlaps). The second clause — "no first match wins" — changes in a
contained way: first-match-wins applies only *within* a collection, among its
declared variants, never across collections. The predictability the original
invariant bought (an item's collection is unambiguous) is preserved; what
becomes item-dependent is only *which subset of one collection's checks* an item
runs, and that is explicit and ordered in one file. The reworded invariant and
its rationale graduate into `domain-model.md` when this ships.

## Open Questions

1. **Discriminator keyword shape.** Recommended: a `when:` block holding
   `where:` (metadata predicates, the `--filter` grammar) and/or `path:` (a
   filesystem glob), AND-combined, plus a bare-string `when: "<glob>"`
   shorthand. Alternatives weighed: a flat `match:` glob (the rejected
   glob-only design); `if:`/`unless:`; folding everything into one `where:` list
   with a reserved `@path` field (collision risk — see Rejected alternatives).
   Maintainer's call on the exact keys.
2. **No-variant-match behavior.** An item matching `pattern` but no variant runs
   the base profile only (recommended: lenient; pair with a trailing catch-all
   variant when a project wants every item covered). The strict alternative —
   error on an unrouted item — is heavier and can come later behind a collection
   flag.
3. **Native-condition evaluation mechanism.** How the engine learns whether a
   `path:` condition matched without inlining path logic: a seam method
   (`CollectionDefinition.Matches(c, ref, glob)`) versus the definition
   pre-tagging each `Item` with the variant indices its reference satisfies.
   Pre-tagging keeps all doublestar use inside `internal/storage` and avoids a
   per-item seam call; settle in the plan.
4. **Condition combinators.** `where:` entries AND together and `where:`+`path:`
   AND together. Whether OR / NOT across conditions is needed now (e.g. a
   variant for "drafts or future-dated") or deferred until a real case appears.
   Recommended: AND-only now; the `--filter` grammar already gives per-field
   negation (`!=`, `!field`), which covers most needs.

## Documentation updates

**User docs (Hugo, `docs/content/`):**

- `reference/configuration.md`: document `variants:` — the `when`
  (`where`/`path`) discriminator, that `where` is the `item list --filter`
  grammar and `path` is filesystem-scoped, first-match-wins ordering, additive
  base-plus-variant composition, and that membership/`Unmatched` stay governed
  by `pattern`. Note in the object-schema precedence section that a variant
  schema joins the configured tier.
- `how-to/configure-rules.md`: add the `_index.md`-vs-content-page worked
  example as the motivating case, showing both a `where:` and a `path:` variant.
- `reference/commands.md` / the `item list --filter` reference: cross-link the
  shared predicate grammar so users learn it once.
- `deep-dives/domain-model.md`: reword invariant #4; fold in the
  within-collection first-match-wins rationale.
- `deep-dives/storage.md`: note that variants discriminate items by properties,
  that `where:` is portable across StorageTypes while `path:` is path-addressable
  only, and that native (path) conditions evaluate behind the seam.
- `reference/glossary.md`: add **Variant** (a discriminated check group within a
  collection) and **Discriminator** (the predicate selecting a variant).

**Developer docs:**

- `internal/config/README.md`: document `variants:`, its validation, the
  path-addressable-type gate, and that it is a check-routing concern.
- `internal/query/` package doc: note the predicate grammar is reused by variant
  discriminators, not only `item list --filter`.
- `cmd/engine.go` / `internal/config/config.go` doc comments: variant selection
  and additive composition.
- `AGENTS.md`: record that per-item check *routing* lives in the engine, while
  membership, locators, and **native (path) discriminators** stay in
  `internal/storage`.

**Specs / config (cross-references):** update `.katalyst/storage/local.yaml` to
use `variants` (retiring the `# issue #41` workaround comments) and note in
`product/specs/dogfood-docs-spec.md` that per-page-type enforcement is unblocked.

## Rejected alternatives

- **Glob-only discriminator (`match: <glob>`).** The first draft of this spec.
  Rejected on the maintainer's point: a glob is a filesystem property, and the
  storage layer exists so a second StorageType (SQLite, granularity
  `UnitIsCollection`) can land — a backend with rows and no paths. A glob-only
  discriminator could not select those items at all. The metadata predicate is
  portable; the glob survives only as a path-addressable convenience.
- **Glob negation / exclude + multiple collections.** Add `exclude:` to a
  collection and model each page type as its own collection over the same tree.
  Rejected: it double-owns the tree, violating the storage spec's retained
  invariant #4, and each collection's `Unmatched` flags the other's files.
  Variants keep one collection, one membership pass, one owner per item.
- **Conditional (`oneOf`/`if-then`) JSON Schema as the whole answer.** Covers
  only frontmatter validation, leaving markdown/filesystem checks
  (`requires_h1`, kebab filename) uniform and unable to exempt `_index.md` from a
  filename check. It composes fine *inside* a variant's `schema`, so variants and
  conditional schemas cooperate rather than compete.
- **Merge `path` into the metadata grammar as an intrinsic field.** Have the
  filesystem backend contribute reserved fields (`@path`, `@name`, `@ext`) so a
  single `where:` grammar covers everything (`@path=~^reference/`). Attractive
  (one mechanism) but deferred: it needs a reserved namespace to avoid colliding
  with real frontmatter keys, globs read better than regex for paths, and it
  spreads backend-specific field names into the portable grammar. Keeping
  `path:` as a separate, clearly fenced, storage-type-scoped key is simpler now;
  the intrinsic-field unification can subsume it later without breaking configs.
