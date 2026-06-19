# Project layout & init

> **Status: planning.** Moves project config into a `.katalyst/` directory,
> defines schemas and collections as one named YAML file each, and redefines
> `init` as "prepare this directory," not "scaffold example content." Retires
> once the new layout ships and `decisions.md` D1 is updated.

## Overview

`katalyst init` stops generating example content. Today it writes an example
schema and an example document alongside the config; instead it should only
*prepare the current directory* as a katalyst project. In the same move, project
configuration leaves the repo root for a `.katalyst/` directory, and both
**schemas and collections** are defined the same way — one named YAML file each,
under `.katalyst/schema/` and `.katalyst/collections/`. The file's stem is the
schema or collection name.

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
    config.yaml        # project-level settings (optional; empty in v0)
    schema/            # one named YAML file per schema
      book.yaml
    collections/       # one named YAML file per collection
      notes.yaml
  ...                  # the user's own documents, untouched
```

The **project root** is the directory *containing* `.katalyst/`, not `.katalyst/`
itself. Paths inside a collection definition resolve against the root, so a
collection's `path: notes` still means `<root>/notes`.

This supersedes **D1**: the project marker becomes the `.katalyst/` directory
rather than a root-level `katalyst.yaml`. The nearest-ancestor discovery rule is
unchanged in spirit — walk up until the marker is found — only the marker
differs. D1 is rewritten, not contradicted silently.

### Discovery

`config.find()` ascends looking for an ancestor that contains a `.katalyst/`
directory, and returns that ancestor as `Root`. `config.Filename` is replaced by
a `Dir = ".katalyst"` constant; symlink resolution of the root is preserved (it
matters for macOS temp dirs). `ErrNotFound` and its message update to name
`.katalyst/`.

### Schemas and collections share one convention

Both schemas and collections are defined **one named YAML file each**, and the
file's stem is the name. The loader scans the two directories and builds the
maps it builds today — no central registry, no name declared twice.

**Schemas** live at `.katalyst/schema/{name}.yaml`. `.katalyst/schema/book.yaml`
defines schema `book`. The loader scans `.katalyst/schema/*.yaml` and populates
`Config.Schemas` (name → absolute path), so a `schema: book` reference resolves
to that file. `cmd/schema.go` keeps reading `cfg.Schemas` unchanged.

Schemas are authored in **YAML**, not JSON. JSON Schema is just a data shape, so
a YAML document parses to the same structure; the validator already works in
terms of decoded `any` values. The loader unmarshals `*.yaml` and feeds the
validator the resulting structure instead of raw JSON bytes. (See Q2 for whether
`.json` is still accepted in `schema/`.)

**Collections** live at `.katalyst/collections/{name}.yaml`. The file holds what
a `collections:` map entry holds today — `path`, `pattern`, `schema`, `checks` —
minus the name, which is the stem:

```yaml
# .katalyst/collections/notes.yaml
path: notes              # optional; defaults to the collection name
pattern: "*.md"          # optional; default "*.md"
schema: book             # a schema name from .katalyst/schema/; OR use checks:
checks:
  - kind: markdown_title_matches_h1
  - kind: filesystem_filename_matches_slug
```

The loader scans `.katalyst/collections/*.yaml` and builds `Config.Collections`
(sorted by name, as today). This replaces the top-level `collections:` map from
`cli-spec.md`. All existing per-collection semantics carry over verbatim: a
collection needs a `schema` or non-empty `checks` (else load error); `path`
defaults to the name; `pattern` defaults to `*.md`; the first object check's
schema mirrors into `Collection.Schema` for display.

### `config.yaml`

With schemas and collections both file-per-definition, `config.yaml` holds only
**project-level settings**, of which v0 has none. It is therefore **optional**:
the project marker is the `.katalyst/` directory, and a project with no
`config.yaml` loads fine. `init` still writes a commented placeholder so the
file exists as the obvious home for future settings (see Q3).

### `init` semantics

`katalyst init [--dir <path>]` prepares the target directory:

1. Creates `.katalyst/`, `.katalyst/schema/`, and `.katalyst/collections/`.
2. Writes a commented placeholder `.katalyst/config.yaml`. **No example schema,
   no example collection, no example document.**
3. Refuses to overwrite: if `.katalyst/` already exists, it errors (exit 2) and
   writes nothing, preserving today's all-or-nothing guarantee.
4. Prints one line per path created.

A freshly-`init`ed project loads cleanly and `check` is a no-op success (zero
collections → nothing to check, exit 0). This replaces the old invariant that
the scaffold ships a passing example; the new invariant is that the scaffold
is *valid and empty*.

### Domain-model impact

The **Project** concept (`product/domain-model.md`, `cli-spec.md` "Concepts")
is redefined: "the directory containing `.katalyst/`" rather than "the directory
containing `katalyst.yaml`." The **schema** and **collection** vocabulary gains
the shared convention that the name is the filename stem under
`.katalyst/schema/` and `.katalyst/collections/` respectively.

### Code touch points

- `cmd/init.go` — drop `scaffoldSchema`/`scaffoldExample`/`scaffoldConfig`'s
  collection body; write the `.katalyst/` scaffold; update the `Short` help text.
- `cmd/init_test.go` — replace the three-file assertions; drop
  `TestInit_scaffoldChecksCleanly`; keep refuse-to-overwrite and
  fix-canonical (the placeholder config must still be in `fix` canonical form).
- `internal/config/config.go` — `Dir` constant + directory-marker discovery;
  scan `.katalyst/schema/*.yaml` for `Schemas` and `.katalyst/collections/*.yaml`
  for `Collections`; drop `rawConfig`'s `schemas:`/`collections:` maps (a
  per-collection `rawCollection` file replaces the map value); YAML schema
  parsing. Per-collection validation (`no checks configured`, unknown schema)
  moves to the per-file loop but is otherwise unchanged.
- `internal/validator` — accept a decoded schema structure (or YAML bytes)
  rather than only JSON.
- `cmd/testdata/` and `internal/validator/testdata/` — schemas and collection
  configs move to the new location/format; `cli-spec.md` "Config (v0)" updates.
- `product/decisions.md` D1, `product/domain-model.md`, `docs/configuration.md`.

## Open Questions

1. **Schema/collection format — YAML only or YAML+JSON.** Recommend YAML as the
   authored format for both. Open: do we still accept `.json` schema files in
   `schema/` (scan both extensions) for users who already have JSON Schema, or
   YAML-only and convert? (Collections were never JSON, so this is schema-only.)
2. **Folder naming — `schema/` (singular) vs `collections/` (plural).** As
   stated by the user the two differ. Recommend making them consistent — either
   both singular (`schema/`, `collection/`) or both plural
   (`schemas/`, `collections/`). Mild preference for plural, matching the
   directories holding many files.
3. **What does `init`'s `config.yaml` contain, and is it written at all?** Since
   v0 has no project-level settings and the marker is the directory, `config.yaml`
   is optional. Recommend writing a commented placeholder so the settings home is
   obvious. Alternative: don't write it until there's a setting to put in it.
4. **Backward compatibility.** Recommend a hard switch — katalyst is pre-v0 with
   no external users, so no transitional support for a root `katalyst.yaml`.
   Confirm there's no installed base to migrate.
5. **Does `init` create empty `schema/` and `collections/` dirs?** Git won't
   track an empty directory, so they vanish on commit. Options: create them
   anyway (local convenience), add `.gitkeep`s, or don't create them until the
   first definition exists. Recommend creating them without `.gitkeep`s.

## Rejected alternatives

- **Keep examples behind a flag (`init --example`).** Adds surface area to
  preserve behavior we decided is wrong by default; a docs example or a separate
  `examples/` repo serves the "show me a working project" need better.
- **Keep `katalyst.yaml` at the root, add only `schema/`.** Leaves project state
  split between a root file and a folder; the point of `.katalyst/` is one
  hideable home for everything katalyst owns.
- **`.katalyst.yaml` single dotfile instead of a directory.** Cleaner for the
  config alone, but schemas, collections (and future cache/state) have nowhere
  to live; a directory scales, a single file doesn't.
- **Keep collections in a `collections:` map inside `config.yaml`.** Asymmetric
  with schemas once schemas become files, and a long config map is harder to
  diff and reorganize than one file per collection. File-per-collection also
  lets a single collection move or be removed without touching a shared file.

## Test checklist (what the pending tests assert)

`init`:
- [ ] creates `.katalyst/`, `.katalyst/schema/`, `.katalyst/collections/`
- [ ] writes no example schema, collection, or document
- [ ] refuses to run when `.katalyst/` already exists; writes nothing
- [ ] scaffolded `config.yaml` is in `fix` canonical form
- [ ] `check` on a freshly-`init`ed project exits 0 with no collections

Config & discovery:
- [ ] project root is the ancestor containing `.katalyst/`
- [ ] `.katalyst/schema/{name}.yaml` is discovered as schema `{name}`
- [ ] `.katalyst/collections/{name}.yaml` is discovered as collection `{name}`
- [ ] a collection's `schema: foo` resolves to `.katalyst/schema/foo.yaml`
- [ ] a collection file with neither `schema` nor `checks` → load error
- [ ] `path` defaults to the collection name; `pattern` defaults to `*.md`
- [ ] YAML-authored schema validates the same documents the old JSON one did
- [ ] a project with no `config.yaml` (but a `.katalyst/` dir) still loads
- [ ] no `.katalyst/` in any ancestor → `ErrNotFound`, exit 2
