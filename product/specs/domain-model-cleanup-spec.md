# Spec — domain model cleanup

> **Status: planning.** Reconcile katalyst's terminology into one authoritative
> glossary and sharpen the division of labor between the core-concepts and
> domain-model docs. Covers issue #26. Issue #39 (package realignment) is
> explicitly out of scope here, see Current State.

## Overview

Terminology has accreted across the docs and code: the same concept wears
different names in different places, and a few names exist in only one place.
This spec makes the glossary the single source of truth, governs the
general-vs-specific term split with one rule, and reconciles the two conceptual
deep-dives against the glossary with a clear boundary. The reconciliation
evidence is the [terminology matrix](./domain-model-terminology-matrix.md): one
row per concept across code, CLI, and the three docs.

## Value

A reader, contributor, or agent should learn a term once and find it used the
same way in the CLI, the code, and every doc. Today they must translate between
*item* and *document*, *data interface* and *storage instance*, *source* and
*raw-source*, and reconcile a glossary that omits terms the core-concepts doc
leans on. Settling the vocabulary once removes that tax and keeps the divergence
from reappearing.

## Current State

Three docs define overlapping vocabulary at different altitudes, and neither the
code nor the CLI fully agrees with them:

- `docs/content/deep-dives/core-concepts.md` is the tool-agnostic model
  (data interface, item, collection, attribute, operation, check, inspector).
- `docs/content/deep-dives/domain-model.md` is the katalyst-specific model
  (markdown document, schema, config, resolver, the check families, invariants).
- `docs/content/reference/glossary.md` is meant to be the quick-lookup source of
  truth, but it omits some core-concepts terms (*attribute*, *operation*,
  *aggregate*) and relegates others (*family*) to sub-clauses.

PR #73 reshaped this landscape: it slimmed `domain-model.md` into a
katalyst-specific hub that summarizes each entity and links out, and moved the
detail into new `collections.md` and `inspectors.md` deep-dives. That fixes most
of the old "monolith re-defines everything" problem. What remains is
`core-concepts.md`, still encyclopedic, re-defining the same nouns the glossary
should own and the hub now indexes, so the general-vs-specific boundary is still
the division-of-labor problem to fix, just concentrated in core-concepts now.

The [terminology matrix](./domain-model-terminology-matrix.md) catalogues the
full set of divergences. The sharpest:

- **`item` vs `document`** for the same filesystem unit (core-concepts/CLI say
  *item*; domain-model/glossary/`frontmatter.Document` say *document*).
- **"data interface" vs `StorageType`/`StorageInstance`** for the backend.
- **"engine"** is user-facing in CLI help yet defined in no doc.
- **"Query"** is a supported operation in core-concepts, *out of scope* in
  domain-model, and shipped as `internal/query` + `item list --filter` in code.

**Out of scope: package realignment (#39).** #39 realigns the `internal/` layout
to the concepts. It is a rename-heavy refactor entangled with #31 (connectors,
not yet built) and #35 (per-directory `AGENTS.md`), and it names this glossary
work as its prerequisite. It is deferred entirely from this spec, no execution
and no planning here, and picks up once the vocabulary settles. Prior work this
spec builds on:

- [`check-terminology-spec.md`](./check-terminology-spec.md) already established
  *check type* vs *check instance* and moved the generated reference from "rules"
  to "check types"; that vocabulary is settled and this spec extends it.
- [`storage-layer-spec.md`](./storage-layer-spec.md) owns the `StorageType` /
  `StorageInstance` / `CollectionDefinition` / `Granularity` vocabulary; the
  "data interface" resolution below stays consistent with it.

## Design

### General and specific terms

One rule governs every general/specific pair (item/document, attribute/field):

> Default to the general term. Use the specific term only where the concrete
> form is the subject, where the sentence is true *because* the unit is a
> markdown file or its characteristic is an object key, and would not hold for
> another backend.

Test: if a Postgres row could replace the markdown file and the sentence still
reads true, the general term is right. Specific terms are **specializations, not
synonyms**: a document is the markdown form of an item; a field is the
object-key form of an attribute. "A document is an item" and "a field is an
attribute" always hold; the reverse does not.

**Item / document.** *Item* is the default and the CLI noun: the unit in a
collection, addressed by selector, that `check`/`fix`/`get`/`list` operate on.
*Document* names the markdown file form (frontmatter + body + line map) and is
used only where that form is the subject:

- parsing and serialization: the `fix` round-trip, fence/format detection, line
  maps;
- the body-vs-frontmatter structure itself;
- the `document_shape` inspector, which profiles raw files before item identity
  exists;
- backend illustrations ("a document in MongoDB").

`frontmatter.Document` keeps its name; it is the parser for the file form. This
matches where *document* already clusters in the code (`internal/frontmatter`,
`cmd/fix`, the source-layer inspectors).

**Attribute / field.** *Attribute* is the general umbrella: any named
characteristic of an item, a frontmatter key, but also its filename, path, or
extension. It lives in core-concepts. *Field* is the structured-object
specialization: a key in the item's object/frontmatter map. Every field is an
attribute; a filename is an attribute but not a field. Use *field* in
object/frontmatter contexts. This ratifies the de-facto state: *attribute*
appears only in `core-concepts.md` today, while *field* is used everywhere
concrete and in 200+ Go identifiers, and filesystem checks already mean
"frontmatter key" by *field* (`name_matches_field`).

### Division of labor between the conceptual docs

The glossary is canonical; the deep-dives narrate.

- **Glossary** (`reference/glossary.md`) is the single source of truth: every
  term defined exactly once, here, including the specialization links (document =
  markdown item; field = object attribute). Both deep-dives link to it for
  definitions instead of restating them.
- **Core concepts** (`deep-dives/core-concepts.md`) is the general,
  backend-neutral model, written entirely in general terms (item, attribute,
  collection, storage, operation, check, inspector). It explains *why* the
  abstractions exist and what they would mean for Postgres or Mongo. It narrates
  and links to the glossary; it does not define terms.
- **Domain model** (`deep-dives/domain-model.md`) is katalyst's concrete
  instantiation: the filesystem backend, markdown documents, object fields, JSON
  Schema, the Go types, the `check`/`fix` lifecycles, and the invariants. It uses
  general terms by default and the specific terms where the concrete form is the
  subject.

The difference between the two deep-dives *is* the general/specific seam:
core-concepts is the model katalyst could implement on any backend; domain-model
is the model it does implement on markdown files. The terminology rule above is
what keeps them sharing one vocabulary.

### Settled naming

- **Storage, not data interface.** *Data interface* is deprecated. The backend
  vocabulary is *StorageType* / *StorageInstance* (per `storage-layer-spec.md`),
  used even at the general level; core-concepts drops "data interface."
- **Source, not raw-source** for the inspector layer, matching code and CLI
  (`inspect.SourceView`, the `source/` reference directory). Rename the docs.
- **Item primary, document specialized**; **attribute general, field
  specialized**, per the rule above.

Two contested calls, *Query* and *engine*, are **deferred to their own
branches**: each is large enough to need separate treatment and would bloat this
cleanup.

- **Query** is genuinely contradictory across the sources (a supported operation
  in core-concepts, *out of scope* in domain-model, shipped as `internal/query` +
  `item list --filter` in code). Resolving it means deciding what query actually
  is in katalyst today, which is its own investigation.
- **Engine** is user-facing in CLI help yet defined in no doc. Whether to define
  it or scrub it from help text is a small but separate sweep.

Both are listed here only so the boundary is explicit; neither blocks the
glossary or the doc reconciliation.

## Open Questions

_None._ Query and engine are deferred to their own branches (see Settled
naming). The doc-consolidation direction settled after rebasing on #73, which
already turned `domain-model.md` into a katalyst-specific hub and rehomed its
detail into the new `collections.md` and `inspectors.md`: keep `domain-model.md`
as that hub, slim `core-concepts.md` into the parallel general-altitude hub, and
terminology-align the new pages.

## Documentation updates

- **`docs/content/reference/glossary.md`** — the primary deliverable: one entry
  per concept, conflicts resolved. Add entries for *attribute* and *field* (with
  the specialization link), *operation*, *aggregate*, and *validation result*;
  fold *family* into its own row; apply the storage / source / item-document
  decisions.
- **`docs/content/deep-dives/core-concepts.md`** — the primary doc target now:
  slim from the encyclopedic definitions into a general-altitude hub mirroring
  `domain-model.md`. Define each general term in a line, link to the glossary for
  the definition and to where the general idea is discussed (e.g. *operation* →
  progressive-operations); keep only its own thesis (the structured/unstructured
  bridge). Written in general terms only; drop "data interface" for storage terms.
- **`docs/content/deep-dives/domain-model.md`** — **kept** as the katalyst hub
  #73 built. No structural change; terminology-align it (item/document,
  source/raw-source) and sharpen its one-line statement of how it differs from
  core-concepts.
- **`docs/content/deep-dives/collections.md`, `inspectors.md`** — new in #73 and
  the homes for the former domain-model detail (resolver table, `check`
  lifecycle, invariants, inspector layers). Apply the source/raw-source and
  item/document decisions here.
- **`docs/content/deep-dives/storage.md`** — confirm wording now that "data
  interface" is deprecated in favor of the storage vocabulary.
- **Generated check-types/inspectors reference** — regenerate with `make
  docs-gen` if any family or layer label changes (e.g. raw-source → source);
  never hand-edit.
- **`cmd/` help text** — apply the source/raw-source rename so help strings match
  the glossary.

## Rejected alternatives

- **One mega-doc merging core-concepts, domain-model, and glossary.** Rejected:
  the three serve different jobs (theory, instantiation, lookup); collapsing them
  trades the division-of-labor clarity this spec creates for one unreadable page.
- **Keep both "data interface" and "storage."** Rejected: two names for the
  backend force readers to learn they are the same thing. The shipped code and
  `storage-layer-spec.md` already use storage; data interface loses.
- **A generated terminology matrix.** Rejected for now: the sources (Go
  identifiers, CLI help, three prose docs) share no machine-readable shape, so the
  [matrix](./domain-model-terminology-matrix.md) is maintained by hand and
  refreshed when things move.
