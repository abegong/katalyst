# v0 implementation plan — bringing the CLI up to `cli-spec.md`

> Working doc. Derives the build order from [`cli-spec.md`](cli-spec.md).
> The spec's "Test checklist" (spec lines 216–256) is the acceptance
> criteria: each phase lands the tests that prove it.

## Decisions baked in

- **Verb name:** `check` (and `fix`). Confirmed against the spec's open
  naming call. `validate`/`fmt` are removed (hard rename, no alias — spec
  "Changes from the current CLI").
- **Inline `schema:` precedence stays** (decisions.md D2): for `check` and
  the validate-on-write path, resolution is `--schema` flag → inline
  `schema:` key → the collection's configured schema/checks.
- **Config:** named `collections:` map *replaces* the anonymous `rules:`
  list. No dual-format support.
- **Engine is stable.** `internal/checks`, `internal/frontmatter`,
  `internal/validator` need no structural change. This is a **config
  restructuring + command reorganization** around a new collection/item
  domain layer.

Module path: `github.com/abegong/katalyst`.

## What stays vs. what changes

| Area | Today | v0 | Disposition |
|---|---|---|---|
| Engine (`internal/checks`) | 17 checks, `RunAll` | same | **reuse as-is** |
| Frontmatter parse + line map | `internal/frontmatter` | same | reuse |
| Schema validation | `internal/validator` | same | reuse |
| Frontmatter format | `internal/frontmatter/format.go` | same | reuse (powers `fix`) |
| Config shape | `rules: []` + doublestar globs | `collections: {}` named map | **rewrite** |
| Addressing | filesystem paths | `<collection>/<item>` selectors | **new layer** |
| Commands | `validate fmt create read update delete init schema` | `check fix init schema` + `collection` + `item` nouns | **restructure** |

## New concepts to build

### 1. Selector grammar (spec "Selector grammar")
A parser/resolver that maps a string selector to targets, by depth:
- `""` → all collections.
- `<collection>` → one collection (all items).
- `<collection>/<item>` → one item.

Rules to enforce:
- First segment is **always** a collection; a bare token is never an item.
- Blessed verbs (`check`, `fix`) accept a selector at **any depth** and
  accept **multiple** selectors.
- Noun commands expect a **fixed depth** (stated per command); wrong depth
  → exit 2.
- Unknown collection / unknown item → exit 2.

### 2. Collection / item domain layer
Today there is no collection or item abstraction — everything is flat,
path-based. Introduce a small package (proposed `internal/project`) that
sits on top of `config`:

- `Project` — loaded from a `*config.Config`; holds the named collections.
- `Collection` — `Name, Dir, Pattern, Checks []config.Check` plus the
  resolved object schema name.
- `Item` — `Collection, ID` with `Path()` → `<dir>/<id><ext>` (reverse
  resolution: `notes/dune` → `notes/dune.md`).
- `Resolve(selectors ...string) ([]Target, error)` — the one entry point
  the commands call; returns items (and the collection context) or a
  usage error (exit 2) on unknown/bad-depth selectors.
- `Items(collection)` — globs `Dir`/`Pattern`; also reports **unmatched
  references** (files in the dir that don't match `pattern`) for `check`
  to surface as errors (spec "Config" + decisions.md D2).

This package is where the spec's "Selector resolution" and "Config" test
groups live.

## Phased work

Each phase is independently compilable and testable. Order minimizes churn:
the data model first, then the resolver, then commands on top.

### Phase 1 — Config: `collections:` map
Files: `internal/config/config.go`, `internal/config/config_test.go`.

- Replace `rawConfig.Rules` with `Collections map[string]rawCollection`
  (`path`, `pattern` (default `*.md`), `schema`, `checks`).
- New `Collection` struct on `Config`; drop `Rule`/`RuleFor`/`Match`
  glob machinery (or keep `Match` private only if still needed).
- Reuse `normalizeCheck` verbatim to turn a collection's `schema:`/`checks:`
  into `[]Check` (the `schema:` shorthand → a `CheckObject`, exactly as
  rules do today).
- Validation at load: every collection `schema` references a known
  `schemas:` entry; `path` non-empty; default the pattern.
- Keep `Schemas`, `SchemaPath`, `SchemaNames`, nearest-ancestor `find`,
  symlink handling — all unchanged.

Lands spec tests: **Config** group (named collections load; id→path
resolution; reverse resolution; every collection schema known).

### Phase 2 — Selector + domain layer
Files: new `internal/project/` (`project.go`, `selector.go`, tests).

- Implement `Project`, `Collection`, `Item`, `Resolve`, `Items`, and
  unmatched-reference detection.
- Depth + unknown-target validation returning the exit-2 usage error
  (reuse `cmd`'s `exitError`/`usageErr`, or mirror it — see note below).

Lands spec tests: **Selector resolution** group (empty→all,
collection→items, item→one, bare token = collection, unknown→2,
wrong-depth→2).

> Note: `exitError`/`usageErr` currently live in `cmd` (`validate.go`).
> Move them to a tiny shared spot (e.g. `internal/exit` or keep in `cmd`
> and have the resolver return typed sentinel errors the commands map to
> exit codes). Decide in Phase 2; low effort either way.

### Phase 3 — `check` (replaces `validate`)
Files: rename `cmd/validate.go` → `cmd/check.go`; rework `resolver`.

- `Args`: zero-or-more **selectors** (not paths). No selector → whole
  project. Resolve via `internal/project`.
- Reuse `validateFile` almost verbatim — same per-item output contract:
  `<path>: OK` or `<path>:<line>: /<pointer>: <message>`, ancestor-line
  fallback via `lookupLine`.
- Checks now come from the **collection** (not a glob rule). Keep object
  schema precedence: `--schema` → inline `schema:` → collection.
- **Unmatched files** in a collection directory → error line + exit 1
  (new semantics vs today's "no rule matched").
- Exit: `0` all valid · `1` any failure/unmatched · `2` usage/IO.

Lands spec tests: **`check`** group (OK/exit0, error format/exit1,
ancestor-line fallback, unmatched file, `--schema` override).

### Phase 4 — `fix` (replaces `fmt`)
Files: rename `cmd/fmt.go` → `cmd/fix.go`.

- `Args`: selectors (any depth, multiple). No selector → whole project.
- Body of work is today's `fmt`: `frontmatter.Format` (sorted top-level
  keys, block style, single trailing newline, body bytes verbatim).
- `--check`: write nothing, print would-change paths, exit 1 if any.
- **D3 guard:** never inject values for missing required keys; only
  apply deterministic, lossless normalization.
- Exit: `0` clean/fixed · `1` (`--check`) pending · `2` usage/IO.

Lands spec tests: **`fix`** group (normalize + preserve body, idempotent,
`--check` behavior, never injects).

### Phase 5 — `collection` noun
Files: new `cmd/collection.go` (parent + `list`, `get`).

- `collection list` → name, directory, item count, schema. Exit 0/2.
- `collection get <collection>` → path, pattern, schema/checks, item
  count. Depth = 1; wrong depth → 2; unknown → 2.

Lands spec tests: **`collection`** group (list fields; get detail).

### Phase 6 — `item` noun
Files: new `cmd/item.go` (parent), re-home `create/read/update/delete`
logic under it; add `list`.

- `item list <collection>` → id + status (`ok` / `n errors`). Depth 1.
  Reuses the Phase-3 check path per item for status.
- `item get <collection>/<item> [--frontmatter|--body]` — **default is
  frontmatter AND body** (change from today's `read`, which dumps raw
  bytes). Depth 2; not found → 2.
- `item add <collection>/<item> [key=value ...]` (was `create`):
  frontmatter + empty body; YAML-scalar typing of values; refuse
  overwrite (2); validate-on-write (default on), `--no-validate`,
  `--schema`. Exit 0/1/2.
- `item update <collection>/<item> key=value ...` (was `update`): merge
  keys, body untouched, validate result; `--no-validate`, `--schema`.
  `--unset` is out of scope.
- `item delete <collection>/<item> ...` (was `delete`): one or many;
  missing → 2.
- Reuse `cmd/write_validation.go` helpers and the existing `key=value`
  YAML-scalar parser.

Lands spec tests: **`item` CRUD** group.

### Phase 7 — Root wiring + `init` template
Files: `cmd/root.go`, `cmd/init.go`.

- Root now attaches: `init`, `check`, `fix`, `collection`, `item`,
  `schema`. Remove `validate`, `fmt`, `create`, `read`, `update`,
  `delete`.
- `init` scaffolds a `collections:` config (one collection `notes`), one
  schema under `schemas/`, one example item under `notes/`. Still refuses
  to overwrite. Globals (`--version`, `completion`, `--help`) are Cobra
  defaults — unchanged.

### Phase 8 — Test fixtures + checklist sweep
Files: `cmd/testdata/**`, `cmd/fixtures_test.go`, all `*_test.go`.

- Migrate `testdata/configs/*.yaml` from `rules:` to `collections:`.
- Reorganize fixture markdown into per-collection directories.
- Rename/rewrite `validate_test.go`/`fmt_test.go`/`crud_test.go` to the
  new command surface; add `collection`/`item list`/selector tests.
- Final pass: walk every box in the spec's Test checklist and confirm a
  test asserts it. CI (`go vet`, race `go test`, `go build`) green.

## Suggested commit sequence
1. config: named collections map (Phase 1)
2. project: selector + item resolution layer (Phase 2)
3. cmd: check (Phase 3)
4. cmd: fix (Phase 4)
5. cmd: collection noun (Phase 5)
6. cmd: item noun + CRUD re-home (Phase 6)
7. cmd: root wiring + init template (Phase 7)
8. test: fixtures + spec checklist sweep (Phase 8)

## Risks / watch-items
- **Unmatched semantics shift.** Today "unmatched" = no glob rule matched
  a path. v0 = a file *inside a collection dir* not matching `pattern`.
  Make sure the old behavior isn't silently carried over.
- **`item get` default change.** `read` dumps raw bytes; `get` defaults to
  parsed frontmatter + body. The `--frontmatter`/`--body` narrowing is new.
- **Single vs. multiple collections.** v0 ships the single-collection case
  but the collection is always explicit/addressable — don't special-case
  "one collection" into implicit behavior.
- **`schema:` directive stripping.** `check` strips the inline `schema:`
  key before validation (so `additionalProperties: false` schemas don't
  reject self-describing docs). Preserve this in the rewrite.
- **Out of scope (don't drift in):** the storage layer / non-FS backends, `diff`,
  `query`, `infer`, `migrate`, `--json`, `--unset`, bulk-add, watch.
