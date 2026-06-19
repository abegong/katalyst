# Project layout & init

> **Status: planning.** Moves project config into a `.katalyst/` directory and
> redefines `init` as "prepare this directory," not "scaffold example content."
> Retires once the new layout ships and `decisions.md` D1 is updated.

## Overview

`katalyst init` stops generating example content. Today it writes an example
schema and an example document alongside the config; instead it should only
*prepare the current directory* as a katalyst project. In the same move, project
configuration leaves the repo root: a project is now marked by a `.katalyst/`
directory containing `config.yaml`, with schemas under `.katalyst/schema/`.

## Value

Dropping `schemas/book.json` and `notes/example.md` into a user's repo is
presumptuous — `init` litters the tree with files the user did not ask for and
must delete before doing real work. Preparing the directory (creating only
`.katalyst/`) is the unobtrusive thing, and it matches how `git init`,
`npm init`, and `terraform init` behave: they set up tooling state, not sample
content.

Grouping katalyst's project state under one hideable `.katalyst/` directory —
the way `.git`, `.github`, and `.vscode` do — keeps the user's working tree
clean instead of scattering a top-level `katalyst.yaml` and a sibling `schemas/`
folder among their actual documents.

## Current State

- **`init` ships content, not just setup.** `cmd/init.go` writes three files:
  `katalyst.yaml`, `schemas/book.json`, and `notes/example.md`. Only the first
  is project setup; the other two are an example schema and an example document.
  The scaffold strings (`scaffoldSchema`, `scaffoldExample`) and the tests that
  assert they exist and `check` cleanly (`cmd/init_test.go`:
  `TestInit_scaffoldChecksCleanly`, `TestInit_scaffoldIsCanonical`) all encode
  "init produces a working example."
- **Config lives at the root.** `internal/config/config.go` sets
  `Filename = "katalyst.yaml"`; `find()` ascends from the working directory
  looking for the nearest ancestor containing that file, and that directory
  becomes `Root`. This is locked as `decisions.md` **D1**.
- **Schemas are a name→path map of JSON files.** Config carries a `schemas:`
  map (`book: ./schemas/book.json`); `config.Load` resolves each to an absolute
  path, and `cmd/schema.go` reads `cfg.Schemas` for `schema list`/`schema show`.
  The validator (`internal/validator`) consumes JSON Schema.

## Design

### Project layout

A project is a directory that contains a `.katalyst/` subdirectory:

```
<project root>/
  .katalyst/
    config.yaml        # the project config (formerly root katalyst.yaml)
    schema/            # one file per schema
      foo.yaml
  ...                  # the user's own documents, untouched
```

The **project root** is the directory *containing* `.katalyst/`, not `.katalyst/`
itself. Paths in `config.yaml` resolve against the root, so a collection's
`path: notes` still means `<root>/notes`.

This supersedes **D1**: the project marker becomes the `.katalyst/` directory
(more precisely, `.katalyst/config.yaml`) rather than a root-level
`katalyst.yaml`. The nearest-ancestor discovery rule is unchanged in spirit —
walk up until the marker is found — only the marker differs. D1 is rewritten,
not contradicted silently.

### Discovery

`config.find()` ascends looking for an ancestor whose `.katalyst/config.yaml`
exists, and returns that ancestor as `Root`. `config.Filename` is replaced by a
`Dir = ".katalyst"` constant plus the `config.yaml` leaf; symlink resolution of
the root is preserved (it matters for macOS temp dirs). `ErrNotFound` and its
message update to name `.katalyst/`.

### Schemas

Schemas live at `.katalyst/schema/{name}.yaml`. The file *is* the schema named
by its stem: `.katalyst/schema/book.yaml` defines schema `book`. Discovery is
**by convention** — the loader scans `.katalyst/schema/*.yaml` and populates
`Config.Schemas` (name → absolute path), so a `schema: book` reference in a
collection resolves to that file. This drops the explicit top-level `schemas:`
map; a name no longer has to be declared twice (once in `schemas:`, once where
it's used). `cmd/schema.go` keeps reading `cfg.Schemas` and is otherwise
unaffected.

Schemas are authored in **YAML**, not JSON. JSON Schema is just a data shape, so
a YAML document parses to the same structure; the validator already works in
terms of decoded `any` values. The loader unmarshals `*.yaml`, and the validator
is fed the resulting structure instead of raw JSON bytes. (See Open Questions Q2
for whether `.json` is still accepted in `schema/`.)

### `init` semantics

`katalyst init [--dir <path>]` prepares the target directory:

1. Creates `.katalyst/` and `.katalyst/schema/`.
2. Writes a minimal `.katalyst/config.yaml` — **no collections defined**, just a
   commented template pointing at the docs. No example schema, no example
   document.
3. Refuses to overwrite: if `.katalyst/` already exists, it errors (exit 2) and
   writes nothing, preserving today's all-or-nothing guarantee.
4. Prints one line per path created.

A freshly-`init`ed project loads cleanly and `check` is a no-op success (zero
collections → nothing to check, exit 0). This replaces the old invariant that
the scaffold ships a passing example; the new invariant is that the scaffold
is *valid and empty*.

### Domain-model impact

The **Project** concept (`product/domain-model.md`, `cli-spec.md` "Concepts")
is redefined: "the directory containing `.katalyst/config.yaml`" rather than
"the directory containing `katalyst.yaml`." The **schema** vocabulary gains the
convention that a schema's name is its `.katalyst/schema/` filename stem.

### Code touch points

- `cmd/init.go` — drop `scaffoldSchema`/`scaffoldExample`; write `.katalyst/`
  scaffold; update the `Short` help text.
- `cmd/init_test.go` — replace the three-file assertions; drop
  `TestInit_scaffoldChecksCleanly`; keep refuse-to-overwrite and
  fix-canonical (the empty config must still be in `fix` canonical form).
- `internal/config/config.go` — `Dir` constant + discovery change; scan
  `.katalyst/schema/*.yaml` to build `Schemas`; remove the `schemas:` YAML key
  from `rawConfig`; YAML schema parsing.
- `internal/validator` — accept a decoded schema structure (or YAML bytes)
  rather than only JSON.
- `cmd/testdata/` and `internal/validator/testdata/` — schemas move to the new
  location/format; `cli-spec.md` config example updates.
- `product/decisions.md` D1, `product/domain-model.md`, `docs/configuration.md`.

## Open Questions

1. **Schema discovery — convention vs. explicit map.** Design proposes
   convention: scan `.katalyst/schema/*.yaml`, name = stem, no `schemas:` key.
   Alternative: keep an explicit `schemas:` map (now pointing into
   `.katalyst/schema/`) for indirection/aliasing. Recommend convention; it
   removes a redundant declaration and matches the user's "schemas are files in
   `.katalyst/schema/`" framing.
2. **Schema format — YAML only or YAML+JSON.** Recommend YAML as the authored
   format. Open: do we still accept `.json` files in `schema/` (scan both
   extensions) for users who already have JSON Schema, or YAML-only and convert?
3. **What does `init`'s `config.yaml` contain?** Recommend a commented template
   with zero active collections (loads clean, `check` is a no-op). Alternative:
   a single empty `collections:` map, or a fully-commented file with a worked
   example in comments.
4. **Backward compatibility.** Recommend a hard switch — katalyst is pre-v0 with
   no external users, so no transitional support for a root `katalyst.yaml`.
   Confirm there's no installed base to migrate.
5. **Does `init` create an empty `.katalyst/schema/`?** Git won't track an empty
   directory, so it vanishes on commit. Options: create it anyway (local
   convenience), add a `.gitkeep`, or don't create it until the first schema
   exists. Recommend creating it without a `.gitkeep`.

## Rejected alternatives

- **Keep examples behind a flag (`init --example`).** Adds surface area to
  preserve behavior we decided is wrong by default; a docs example or a separate
  `examples/` repo serves the "show me a working project" need better.
- **Keep `katalyst.yaml` at the root, add only `schema/`.** Leaves project state
  split between a root file and a folder; the point of `.katalyst/` is one
  hideable home for everything katalyst owns.
- **`.katalyst.yaml` single dotfile instead of a directory.** Cleaner for the
  config alone, but schemas (and future cache/state) have nowhere to live; a
  directory scales, a single file doesn't.

## Test checklist (what the pending tests assert)

`init`:
- [ ] creates `.katalyst/config.yaml` and `.katalyst/schema/`
- [ ] writes no example schema and no example document
- [ ] refuses to run when `.katalyst/` already exists; writes nothing
- [ ] scaffolded `config.yaml` is in `fix` canonical form
- [ ] `check` on a freshly-`init`ed project exits 0 with no collections

Config & discovery:
- [ ] project root is the ancestor containing `.katalyst/config.yaml`
- [ ] `.katalyst/schema/{name}.yaml` is discovered as schema `{name}`
- [ ] a collection's `schema: foo` resolves to `.katalyst/schema/foo.yaml`
- [ ] YAML-authored schema validates the same documents the old JSON one did
- [ ] no `.katalyst/` in any ancestor → `ErrNotFound`, exit 2
