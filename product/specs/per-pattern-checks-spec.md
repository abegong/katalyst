# Per-pattern check scoping spec

> **Status: planning.** Implements issue #41. Lets one collection apply
> different checks to different files in its tree, selected by glob, without
> splitting the tree into overlapping collections. Builds on the storage layer
> (`product/specs/storage-layer-spec.md`, #31): collections now live inside a
> StorageInstance's `collections:` block, and this spec adds an optional
> `variants:` layer to a collection definition.

## Overview

A collection runs one check list against every item under its tree. There is no
way to require `weight` on content pages while exempting `_index.md`, or to
demand a kebab-case filename on content pages while letting `_index.md` keep its
underscore — those files share a directory and a collection has one check list.
This spec adds **variants**: an ordered list of glob-scoped check groups inside
a collection. An item runs the collection's base checks plus the checks of the
first variant whose `match` glob it satisfies. The item still belongs to exactly
one collection; only its check profile varies by page type.

## Value

Strict, page-type-aware enforcement is the difference between "the docs have *a*
title" and "content pages are correctly ordered and section indexes are
correctly structured." Today the dogfooding config
(`.katalyst/storage/local.yaml`) ships one permissive `page` schema for the
whole `docs/content/` tree precisely because it cannot say "content pages
require `weight`, `_index.md` files do not, and the generated reference pages are
exempt from `requires_h1`." Those comments in the config name #41 as the blocker.
Closing the gap lets the corpus be validated strictly against a green tree
instead of leniently.

## Current State

After the storage refactor (#31), a collection is declared inside a storage
instance and carries one check profile:

- `internal/config/config.go:171` — `Collection{Name, Path, Dir, Pattern,
  Schema, Checks, Query, Storage}`. `Pattern` selects membership; `Schema` and
  `Checks` are the single profile applied to every item.
- `internal/config/config.go:253` — `rawCollection{Path, Pattern, Schema,
  Checks, Query}` is the YAML shape inside an instance's `collections:` block.
- `cmd/engine.go:73` — `checksFor(c, meta)` builds an item's runnable check list
  from `c.Checks` (plus inline/flag object-schema overrides). The list is a
  function of the collection alone, not the item's path within it.
- `internal/storage/filesystem.go:64` — `Unmatched(c)` walks `c.Dir` and flags
  every file failing `c.Pattern`. doublestar has no negation, so a pattern
  cannot say "all `.md` except `_index.md`."

Two constraints make the gap concrete (both from #41):

1. `Unmatched` reports every file under the tree that misses the single
   `pattern`, so two collections cannot share or overlap a subtree without each
   flagging the other's files as unmatched.
2. doublestar globs have no negation, so "everything but `_index.md`" is
   inexpressible in one pattern.

The relevant invariant is `docs/content/deep-dives/domain-model.md:324`
(invariant #4): *"A collection owns its checks; an item belongs to one
collection. There is no glob-ordering 'first match wins' — an item's checks are
the checks of the collection whose directory contains it."* This spec
deliberately relaxes the **second clause** (no first-match-wins) while keeping
the first (one item, one collection); see Design.

The storage spec itself reinforces the constraint this spec must respect: it
retains *"a file belongs to exactly one collection"* (invariant #4) and keeps
"a file mapping into more than one collection" out of scope. So the #41 gap
cannot be closed by overlapping two collections over one tree — that would
double-own `_index.md`. It must be closed *within* a single collection.

## Design

### Variants: glob-scoped check groups inside a collection

A collection gains an optional, ordered `variants:` list. Each variant has a
`match` glob (relative to the collection directory, doublestar semantics — the
same engine `pattern` already uses) and its own optional `schema` and `checks`.

```yaml
collections:
  pages:
    path: docs/content
    pattern: "**/*.md"        # membership: which files are items
    schema: page              # base: the title contract, every page type
    checks:
      - kind: filesystem_extension_in
        values: [.md]
    variants:
      - match: "**/_index.md" # section landing pages
        schema: section_index
      - match: "reference/check-types/**/*.md"
        # generated reference pages: exempt from requires_h1, nothing added
      - match: "**/*.md"      # every other page is a content page
        schema: content_page
        checks:
          - kind: object_required_field
            field: weight
          - kind: filesystem_name_case
            style: kebab
          - kind: markdown_requires_h1
```

### Resolution semantics

For each item:

1. **Membership** is unchanged: a file is an item iff it matches the
   collection's `pattern`. Files under the tree that miss `pattern` are still
   `Unmatched` (invariant #6 holds). Variants never widen or narrow membership.
2. **First match wins among variants.** The item is routed to the *first*
   variant whose `match` glob it satisfies, in declaration order. At most one
   variant applies. An item matching no variant runs the base profile only.
3. **Base + variant compose additively.** The item's effective check list is the
   collection's base `schema`/`checks` (always) followed by the matched
   variant's `schema`/`checks`. Object schemas are additive: an item with both a
   base `page` schema and a variant `content_page` schema is validated against
   both, so the variant *adds* required fields rather than restating the title
   contract.

Additive base-plus-variant is what makes **exemption** work without negation: a
check belongs in a variant, not the base, exactly when some page type must *not*
run it. `_index.md` is exempt from `weight`/kebab/`requires_h1` simply because
those checks live in the `content_page` variant and `_index.md` is routed to the
`section_index` variant first. The generated reference pages are exempt from
`requires_h1` because their variant adds nothing. No `!`-negation, no second
overlapping collection.

### Object-schema precedence is unchanged

`cmd/engine.go:73` resolves object schemas highest-precedence-first: `--schema`
flag, inline `schema:` key, then the collection's configured object checks. The
matched variant's `schema` joins that lowest tier — it contributes object checks
to the "configured" set, still overridden wholesale by an inline key or the
flag. Markdown and filesystem checks (base and variant alike) always run; only
object-schema selection has precedence tiers, as today
(`docs/content/reference/configuration.md`, "Object-schema resolution
precedence").

### Where the routing lives

Variant matching is a property of *which checks an item runs*, not of *what an
item is*, so it lives in the check engine, not the storage seam:

- `internal/config`: add `Variants []CollectionVariant` to `Collection` and a
  `variants:` field to `rawCollection`. A `CollectionVariant` is `{Match string,
  Schema string, Checks []CheckInstance}`. Validate each `Match` is a non-empty,
  compilable doublestar pattern and each variant's `Checks`/`Schema` through the
  existing `normalizeCheck` path. A variant `schema` must name a known schema,
  same rule as a collection `schema`.
- `cmd/engine.go`: `checksFor(c, meta)` gains the item's collection-relative id
  (already known to its caller as `project.Item.ID`) so it can select the
  matched variant and append its checks. The base-checks loop is unchanged; the
  variant's checks feed the same `switch` that compiles `c.Checks`.
- `internal/storage` is **untouched**: `Items`, `Unmatched`, `Reference`, and
  granularity are all membership/locator concerns that variants do not affect.
  This keeps the seam from #31 closed.

Routing by the collection-relative id (the same string doublestar matched in
`Items`) means a variant `match` is written against the same relative paths as
`pattern` — `**/_index.md`, not an absolute path — so the two read consistently.

### Invariant #4 is revised, not broken

Invariant #4 (`domain-model.md:324`) gets reworded. The first clause holds
verbatim: an item belongs to exactly one collection (variants live *inside* one
collection; nothing overlaps). The second clause — "there is no glob-ordering
'first match wins'" — is the thing #41 asks to change, and it changes in a
contained way: first-match-wins applies only *within* a collection, among its
own declared variants, never across collections. The predictability the original
invariant bought (an item's collection is unambiguous from its directory) is
preserved; what becomes path-dependent is only *which subset of one collection's
checks* an item runs, and that is explicit and ordered in one file. The reworded
invariant and its rationale graduate into `domain-model.md` when this ships.

## Open Questions

1. **Keyword name.** `variants:` reads as "page-type variants of items in this
   collection" and avoids collision with "rule," which the docs already use for
   a check (`how-to/configure-rules.md`). Alternatives weighed: `rules:`
   (collides), `overrides:` (implies replacement, but the semantics are
   additive), `byPattern:`/`match:` (describe the key, not the concept).
   Recommendation: `variants`. Maintainer's call.
2. **No-variant-match behavior.** An item matching `pattern` but no variant runs
   the base profile only (recommended: lenient, composes with an optional
   trailing `match: "**/*"` catch-all when a project wants every item covered by
   a variant). The strict alternative — treat a no-variant item as an error so
   coverage is provably exhaustive — is heavier and can be added later behind a
   collection flag. Recommendation: lenient now.

## Documentation updates

**User docs (Hugo, `docs/content/`):**

- `reference/configuration.md`: document `variants:` under `collections` — the
  `match`/`schema`/`checks` keys, first-match-wins ordering, additive
  base-plus-variant composition, and that membership/`Unmatched` are governed by
  `pattern`, not variants. Extend the object-schema precedence section to note
  the variant schema joins the configured tier.
- `how-to/configure-rules.md`: add a short worked example — the `_index.md` vs
  content-page split — as the canonical motivating case.
- `deep-dives/domain-model.md`: reword invariant #4 (first clause kept,
  "first match wins" scoped to within-a-collection variants) and fold in the
  rationale.
- `deep-dives/storage.md`: a sentence noting variants route checks *within* a
  collection and do not affect the storage seam (membership, `Unmatched`,
  `Reference` are unchanged), so the seam stays closed.
- `reference/glossary.md`: add a **Variant** row (a glob-scoped check group
  within a collection).

**Developer docs:**

- `internal/config/README.md`: document the `variants:` field, its validation,
  and that it is a check-routing concern (not a storage/membership one).
- `cmd/engine.go` / `internal/config/config.go` doc comments: describe variant
  selection and additive composition.
- `AGENTS.md`: record that per-item check *routing* lives in the engine
  (`checksFor`), while membership and locators stay in
  `internal/storage` — variants must not leak into the seam.

**Specs (cross-references):** update `.katalyst/storage/local.yaml` to use
`variants` (retiring the `# issue #41` workaround comments) and note in
`product/specs/dogfood-docs-spec.md` that per-page-type enforcement is now
unblocked.

## Rejected alternatives

- **Glob negation / exclude patterns + multiple collections.** Add `exclude:`
  to a collection's `pattern` and express each page type as its own collection
  over the same tree (`content_pages` with `exclude: ["**/_index.md"]`,
  `section_indexes` with `pattern: "**/_index.md"`). Rejected: it double-owns the
  tree, violating the storage spec's retained invariant #4 ("a file belongs to
  exactly one collection"), and each collection's `Unmatched` pass would flag the
  other's files unless every collection also excludes every other's territory —
  the exact "two collections can't overlap a subtree" constraint #41 names.
  Variants keep one collection, one membership pass, one owner per file.
- **Conditional (`oneOf`/`if-then`) JSON Schema.** Author one schema that accepts
  either page shape. Rejected as the *sole* solution: it covers only object
  (frontmatter) validation, leaving markdown and filesystem checks
  (`requires_h1`, kebab filename) still uniform across page types, and it cannot
  exempt `_index.md` from a filename-casing check at all. It remains usable
  *inside* a variant's `schema` for the frontmatter portion — variants and
  conditional schemas compose rather than compete.
