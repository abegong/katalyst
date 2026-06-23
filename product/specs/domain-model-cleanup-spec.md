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

The two deep-dives re-define the same nouns (item, collection, schema, check,
inspector) independently, so the boundary between "general theory" and
"katalyst specifics" is blurred, that is the division-of-labor problem to fix.

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

The remaining contested calls, *Query* and *engine*, are carried as open
questions.

## Open Questions

1. **The "Query" contradiction.**
   **Context.** core-concepts lists *Query* as a supported operation;
   domain-model lists it under *out of scope*; `internal/query` and `item list
   --filter` exist in code today. The three sources disagree with each other and
   with reality, and the right framing needs a closer look at what actually
   shipped. (Owner is digging into this.)
   **Choices & tradeoffs.**
   - *Declare query partially shipped:* name what exists (single-collection
     filter/sort via `item list`) and scope the gap (cross-collection query, a
     dedicated `query` verb) as still out of scope. Honest about the code; needs
     core-concepts and domain-model both reworded.
   - *Keep "query" out of scope as a verb:* reframe `item list --filter` as a
     listing convenience, not "query," and leave the `query` operation aspirational.
     Simpler doc story; risks under-selling a real capability.
   **Recommendation.** Lean toward *partially shipped*, name the shipped subset
   and the gap so all three sources agree, but deferred to the owner's
   investigation.

2. **Define or drop "engine."**
   **Context.** *Engine* appears in CLI help ("the engine can run/enforce") and
   `cmd/engine.go`, but no doc defines it. It informally means the registry of
   check types and inspectors the CLI runs.
   **Choices & tradeoffs.** *Define it:* add a glossary entry and keep the help
   text, one more term to maintain but the help copy already leans on it.
   *Drop it:* rephrase help in terms of already-defined nouns (check types,
   inspectors), fewer terms but a small help-text sweep.
   **Recommendation.** Drop it from user-facing copy; it adds a term without a
   concept users configure or address. Reword help to "the check types/inspectors
   katalyst can run."

## Documentation updates

- **`docs/content/reference/glossary.md`** — the primary deliverable: one entry
  per concept, conflicts resolved. Add entries for *attribute* and *field* (with
  the specialization link), *operation*, *aggregate*, and *validation result*;
  fold *family* into its own row; apply the storage / source / item-document
  decisions.
- **`docs/content/deep-dives/core-concepts.md`** — reframed to tool-agnostic
  theory that links to the glossary for definitions; written in general terms
  only; drop "data interface" for storage terms; fix the *Query* framing (OQ 1).
- **`docs/content/deep-dives/domain-model.md`** — reframed to the katalyst
  instantiation linking up to core-concepts and across to the glossary; apply the
  item/document and source/raw-source decisions; refresh the stale Check list (it
  predates `text_*`, `writing_tells`, `sentence_case`); reconcile the
  out-of-scope *Query* note (OQ 1).
- **`docs/content/deep-dives/storage.md`** — confirm wording now that "data
  interface" is deprecated in favor of the storage vocabulary.
- **Generated check-types/inspectors reference** — regenerate with `make
  docs-gen` if any family or layer label changes (e.g. raw-source → source);
  never hand-edit.
- **`cmd/` help text** — apply the "engine" decision (OQ 2) and the
  source/raw-source rename so help strings match the glossary.

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
