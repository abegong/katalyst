# Collection variants spec

> **Status: planning.** Implements issue #41. Lets one collection apply
> different checks to different items, selected by a **discriminator** — a
> predicate over an item's metadata. Builds on the storage layer
> (`product/specs/storage-layer-spec.md`, #31): collections live inside a
> StorageInstance's `collections:` block, and this spec adds an optional
> `variants:` layer whose discriminator reuses the portable `item list --filter`
> predicate grammar. Path-based discrimination is deferred so v1 stays
> backend-agnostic and leaves the storage seam untouched.

## Overview

A collection runs one check list against every item under it. There is no way to
require `weight` on content pages while exempting `_index.md`, or to demand a
kebab-case filename on content pages while letting `_index.md` keep its
underscore. This spec adds **variants**: an ordered list of discriminated check
groups inside a collection. An item runs the collection's base checks plus the
checks of the first variant whose discriminator it satisfies.

The discriminator is a predicate over the item's **metadata** — frontmatter
fields for a file, column values for a future tabular row — reusing the grammar
`item list --filter` already speaks. Metadata is the one property every item
yields on every backend, so the discriminator is portable by construction. The
item still belongs to exactly one collection; only its check profile varies.

## Value

Strict, page-type-aware enforcement is the difference between "the docs have *a*
title" and "content pages are correctly ordered and section indexes are
correctly structured." Today the dogfooding config
(`.katalyst/storage/local.yaml`) ships one permissive `page` schema for the
whole `docs/content/` tree precisely because it cannot say "content pages
require `weight`, section `_index.md` files do not." Variants close that gap for
every page type distinguishable by frontmatter — section indexes carry
`bookCollapseSection`, so a `section_index` variant and a `content_page` variant
can finally diverge (see Design; the purely path-distinguished cases are a known
limit, below).

Choosing a metadata predicate over a path glob keeps the feature aligned with
the storage layer's reason for existing: a SQLite or tabular backend
(granularity `UnitIsCollection`, row = item) has no path to glob, but its rows
still carry discriminable column values. A glob-based discriminator would be dead
on arrival the moment a second StorageType lands; a metadata one works on every
backend, and needs no change to `internal/storage` at all.

## Current State

After the storage refactor (#31), a collection is declared inside a storage
instance and carries one check profile:

- `internal/config/config.go:180` — `Collection{Name, Path, Dir, Pattern,
  Schema, Checks, Query, Storage}`. `Pattern` selects membership; `Schema`/
  `Checks` are the single profile applied to every item.
- `internal/config/config.go:262` — `rawCollection{Path, Pattern, Schema,
  Checks, Query}`, the YAML shape inside an instance's `collections:` block.
- `cmd/engine.go:73` — `checksFor(c, meta)` builds an item's runnable check list
  from `c.Checks`. The list is a function of the collection alone, not of the
  individual item. It already receives the item's parsed `meta` — the input a
  metadata discriminator needs.
- `internal/storage/filesystem.go:64` — `Unmatched(c)` walks `c.Dir` and flags
  every file failing `c.Pattern`. Unaffected by this spec.

One reusable piece is central to the design:

- **A metadata predicate grammar.** `internal/query/filter.go` (`ParseFilter`,
  `Predicate.match`) parses and evaluates `item list --filter` expressions
  against an item's frontmatter map: `field=value`, `field!=value`,
  `field>=n`, `field=~regex`, `field=a,b` (in), bare `field` (exists), `!field`
  (absent), dot-paths into nested maps, array-contains semantics. It already
  operates on a plain `map[string]any`, so it is backend-agnostic. Variants
  reuse this grammar rather than inventing a second one. Its
  `filterTypeMismatch` setting (`config.QuerySettings`, the `query:` block)
  already defines what a type-incompatible comparison does (`skip` or `error`).

The relevant invariant is `docs/content/deep-dives/domain-model.md:324`
(invariant #4): *"A collection owns its checks; an item belongs to one
collection. There is no glob-ordering 'first match wins'."* This spec relaxes
the **second clause** while keeping the first; see Design. The storage spec
itself retains *"a file belongs to exactly one collection"* and keeps a
file-in-many-collections out of scope — so the #41 gap must be closed *within*
one collection, not by overlapping two.

## Design

### Variants: discriminated check groups inside a collection

A collection gains an optional, ordered `variants:` list and an opt-in
`useExhaustiveVariants:` flag. Each variant has a `when` discriminator and its
own optional `schema`/`checks`.

```yaml
collections:
  pages:
    path: docs/content
    pattern: "**/*.md"             # membership: which items belong
    schema: page                   # base: the title contract, every page type
    checks:
      - kind: filesystem_extension_in
        values: [.md]
    useExhaustiveVariants: false   # default; an unrouted item runs base only
    variants:
      - when:                      # section landing pages carry this flag
          where: ["bookCollapseSection"]
        schema: section_index
      - when:                      # every other page is a content page
          where: ["!bookCollapseSection"]
        schema: content_page
        checks:
          - kind: object_required_field
            field: weight
          - kind: markdown_requires_h1
```

### The discriminator: a metadata predicate

`when` holds a `where:` list of metadata predicates in the `item list --filter`
grammar (`internal/query/filter.go`), evaluated against the item's metadata map.
Multiple entries AND together; the variant matches only when all pass.
Type-incompatible comparisons follow the collection's resolved
`filterTypeMismatch` (`skip` means the predicate fails to match; `error` aborts
the run), reusing the exact semantics `item list --filter` documents.

`when` accepts a string or list shorthand that desugars to `where:`, so the
common case stays terse:

```yaml
when: "bookCollapseSection"            # → when: { where: ["bookCollapseSection"] }
when: ["kind=section", "weight>=1"]    # → when: { where: [...] }
```

`when` is a **block, not a bare predicate list**, so a future condition kind
(notably `path:`, Deferred below) slots in as a new key without breaking
existing configs — the same forward-compatible shape the storage spec used for
its discovery blocks.

### Resolution semantics

For each item:

1. **Membership is unchanged.** A file is an item iff it matches the
   collection's `pattern` (or, per granularity, the backend's unit rule). Items
   that miss membership are still `Unmatched` (invariant #6 holds). Variants
   never widen or narrow membership; they only route checks.
2. **First match wins among variants.** The item is routed to the *first*
   variant whose `when` matches, in declaration order. At most one variant
   applies.
3. **Unrouted items follow `useExhaustiveVariants`.** An item matching no
   variant runs the base profile only when the flag is `false` (the default);
   when `true`, an unrouted item is a check failure
   (`<file>: matches no variant`), so a collection can prove every member is
   accounted for. The flag is a per-collection boolean, resolved like the rest
   of the collection config.
4. **Base + variant compose additively.** Effective checks = the collection's
   base `schema`/`checks` (always) followed by the matched variant's
   `schema`/`checks`. Object schemas are additive: an item validated against
   both a base `page` schema and a variant `content_page` schema must satisfy
   both, so a variant *adds* required structure rather than restating the base.

Additive base-plus-variant is what makes **exemption** work without negation: a
check lives in a variant, not the base, exactly when some item type must *not*
run it. Section `_index.md` files are exempt from `weight`/`requires_h1` because
those checks live in the `content_page` variant and the `bookCollapseSection`
predicate routes index pages to `section_index` first.

### Where the routing lives

Variant routing decides *which checks an item runs*, and — because the
discriminator reads only metadata — it lives entirely in the check engine
(`cmd/engine.go`). The engine already parses each item's frontmatter
(`checkItem`) and `query` already exposes predicate matching over a
`map[string]any`, so routing is: evaluate each variant's `where:` against the
item's `meta`, take the first match, append its checks. **`internal/storage` is
untouched** — no seam method, no per-item storage call. Keeping path out of v1
(Deferred) is what buys this simplicity.

Config changes:

- `internal/config`: add `Variants []CollectionVariant` and
  `UseExhaustiveVariants bool` to `Collection`, and `variants:` /
  `useExhaustiveVariants:` to `rawCollection`. `CollectionVariant` is
  `{Where []query.Predicate, Schema string, Checks []CheckInstance}`. Validate
  each `Where` entry through `query.ParseFilter` and `Schema`/`Checks` through
  the existing `normalizeCheck` path.
- `cmd/engine.go`: `checksFor` selects the matched variant from the item's
  `meta` and appends its checks to the same compile `switch` that handles
  `c.Checks`; an unrouted item under `UseExhaustiveVariants` yields a violation.

### Granularity makes the case concrete

For `FileIsItem` (filesystem markdown), an item's metadata is its frontmatter;
`where:` discriminates on it. For a future `UnitIsCollection` (tabular) backend,
a row item's metadata is its column values; the *same* `where:` grammar
discriminates (`where: ["status=published"]`) with no new code. This is the
"Item and Collection are roles, not file counts" principle from the domain
model, extended to discrimination: a variant selects items by *what they are*,
uniformly across backends.

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

### Deferred: path-based discrimination

A variant cannot select items by location in v1. The `when` block is shaped to
accept a `path:` glob later (a storage-type-scoped condition, valid only on
path-addressable backends and evaluated behind the `internal/storage` seam), but
that condition — and the seam mechanism it needs — is out of scope here. The
consequence is concrete and worth stating: page types distinguished **only** by
path, not frontmatter, cannot be discriminated yet. In the dogfood corpus the
generated reference pages under `reference/check-types/**` are exactly this case
— they carry no frontmatter marker, so a `content_page` variant's
`requires_h1` would still flag them. Closing that needs either the deferred
`path:` condition or a frontmatter marker (e.g. a `generated: true` key the
`make docs-gen` step writes), tracked as follow-up. v1 ships the portable
mechanism; the locational case follows.

## Open Questions

_None._ The four drafting questions are resolved and folded into Design:

1. **Discriminator shape** → a `when:` block holding a `where:` predicate list,
   with string/list shorthand ("The discriminator").
2. **No-variant-match behavior** → the `useExhaustiveVariants` flag, default
   `false` (lenient), opt-in strict (Resolution semantics §3).
3. **Path-condition evaluation** → resolved by taking `path:` **out of scope**;
   v1 is metadata-only and the storage seam is untouched ("Where the routing
   lives," "Deferred").
4. **Combinators** → AND-only; `where:` entries AND together, and the
   `--filter` grammar already gives per-field negation (`!=`, `!field`). OR/NOT
   revisited if a real disjunctive case appears.

## Documentation updates

**User docs (Hugo, `docs/content/`):**

- `reference/configuration.md`: document `variants:` and
  `useExhaustiveVariants:` — the `when`/`where` discriminator as the
  `item list --filter` grammar, first-match-wins ordering, additive
  base-plus-variant composition, the unrouted-item behavior, and that
  membership/`Unmatched` stay governed by `pattern`. Note in the object-schema
  precedence section that a variant schema joins the configured tier.
- `how-to/configure-rules.md`: add the section-index-vs-content-page worked
  example as the motivating case.
- `reference/commands.md` / the `item list --filter` reference: cross-link the
  shared predicate grammar so users learn it once.
- `deep-dives/domain-model.md`: reword invariant #4; fold in the
  within-collection first-match-wins rationale.
- `deep-dives/storage.md`: note that variants discriminate items by metadata
  (portable across StorageTypes) and that path-based discrimination is deferred.
- `reference/glossary.md`: add **Variant** (a discriminated check group within a
  collection) and **Discriminator** (the metadata predicate selecting a variant).

**Developer docs:**

- `internal/config/README.md`: document `variants:` /
  `useExhaustiveVariants:`, their validation, and that variants are a
  check-routing concern (not a storage/membership one).
- `internal/query/` package doc: note the predicate grammar is reused by variant
  discriminators, not only `item list --filter`.
- `cmd/engine.go` / `internal/config/config.go` doc comments: variant selection
  and additive composition.
- `AGENTS.md`: record that per-item check *routing* lives in the engine
  (`checksFor`), keyed on item metadata.

**Specs / config (cross-references):** update `.katalyst/storage/local.yaml` to
add the section-index/content-page variants (the `_index.md` split the `# issue
#41` comments call out), noting the generated-reference exemption still waits on
deferred path discrimination; note in `product/specs/dogfood-docs-spec.md` that
per-page-type enforcement is partially unblocked.

## Rejected alternatives

- **Glob as the discriminator (`match: <glob>`).** The first draft of this spec.
  Rejected on the maintainer's point: a glob is a filesystem property, and the
  storage layer exists so a second StorageType (SQLite, granularity
  `UnitIsCollection`) can land — a backend with rows and no paths. A glob-based
  discriminator could not select those items at all. Metadata is the portable
  property; a `path:` glob is deferred to a later, storage-type-scoped condition
  rather than made the foundation.
- **Glob negation / exclude + multiple collections.** Add `exclude:` to a
  collection and model each page type as its own collection over the same tree.
  Rejected: it double-owns the tree, violating the storage spec's retained
  invariant #4, and each collection's `Unmatched` flags the other's files.
  Variants keep one collection, one membership pass, one owner per item.
- **Conditional (`oneOf`/`if-then`) JSON Schema as the whole answer.** Covers
  only frontmatter validation, leaving markdown/filesystem checks
  (`requires_h1`, kebab filename) uniform across page types. It composes fine
  *inside* a variant's `schema`, so variants and conditional schemas cooperate
  rather than compete.
