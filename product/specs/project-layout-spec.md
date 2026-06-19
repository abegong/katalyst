# Project layout & init

> **Status: done.** Moves project config into a `.katalyst/` directory,
> defines schemas and collections as one named file each (discovered by
> convention, configurable in `config.yaml`), and redefines `init` as "prepare
> this directory," not "scaffold example content." Shipped; `decisions.md` D1,
> the domain model, `cli-spec.md`, and the user docs are reconciled. Retained
> for reference pending graduation cleanup.

## Overview

`katalyst init` stops generating example content. Today it writes an example
schema and an example document alongside the config; instead it should only
*prepare the current directory* as a katalyst project. In the same move, project
configuration leaves the repo root for a `.katalyst/` directory, and both
**schemas and collections** are defined the same way — one named file each, under
`.katalyst/schemas/` and `.katalyst/collections/`, where the file's stem is the
name. This convention is the default; how each is discovered (by directory scan
or an explicit map) and what format each file is in (YAML or JSON) are settable
per kind in `.katalyst/config.yaml`.

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
    config.yaml        # project-level settings (optional; all keys default)
    schemas/           # one named file per schema
      book.yaml
    collections/       # one named file per collection
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

Schemas and collections are defined the same way: **one named file each**, where
the file's stem is the name. The loader scans the two directories and builds the
same maps it builds today (`Config.Schemas`, `Config.Collections`) — no central
registry, no name declared twice.

**Schemas** live at `.katalyst/schemas/{name}.yaml`. `.katalyst/schemas/book.yaml`
defines schema `book`. The loader scans the directory and populates
`Config.Schemas` (name → absolute path), so a `schema: book` reference resolves
to that file. `cmd/schema.go` keeps reading `cfg.Schemas` unchanged. A YAML
schema is just JSON Schema in YAML syntax — it parses to the same structure, and
the validator already works in terms of decoded `any` values, so the loader
unmarshals the file and feeds the validator the result rather than raw JSON
bytes.

**Collections** live at `.katalyst/collections/{name}.yaml`. The file holds what
a `collections:` map entry holds today — `path`, `pattern`, `schema`, `checks` —
minus the name, which is the stem:

```yaml
# .katalyst/collections/notes.yaml
path: notes              # optional; defaults to the collection name
pattern: "*.md"          # optional; default "*.md"
schema: book             # a schema name from .katalyst/schemas/; OR use checks:
checks:
  - kind: markdown_title_matches_h1
  - kind: filesystem_filename_matches_slug
```

The loader scans `.katalyst/collections/*` and builds `Config.Collections`
(sorted by name, as today). This replaces the top-level `collections:` map from
`cli-spec.md`. All existing per-collection semantics carry over verbatim: a
collection needs a `schema` or non-empty `checks` (else load error); `path`
defaults to the name; `pattern` defaults to `*.md`; the first object check's
schema mirrors into `Collection.Schema` for display.

### `config.yaml` — discovery and format options

`.katalyst/config.yaml` holds **project-level settings**. In v0 those settings
configure, per kind, *how* schemas and collections are found and *what format*
their files are in. Every key has a default, so the file is **optional**: a
project with no `config.yaml` behaves exactly as the convention above describes.

```yaml
# .katalyst/config.yaml — all keys optional; values shown are the defaults.
schemas:
  discovery: convention   # convention: scan .katalyst/schemas/; explicit: use `defs`
  format: yaml            # yaml | json | both — extensions scanned and how files parse
  # defs:                 # consulted only when discovery: explicit (name → path)
  #   book: ./.katalyst/schemas/book.yaml
collections:
  discovery: convention
  format: yaml
  # defs:                 # consulted only when discovery: explicit (name → definition)
  #   notes: { path: notes, schema: book }
```

- **`discovery: convention`** (default) scans the kind's directory; the name is
  the filename stem. **`discovery: explicit`** ignores the directory and reads
  the `defs` map — the pre-spec behavior, preserved for users who want a single
  declared list. `defs` is required (and non-empty) under `explicit`.
- **`format`** selects which extensions count and how files parse: `yaml` →
  `*.yaml`/`*.yml`, `json` → `*.json`, `both` → either, dispatched by extension.
  It governs convention scans and the file contents that `defs` paths point at.
- The options are **independent per kind** — schemas can be `explicit`+`json`
  while collections stay `convention`+`yaml`.

The two `defs` maps under `explicit` are the old top-level `schemas:`/
`collections:` maps relocated under their kind's settings block (so the block
name `schemas`/`collections` doesn't collide with a bare map). This keeps both
the convention and explicit code paths alive, selectable without a rebuild.

### `init` semantics

`katalyst init [--dir <path>]` prepares the target directory:

1. Creates `.katalyst/`, `.katalyst/schemas/`, and `.katalyst/collections/`
   (empty; no `.gitkeep`).
2. Writes `.katalyst/config.yaml` as a **commented template** — the default
   `schemas`/`collections` settings block shown commented out, so the keys are
   discoverable but the file is effectively empty. **No example schema, no
   example collection, no example document.**
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
`.katalyst/schemas/` and `.katalyst/collections/` respectively.

### Code touch points

- `cmd/init.go` — drop `scaffoldSchema`/`scaffoldExample` and the collection
  body of `scaffoldConfig`; write the `.katalyst/` scaffold (dirs + commented
  `config.yaml`); update the `Short` help text.
- `cmd/init_test.go` — replace the three-file assertions; drop
  `TestInit_scaffoldChecksCleanly`; keep refuse-to-overwrite and
  fix-canonical (the placeholder config must still be in `fix` canonical form).
- `internal/config/config.go` — `Dir` constant + directory-marker discovery.
  `rawConfig` becomes two per-kind settings blocks (`discovery`, `format`,
  `defs`) instead of bare `schemas:`/`collections:` maps. `Load` branches on
  `discovery`: convention scans the kind's directory (filtered by `format`),
  explicit reads `defs`. The collection-definition shape (`rawCollection`) and
  its per-collection validation (`no checks configured`, unknown schema) are
  unchanged — only their source (a file vs. a map value) differs.
- `internal/validator` — accept a decoded schema structure (or YAML bytes)
  rather than only JSON.
- `cmd/testdata/` and `internal/validator/testdata/` — schemas and collection
  configs move to the new location/format; `cli-spec.md` "Config (v0)" updates.
- `product/decisions.md` D1, `product/domain-model.md`, `docs/configuration.md`.

## Open Questions

_None — all resolved._ For the record:

- **Discovery & format are config options, not a fixed choice.** Both schema and
  collection discovery (convention vs. explicit `defs` map) and file format (YAML
  / JSON / both) are settable per kind in `config.yaml`, defaulting to
  convention + YAML.
- **Folder naming is plural:** `.katalyst/schemas/` and `.katalyst/collections/`.
- **`init` writes a commented-template `config.yaml`** (default settings shown
  commented out) plus the two empty directories, no `.gitkeep`s.
- **No backward compatibility** — katalyst is pre-v0; the root `katalyst.yaml`
  layout is dropped outright, not supported in parallel.

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
- [ ] creates `.katalyst/`, `.katalyst/schemas/`, `.katalyst/collections/`
- [ ] writes no example schema, collection, or document
- [ ] refuses to run when `.katalyst/` already exists; writes nothing
- [ ] scaffolded `config.yaml` is in `fix` canonical form
- [ ] `check` on a freshly-`init`ed project exits 0 with no collections

Discovery (convention, the default):
- [ ] project root is the ancestor containing `.katalyst/`
- [ ] `.katalyst/schemas/{name}.yaml` is discovered as schema `{name}`
- [ ] `.katalyst/collections/{name}.yaml` is discovered as collection `{name}`
- [ ] a collection's `schema: foo` resolves to `.katalyst/schemas/foo.yaml`
- [ ] a collection file with neither `schema` nor `checks` → load error
- [ ] `path` defaults to the collection name; `pattern` defaults to `*.md`
- [ ] YAML-authored schema validates the same documents the old JSON one did
- [ ] a project with no `config.yaml` (but a `.katalyst/` dir) loads via defaults
- [ ] no `.katalyst/` in any ancestor → `ErrNotFound`, exit 2

Config options:
- [ ] `schemas.discovery: explicit` reads `defs`, ignores the directory scan
- [ ] `collections.discovery: explicit` reads `defs`, ignores the directory scan
- [ ] `explicit` with missing/empty `defs` → load error
- [ ] `format: json` scans `*.json`; `format: both` scans both, parsed by ext
- [ ] options are independent per kind (e.g. schemas `explicit`, collections `convention`)
