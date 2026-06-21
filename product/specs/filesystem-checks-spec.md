# Filesystem checks: expanded & revised library

> **Status: planning.** Revises the six ad-hoc `filesystem_*` checks into a
> smaller, composable set of *name/path* checks, adds the conventions users
> actually reach for (regex escape hatch, suffix, length, charset, depth,
> directory-name rules), and opens a second tier of **collection-scoped**
> checks (uniqueness, required index files, referenced-file existence) that the
> current per-item check model cannot express. Pre-v0: existing kinds are
> renamed/folded outright, not kept in parallel.

## Overview

The `filesystem` family today is six unrelated checks bolted on one at a time:
`filesystem_filename_matches_slug`, `filesystem_extension_in`,
`filesystem_filename_kebab_case`, `filesystem_no_spaces_in_path`,
`filesystem_parent_dir_in`, `filesystem_filename_prefix`. Each is a bespoke
struct with a bespoke message, and several are special cases of a more general
idea (kebab-case is one case style; prefix is half of affix; no-spaces is one
charset rule). Meanwhile the checks users ask for first when they see this
family — "every slug is unique," "every folder has an `_index.md`," "the cover
image actually exists" — cannot be written at all, because a check sees one
item at a time and nothing about its siblings.

This spec does three things:

1. **Revise** the per-item name/path checks into a coherent set built on two
   shared axes — a **target** (what part of the path) and a **rule** (what must
   hold) — folding the four special-case checks into two general ones.
2. **Expand** the per-item set with the missing conventions: an anchored
   `regex` escape hatch, `affix` (prefix *and* suffix), name `length`, path
   `charset`, path `depth`, and directory-name-vs-field matching.
3. **Open** a `collection`-scoped tier for cross-item integrity, gated behind a
   new check interface. This is the load-bearing design decision and is called
   out as the primary open question.

The comparison points are [ls-lint][ll] (rich per-name rules: case styles +
`regex`, plus a `.dir` target and OR-composition) and [folderslint][fl]
(declarative allowed-directory structure). ls-lint maps onto Tier 1/2;
folderslint maps onto the Tier 3 structural checks.

[ll]: https://ls-lint.org
[fl]: https://github.com/denisraslov/folderslint

## Value

- **Fewer kinds, more coverage.** Seven authored checks today cover ~four
  distinct ideas. Collapsing case/affix/charset onto shared rules means one
  documented surface (`style: kebab|snake|...`) covers what would otherwise be
  five or six `*_case` kinds, and adding a style is a new enum value, not a new
  check + struct + Descriptor + dispatch case.
- **The high-value checks become expressible.** Uniqueness and
  referenced-file existence are the guarantees that make a *metadata* tool
  trustworthy — they catch the broken-link and duplicate-slug classes that
  per-field validation structurally can't. They require a one-time interface
  investment, after which each such check is small.
- **Parity with the tools in this space.** A user evaluating katalyst next to
  ls-lint should not find katalyst missing camelCase, a regex fallback, or
  directory-name rules; a user comparing to folderslint should find the
  required-structure idea present.

## Current State

- **Two scopes today: configured per collection, evaluated per item.** Checks
  are declared only in a collection's `.katalyst/collections/{name}.yaml` under
  `checks:` — there is no project-wide or per-item declaration. But execution is
  item-by-item: `cmd/check.go` resolves selectors into a **flat, selector-
  narrowed** list (`res.Items`, each carrying its `Collection`), then for each
  item calls `engine.checksFor(item.Collection, meta)` and
  `checks.RunAll(Context{…}, checkList)`. Nothing holds a whole collection's
  items at once during the check pass; items are processed independently. The
  one collection-granularity operation, `Unmatched` (files not matching the
  pattern), is deliberately **not** a check — it's a separate scan in the
  command, which shows the codebase already reaches for a side-channel when it
  needs collection-level information.
- **Per-item, stateless `Check` contract.** `internal/checks/checks.go` defines
  `Check.Run(ctx Context) []Violation` where `Context = {FilePath string, Doc
  *frontmatter.Document, Meta map[string]any}`. A check sees exactly one item;
  there is no handle to the collection, its other items, or the filesystem
  beyond `FilePath`. `RunAll` flattens violations across a slice of checks for
  one item.
- **The markdown layer does not parse links.** `frontmatter.Document` exposes
  `Body []byte` (raw bytes) plus parsed headings/fences; there is no link or
  image extractor. Any body-link check must write one.
- **Six filesystem checks** in `internal/checks/filesystem.go`, each a struct
  with a `Run`. `filename_kebab_case` hardcodes `^[a-z0-9]+(?:-[a-z0-9]+)*$`;
  `no_spaces_in_path` is `strings.Contains(path, " ")`; `filename_prefix` is
  `strings.HasPrefix`; `parent_dir_in`/`extension_in` are membership tests;
  `filename_matches_slug` compares the basename to a frontmatter field.
- **Wiring is four-touchpoint.** A `kind:` string is parsed into a flat
  `config.Check` struct in `config.normalizeCheck` (which also validates
  required keys), dispatched to a concrete `checks.X{}` in `cmd/engine.go`, and
  documented by a `Descriptor` in `internal/checks/registry.go`.
  `registry_test.go` enforces that every dispatched kind has a Descriptor, so a
  check cannot ship undocumented; `cmd/gendocs` renders
  `docs/content/reference/rules/filesystem/` from the Descriptors.
- **`fix` is a formatter, not a repairer.** `cmd/fix.go` canonicalizes
  frontmatter (sorted keys, block style, single trailing newline) and "never
  invents semantic values." No check has an auto-fix today; filesystem
  violations are reported, never repaired.
- **Pre-v0, no back-compat.** The project-layout spec established that katalyst
  drops old surfaces outright rather than supporting them in parallel. Renaming
  or folding existing `filesystem_*` kinds is therefore in-bounds.

## Design

### Two axes: target × rule

Every per-item filesystem check answers two questions: *which slice of the
path* and *what must be true of it*. Making both explicit is what lets the
special cases collapse.

**Target** — selected by a `target` key, default `filename`:

| `target` | Value tested |
|---|---|
| `filename` | basename without extension (e.g. `my-note`) |
| `filename-ext` | basename with extension (e.g. `my-note.md`) |
| `parent-dir` | immediate parent directory name |
| `path-segments` | every directory segment from the collection root down, plus the basename — each tested independently |

`path-segments` is **inclusive** (resolved, was Q3): it tests every directory
segment *and* the basename, so it's the ls-lint `.dir`-plus-file idea as one
rule — `My Folder/note.md` can't pass a rule the file alone satisfies. The
common ask ("kebab everywhere") is one line; when directories and files need
*different* shapes (e.g. `PascalCase/` folders, `kebab.md` files), compose the
disjoint `parent-dir` and `filename` targets instead.
Targets are resolved relative to the **collection root**, never above it, so a
project living under `/Users/abe/Documents/...` is not penalized for the
ancestors katalyst doesn't own.

### Tier 1 — revised per-item checks

These replace existing kinds. Names use `name` (not `filename`) because the
`target` key now decides what "name" means.

#### `filesystem_name_case` — *replaces* `filesystem_filename_kebab_case`

```yaml
checks:
  - kind: filesystem_name_case
    style: kebab          # kebab | snake | screaming-snake | camel | pascal | point | lower
    target: filename      # optional; default filename
```

| Field | Required | Default | Meaning |
|---|---|---|---|
| `style` | yes | — | One of the seven styles below. |
| `target` | no | `filename` | Path slice to test. |

Styles, with their anchored patterns (mirroring ls-lint's vocabulary):

| `style` | Pattern intent |
|---|---|
| `lower` | letters lowercased; non-letters ignored |
| `kebab` | `^[a-z0-9]+(?:-[a-z0-9]+)*$` |
| `snake` | `^[a-z0-9]+(?:_[a-z0-9]+)*$` |
| `screaming-snake` | `^[A-Z0-9]+(?:_[A-Z0-9]+)*$` |
| `camel` | `^[a-z][a-zA-Z0-9]*$` |
| `pascal` | `^[A-Z][a-zA-Z0-9]*$` |
| `point` | `^[a-z0-9]+(?:\.[a-z0-9]+)*$` |

`kebab` with `target: filename` is exactly today's `filename_kebab_case`.

#### `filesystem_name_matches_field` — *replaces* `filesystem_filename_matches_slug`

```yaml
checks:
  - kind: filesystem_name_matches_field
    field: slug           # optional; default slug
    transform: none       # optional; none | slugify
    target: filename
```

The selected target must equal the frontmatter `field`, optionally after a
`transform` (`slugify` lowercases and kebab-cases the field value before
comparison, so `title: "My First Note"` matches `my-first-note.md`). `field:
slug, transform: none, target: filename` is today's `filename_matches_slug`.

#### `filesystem_name_affix` — *replaces* `filesystem_filename_prefix`

```yaml
checks:
  - kind: filesystem_name_affix
    prefix: book-         # at least one of prefix/suffix required
    suffix: -draft
    target: filename
```

`prefix: X` alone is today's `filename_prefix`.

#### `filesystem_path_charset` — *replaces* `filesystem_no_spaces_in_path`

```yaml
checks:
  - kind: filesystem_path_charset
    deny: [" "]           # deny OR allow, not both
```

`deny: [" "]` is today's `no_spaces_in_path`. `allow:` instead specifies the
only permitted characters (a whitelist), useful for "ASCII letters, digits,
`-`, `/` only." Operates on the collection-relative path string as a whole, so
it spans separators.

#### `filesystem_extension_in` — *kept as-is*

No change; it already fits the model (it's a `filename-ext`-flavored membership
test, but the existing kind is clear and widely the first one configured).

#### `filesystem_parent_dir_in` — *kept as-is*

Membership test on `parent-dir`. Retained; `name_case`/`name_affix` with
`target: parent-dir` cover the *shape* of a parent dir, while this covers the
*allowed set*.

### Tier 2 — new per-item checks

| Kind | Fields | Purpose |
|---|---|---|
| `filesystem_name_regex` | `pattern` (req), `target` | Anchored `^pattern$` over the target — the escape hatch for anything the named styles don't cover. Mirrors ls-lint `regex:`. |
| `filesystem_name_length` | `min`/`max` (≥1 req), `target` | Bound the character length of the target (guards the 255-byte filename limit and overlong slugs). |
| `filesystem_path_depth` | `min`/`max` (≥1 req) | Bound directory nesting **relative to the collection root** (flat collection ⇒ `max: 0`). The folderslint structural idea in scalar form. |
| `filesystem_parent_dir_matches_field` | `field` (req) | Parent directory name must equal a frontmatter field (e.g. `category: recipes` ⇒ lives under `recipes/`). |

### Tier 3 — collection-scoped checks (new interface)

The checks below cannot be answered from one `Context`; they need every item in
the collection at once. This is the architectural decision of the spec.

**Proposed interface.** Add a second optional interface alongside `Check`:

```go
// CollectionCheck validates a concern across all items in a collection.
type CollectionCheck interface {
    RunCollection(ctx CollectionContext) []Violation
}

type CollectionContext struct {
    Root  string         // collection root directory
    Items []ItemContext  // one per item (FilePath + Meta), in resolved order
}
```

The engine runs item `Check`s as today, then runs `CollectionCheck`s once per
collection over the gathered item set. A check kind implements one interface or
the other, never both. `Violation` is unchanged; collection-scoped violations
carry the offending item's `FilePath` (and the partner path, in the message,
for collisions).

**Selector wrinkle (from how scoping works today).** `res.Items` is flat and
**narrowed by the selector** — `katalyst check notes/dune` resolves to a single
item. A `CollectionCheck` cannot consume that list: a uniqueness verdict is only
correct against the collection's *full* item set. So the engine must, for each
collection touched by the selection, **re-scan the whole collection** (via
`project.Items(collection)`) to build `CollectionContext`, independent of how
the selector narrowed the per-item pass. The visible consequence — worth a docs
note — is that a single-item selector still reads every sibling when a
collection-scoped check is configured. Per-item checks keep honoring the
selector exactly as today; only the collection-scoped pass widens.

The initial Tier-3 checks:

| Kind | Fields | Purpose |
|---|---|---|
| `filesystem_unique_filename` | — | No two items in the collection share a basename. |
| `filesystem_unique_field` | `field` (req) | No two items share a value for `field` (slug/id uniqueness). |
| `filesystem_index_file_required` | `name` (opt, default `_index.md`) | Every subdirectory containing items also contains `name`. The folderslint "this dir must exist/contain" idea; ls-lint's `exists`. |
| `filesystem_referenced_files_exist` | `fields` (req) | Each listed **frontmatter** field holds a path (string or list) that resolves to a real file. Catches dead cover-image / attachment references. |

`unique_field` overlaps conceptually with the `object` family but is
intrinsically cross-item, so it lives here, scoped to the collection.

`referenced_files_exist` is **frontmatter-only** (resolved, was Q4): it reads
the named `fields`, treats each value as a path, and resolves it **relative to
the item's own directory** (how a human reads a relative path written in that
file) before `os.Stat`. Body markdown links are deliberately *not* in scope
here — they're a body-content concern with different resolution and skip rules
(external `http(s)://`, `mailto:`, `#anchors`) and require a link extractor the
markdown layer doesn't have yet. They become a separate future
**`markdown_relative_links_resolve`** check, keeping each kind's semantics
single. (`referenced_files_exist` is technically per-item — it needs no
siblings — but is grouped with Tier 3 as a path-integrity check; it can ship on
the per-item `Check` contract.)

### Composition (out of scope, resolved)

ls-lint allows `camelCase | PascalCase` (OR). The natural katalyst shape would
be a generic combinator — `kind: any_of` wrapping a list of checks, passing if
any child passes — not per-check `|` parsing. This is **out of scope** for this
spec: it's cross-cutting (touches every family, not just filesystem) and a
config-language change that deserves its own spec. Tracked as a follow-up.

### `fix` interaction

Filesystem checks remain **check-only**. The single safe auto-fix in this
family is renaming a file, and a rename silently breaks every inbound link and
changes the item's identity — exactly the "invent a value" hazard `fix`
already refuses. The spec does not add filesystem auto-fixes; `name_case` /
`name_matches_field` violations are reported with the expected name in the
message so the user (or their editor) can rename deliberately.

### Code touch points

- `internal/config/config.go` — retire the four folded `CheckKind` constants;
  add the new ones; extend `rawCheck`/`Check` with `Style`, `Target`,
  `Transform`, `Prefix`, `Suffix`, `Pattern`, `Allow`, `Deny`, `Fields`,
  `Name` fields; add `normalizeCheck` cases with key validation (e.g.
  `name_affix` requires prefix or suffix; `path_charset` rejects both
  `allow` and `deny`). New validation messages follow the error grammar in
  `cmd/AGENTS.md` (lowercase, no trailing period, `%q` around kind/field
  values). Check **violations** keep the `path:line: /pointer: message`
  diagnostic format, which that doc exempts from the prose rules.
- `internal/checks/filesystem.go` — replace the four folded structs with
  `NameCase`, `NameMatchesField`, `NameAffix`, `PathCharset`; add Tier-2
  structs; add a shared `resolveTarget(ctx, target) []string` helper.
- `internal/checks/collection.go` (new) — `CollectionCheck`,
  `CollectionContext`, and the four Tier-3 implementations.
- `cmd/engine.go` — dispatch the new kinds; gather per-collection item
  contexts and invoke `CollectionCheck`s after the per-item pass.
- `internal/checks/registry.go` — Descriptors for every new kind (parity is
  test-enforced); a `Scope` field on `Descriptor` (`item` | `collection`) so
  generated docs can note which checks run per-collection.
- `cmd/rules.go` / `cmd/rules_test.go` — `rules` (now split into `rules list`
  and `rules show <kind>`) is registry-driven, so new kinds appear
  automatically; surface the Tier 3 `Scope` field in `runRulesDetail` (the
  `rules show` readout). Its tests count dynamically and unmarshal a field
  subset, so they stay green.

## Documentation updates

- **Generated reference** — `make docs-gen` regenerates
  `docs/content/reference/rules/filesystem/` (do not hand-edit): the four folded
  pages disappear, new per-kind pages appear, and the family `_index.md`
  updates. Tier 3 pages carry `Scope: collection`.
- **User docs (Hugo)** — `docs/content/how-to/configure-rules.md` and
  `docs/content/reference/configuration.md` name the folded kinds and must move
  to the `target × rule` model and new kind names. The domain model that
  references check kinds currently lives in
  `docs/content/explanation/domain-model.md` — see the note below.
  `docs/content/reference/glossary.md` gains "target" and "collection-scoped
  check."
- **Developer docs** — root `AGENTS.md` (and `internal/checks` package docs if
  added): record the `target × rule` model, the `resolveTarget` helper, the
  `Context.CollectionRoot` addition, and the `CollectionCheck` tier + its
  full-collection re-scan. No `.cursor/skills/` changes.
- **Docs-location risk (confirm at graduation).** main currently carries *both*
  `docs/content/explanation/domain-model.md` and
  `docs/content/deep-dives/core-concepts.md`; the `explanation/` page is the one
  that references check kinds today, but the docs convention (write-spec skill)
  treats `docs/deep-dives/` as canonical. Before graduating, confirm which page
  is authoritative and update that one, rather than guessing now.

## Open Questions

- **Q1 — Phasing of Tier 3 (open).** Tiers 1–2 are a pure refactor+extension
  within the existing per-item model; Tier 3 adds the `CollectionCheck`
  interface, a second engine pass, and the full-collection re-scan described
  above. Recommendation: ship Tiers 1–2 first (one plan/phase), then Tier 3
  (second phase) once the interface and the re-scan behavior are reviewed. The
  scoping facts that informed this are now folded into Current State and the
  Tier 3 design.
- **Q2 — Generic `any_of` combinator. Resolved: out of scope.** A config-
  language change touching every family; deserves its own spec. Tracked as a
  follow-up.
- **Q3 — `path-segments` target. Resolved: inclusive.** It tests every
  directory segment *and* the basename. The "different shapes for dirs vs files"
  case is served by composing the disjoint `parent-dir` and `filename` targets.
- **Q4 — `referenced_files_exist` scope. Resolved: frontmatter-only.** Reads the
  named `fields`, resolving each path relative to the item's own directory. Body
  markdown links are split out into a future `markdown_relative_links_resolve`
  check (different resolution + skip rules, and needs a link extractor the
  markdown layer lacks).

## Rejected alternatives

- **Keep one check per case style** (`filename_kebab_case`,
  `filename_snake_case`, …). Mirrors ls-lint's rule names but multiplies
  structs, Descriptors, and dispatch cases for what is one parameterized idea;
  a `style` enum documents the whole set on one page.
- **Per-check `|` OR-syntax** (`style: camel | pascal`). Encodes composition
  inside one field's parser, invisible to other checks; a top-level `any_of`
  combinator (Q2) is the general, discoverable form.
- **Stuff cross-item checks into the per-item `Check`** by giving every check a
  back-reference to the collection. Pollutes the common path with state the
  vast majority of checks never use, and makes ordering/side-effects implicit.
  A separate `CollectionCheck` keeps the per-item contract clean.
- **A folderslint-style declarative `structure:` block** instead of discrete
  checks. Powerful for whole-tree shape, but it is a parallel config dialect
  with its own matching semantics; `path_depth` + `parent_dir_in` +
  `index_file_required` cover the same ground inside the one check vocabulary
  users already learn.
- **Auto-fix filesystem violations by renaming.** Rejected for the same reason
  `fix` won't inject frontmatter values: a rename is a semantic, link-breaking
  act, not a deterministic reformat.

## Test checklist (what the pending tests assert)

Tier 1 (revised, per-item):
- [ ] `name_case` accepts/rejects each `style` against a basename
- [ ] `name_case` with `target: parent-dir` tests the parent, not the file
- [ ] `name_case` with `target: path-segments` flags a bad mid-path segment *and* a bad basename (inclusive)
- [ ] `name_case style: kebab, target: filename` matches old kebab-case behavior
- [ ] `name_matches_field` with `transform: none` equals old matches-slug
- [ ] `name_matches_field` with `transform: slugify` matches a slugified title
- [ ] `name_affix` requires at least one of prefix/suffix (load error otherwise)
- [ ] `name_affix prefix: book-` matches old filename-prefix behavior
- [ ] `path_charset deny: [" "]` matches old no-spaces behavior
- [ ] `path_charset` rejects configuring both `allow` and `deny`
- [ ] `extension_in` / `parent_dir_in` behavior unchanged

Tier 2 (new, per-item):
- [ ] `name_regex` anchors the pattern (`^...$`) and respects `target`
- [ ] `name_length` enforces min/max; requires at least one bound
- [ ] `path_depth` counts depth relative to collection root; `max: 0` ⇒ flat
- [ ] `parent_dir_matches_field` passes/fails on dir-vs-field equality

Tier 3 (collection-scoped):
- [ ] engine runs `CollectionCheck`s once per collection after the item pass
- [ ] a single-item selector still re-scans the whole collection for the
      collection-scoped pass (uniqueness verdict unaffected by selector)
- [ ] per-item checks still honor the selector (only the collection pass widens)
- [ ] `unique_filename` flags two items sharing a basename, names both paths
- [ ] `unique_field` flags duplicate `field` values across items
- [ ] `index_file_required` flags a subdirectory missing `_index.md`
- [ ] `referenced_files_exist` flags a frontmatter path that resolves nowhere
- [ ] `referenced_files_exist` resolves paths relative to the item's directory
- [ ] `referenced_files_exist` accepts both a string and a list field value

Registry / docs:
- [ ] every new kind has a Descriptor (parity test stays green)
- [ ] retired kinds are absent from constants, dispatch, and Descriptors
- [ ] `make docs-gen` regenerates the filesystem reference cleanly
- [ ] Descriptor `Scope` distinguishes item vs collection checks in docs
