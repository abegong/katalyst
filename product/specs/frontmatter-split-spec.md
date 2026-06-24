# Spec — frontmatter package realignment (fix + document)

> **Status: planning.** Next slice of package realignment (#39), building on the
> `project/{config,collection/query}` nesting shipped in #83. Evaluates the
> hunch that `internal/frontmatter` should dissolve into `collection/` plus a new
> `fix/`. Conclusion in brief: the **write path → `fix/`** is well-justified and
> recommended; the **read path → `collection/`** is the wrong altitude as stated,
> and the spec proposes a sharper home for it.

## Overview

`internal/frontmatter` is the last module whose name matches no core concept
(the gap called out alongside Collections in the realignment review). The hunch
is to move its contents into `collection/` and a new `fix/`. This spec takes the
hunch apart along the seam the code already has — a shared **read path** and a
single-consumer **write path** — and assesses each move on its own merits rather
than moving the package wholesale.

## Value

A reader who has learned the core concepts (project, collection, item, check,
inspector, operation) can today open `internal/` and map every directory to one
— except `frontmatter/`, which names a *file-format detail*, not a concept, and
bundles two responsibilities that belong in different places. Splitting it lets
the tree finish telling the concept story, and it untangles the `fix` operation,
whose logic is currently smeared across `cmd/fix.go` and `frontmatter.Format`.

## Current State

`internal/frontmatter` is a **leaf package**: it imports no other `internal/`
package, only the YAML/TOML/JSON libraries. It does two separable jobs.

**Read path — the markdown document model.** `Parse(src) → *Document` turns raw
bytes into the in-memory item form: `Meta` (`map[string]any`), `Body`, the
detected `Kind` (YAML/TOML/JSON), and a pointer→line map for violation
reporting. This is the glossary's **Document**: the markdown file-form of an
**Item**. Its consumers are broad and span layers:

| Consumer | Uses |
|---|---|
| `internal/checks` (`checks.go`, `checktest`) | `Document` as the unit a check runs against |
| `internal/inspect/collection.go` | `Parse` to materialize a configured collection's items |
| `internal/inspect/source.go` | `Parse` to profile **raw files before any collection exists** |
| `cmd/item.go` | `Parse` for `item get` (frontmatter/body) |
| `cmd/fix.go`, `cmd/write_validation.go` | `Parse` + `Document` |

`inspect/collection.go` even documents the parser as "a thin local adapter over
`frontmatter.Parse`; it deliberately does not [know about collection identity]."
The parser is, by design, **collection- and item-agnostic** — it operates on
bytes.

**Write path — the canonical formatter.** `Format(src) → []byte` re-serializes a
file into `fix`'s canonical form (top-level keys sorted, yaml.v3 block style,
single trailing newline, body verbatim). It has **exactly one caller**:
`cmd/fix.go:82`.

**The fix operation itself is split.** `cmd/fix.go` holds the orchestration —
`fixOne` (read → text-fix → format → atomic write), `applyTextFixes` (re-runs
the collection's `text_forbids` fixers and re-checks their work), `textFixers`,
and the temp-file-rename write — while the formatting primitive sits in
`frontmatter`. There is no `fix` package; the engine lives in `cmd/`, unlike its
sibling operation `check` (engine in `internal/checks`, thin `cmd/check.go`).

## Design

### The seam: read path vs write path

The two jobs have opposite shapes. The read path is a **shared primitive** with
six consumers across three layers; the write path is a **single-consumer
operation engine**. They should not move to the same place, and neither should
move *together*. Evaluate each.

### Write path → `internal/fix` (recommended)

Move `Format` and the orchestration now in `cmd/fix.go` into a new
`internal/fix` package; leave `cmd/fix.go` a thin cobra shell that calls it.

**Pros**
- **Single consumer.** `Format` is used only by `fix`; co-locating them couples
  things that are already coupled and decouples nothing that was shared.
- **Operation symmetry.** `check → internal/checks`, `inspect → internal/inspect`,
  and now `fix → internal/fix`: each operation owns a top-level engine package
  with a thin `cmd/` shell. `fix` is the odd one out today; this fixes it.
- **The formatter *is* the operation.** The canonical form is the definition of
  what `fix` does; it is fix-domain logic, not a frontmatter-parsing detail.
- **Consolidation.** `fixOne`/`applyTextFixes`/`textFixers` join the formatter in
  one place, with the atomic-write and CLI plumbing as the only residue in `cmd/`.

**Cons / notes**
- `fix` is an *operation*, not a data noun — but Operation is itself a core
  concept, and operation-named packages already exist (`collection/query`,
  and `checks`/`inspect` are operation+concept). It fits the existing grammar.
- `applyTextFixes` re-runs `plaintext` checks to verify a fix, so `internal/fix`
  imports `internal/checks` and `internal/checks/plaintext`. No cycle: `checks`
  does not import `fix`.
- **Placement: top-level, not nested.** `fix` is a project-wide operation like
  `check`; for symmetry it sits at `internal/fix`, beside `internal/checks`, not
  under `project/collection`.

### Read path → a concept home (the harder call)

The hunch sends this to `collection/`. Three reasons that is the wrong altitude:

1. **Wrong concept.** `Document`/`Parse` is the **Item**'s file form, not a
   Collection concern. The glossary defines Document under Item. A collection is
   a *group* of items; parsing one file is an item-level act.
2. **Layer inversion.** `inspect/source.go` parses **raw files before collection
   configuration exists** (the raw-source layer's whole point). If the parser
   lives in `collection/`, the pre-collection layer must import the collection
   package to read a file — directory semantics that contradict the layering the
   inspectors are built on.
3. **Couples a leaf.** The parser imports nothing internal today. Burying it in
   `project/collection/` drops a six-consumer primitive deep into the project
   subtree (which already imports `config` → `collection/query`), trading a clean
   leaf for a deep node for no functional gain.

So the read path wants a home that is (a) concept-aligned, (b) not below the
raw-source layer, (c) able to stay a leaf. Options, best first:

- **A. Rename the package to `document` (recommended).** Keep it a top-level
  leaf primitive (`internal/document`), but shed the format-detail name for the
  glossary concept its type already carries. Smallest change that closes the
  "name matches no concept" gap; preserves the leaf and the layering. The
  on-disk *frontmatter* block stays the glossary term for the metadata region;
  the *package* takes the document name because it owns Body and the line map
  too, not just frontmatter.
- **B. Home it at item altitude under collection** (`collection/item/` or
  `collection/document/`). Honors the `project ⊃ collection ⊃ item` containment
  the realignment is building, and nests rather than conflates. But it still
  places an item primitive below `collection`, so the raw-source inspector
  imports a deep collection path — the layer-inversion smell from reason 2,
  softened but not gone.
- **C. Into `collection/` directly (the hunch).** Rejected: conflates item-level
  parsing with collection-scoped logic and is the sharpest form of the layer
  inversion.

The recommendation is **A** unless we decide the containment story (B) outweighs
keeping the parser a layer-neutral leaf. That trade is the spec's main open
question.

### What does NOT move

- The `text_forbids` *fixers* live in `internal/checks/plaintext` (they are
  checks that happen to carry a `Fix` template); `internal/fix` consumes them. No
  fixer logic moves into `fix`.
- Format detection (`Kind`) travels with the read-path parser, wherever it lands.

## Open Questions

1. **Read-path home: `document` (A) vs `collection/item` (B)?** The deciding
   question. (A) optimizes for a layer-neutral leaf and minimal churn; (B)
   optimizes for the containment hierarchy at the cost of placing an item
   primitive under collection. Owner's call on which principle wins.
2. **Does `internal/fix` absorb the atomic write, or does that go to storage?**
   The temp-file-rename in `fixOne` is a storage concern (how the filesystem
   backend persists a write), not fix logic. Candidate to push into
   `internal/storage` as a "write item" operation later; out of scope to decide
   here, but flag it so `fix` doesn't ossify around file IO.
3. **Naming `internal/fix` vs folding into a future `operations/` group.** If
   more operations get extracted (a `write`/`get` engine), do they cluster? Not
   blocking; `internal/fix` top-level is the answer until a second case appears.

## Documentation updates

- **`docs/content/reference/glossary.md`** — no new term if option A keeps
  *Document*; confirm the Document entry still points at the renamed package.
  Note *fix* is the operation realized by `internal/fix`.
- **`docs/content/deep-dives/formatting.md`** ("Frontmatter and fix") — update
  the "parsing and formatting live in `internal/frontmatter`" line to the new
  split (`internal/document` + `internal/fix`); this is the architecture home for
  the why, per how-we-document.
- **`internal/frontmatter/AGENTS.md`** — splits into the new packages' AGENTS
  files: the read-path one points at `formatting.md`'s document section, the
  `fix` one at its fix section.
- **Root `AGENTS.md`** — update the layout tree: replace the `internal/frontmatter`
  line with `internal/document` (or `collection/item`) and add `internal/fix`.
- **`product/specs/domain-model-terminology-matrix.md`** — refresh the Document,
  Frontmatter, and (new) fix rows' Internal-code column once the move lands.
- **Generated reference** — run `make docs-gen-check`; this change touches no
  registry labels, so the generated pages must stay byte-identical.

## Rejected alternatives

- **Move `frontmatter/` wholesale into `collection/`.** Rejected: it merges a
  shared, layer-neutral read primitive with a single-consumer write engine and
  buries both below the raw-source layer that depends on the read path. The two
  responsibilities have different consumers, altitudes, and reasons to change.
- **Leave the formatter in the read-path package and only thin `cmd/fix.go`.**
  Rejected: keeps `fix`'s defining logic (the canonical form) in a package named
  for parsing, and leaves `fix` the only operation without an engine package.
- **Keep the `frontmatter` name.** Rejected: it is the one module name that maps
  to no concept — the entire reason this slice exists — and it undersells a
  package that owns the body and line map, not just the frontmatter block.
