# `item list` query — plan

> Spec: [`item list` query](./item-list-query-spec.md)
>
> **Status: not started.**

## Current State

- **`item list` does no filtering and no frontmatter parse.**
  `newItemListCmd` (`cmd/item.go:41`) takes one `<collection>` selector,
  calls `e.proj.Items(c)` (`internal/project/project.go:56`, returns
  `[]project.Item{Collection, ID, Path}` in id order), and for each item
  prints `id <tab> status` via a `tabwriter`. Status comes from
  `itemStatus(e, c, item)` (`cmd/check.go:143`), which parses the file,
  runs the resolved checks, and returns a violation count;
  `statusLabel(n)` (`cmd/item.go:319`) renders `ok` / `n errors`.
- **Config has no query settings.** `internal/config/config.go` parses
  `.katalyst/config.yaml` into `rawConfig{Schemas, Collections}`
  (`config.go:131`), resolves each collection through
  `buildCollection` (`config.go:307`) into `config.Collection`
  (`config.go:111`). There is no project- or collection-level `query:`
  block today.
- **Frontmatter exposes `Meta`, `Body`, `Lines` — but not the raw block.**
  `frontmatter.Parse` (`internal/frontmatter/frontmatter.go:66`) returns a
  `Document` (`frontmatter.go:47`) whose `Body` is a sub-slice of the input
  after the closing fence. The raw frontmatter region (`yamlBlock`,
  `frontmatter.go:86`) is computed but discarded. `--grep-in frontmatter`
  needs it.
- **Scalar typing lives in `cmd`.** `parseAssignment` (`cmd/write_validation.go:114`)
  YAML-decodes a `key=value` RHS into `any`. `parseItem`
  (`cmd/write_validation.go:18`) wraps `frontmatter.Parse`.
- **CLI tests drive the real root.** `runRoot(t, args...)`
  (`cmd/helpers_test.go:14`) builds `cmd.NewRootCmd()` (`cmd/root.go:14`),
  captures out/err, and executes. Disk work scaffolds into `t.TempDir()`
  (`AGENTS.md` Testing).

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Query settings in config | `query:` block (project + per-collection); resolve `filterTypeMismatch`/`sortMissing` into `config.Collection`; `config_test.go` |
| 2 | Raw frontmatter region | add `Document.Frontmatter` to `internal/frontmatter`; `frontmatter_test.go` |
| 3 | Query engine | new `internal/query`: parse `--filter`/`--sort`, grep, the filter→grep→sort→skip→limit pipeline; `query_test.go` |
| 4 | `item list` wiring | flags on `newItemListCmd`; assemble records; call the engine; error→exit 2; `item_test.go` |
| 5 | Docs & graduation | README, `docs/`, `cli-spec.md`; retire spec + plan |

Each phase is **tests-first internally**: write the failing test sub-step,
then the implementation. Phases 1–3 build the substrate bottom-up (config,
parser, pure engine); Phase 4 is the only one touching the user-facing
command; Phase 5 is graduation. Phases 1–3 are independent and may land in
any order, but 4 depends on all three.

## Phases

### Phase 1 — Query settings in config

**Goal:** Resolve `filterTypeMismatch` and `sortMissing` per collection,
collection-over-project-over-default.

1. **File:** `internal/config/config_test.go` *(failing first)* — scaffold a
   project with a project-level `query:` block in `.katalyst/config.yaml`
   and a collection whose YAML carries its own `query:`. Assert: a
   collection with no `query:` inherits the project default; a collection
   `query:` overrides only the keys it sets (the other falls through to
   project, then built-in); absent everywhere → built-in `skip`/`last`; an
   unknown `filterTypeMismatch`/`sortMissing` value is a load error.
2. **File:** `internal/config/config.go` — add the resolved type and field:
   ```go
   type QuerySettings struct {
       FilterTypeMismatch string // "skip" (default) | "error"
       SortMissing        string // "last" (default) | "lowest"
   }
   ```
   Add `Query QuerySettings` to `Collection` (`config.go:111`). Add
   `rawQuery struct { FilterTypeMismatch, SortMissing string }` and hang it
   off both `rawConfig` (`config.go:131`, project default) and
   `rawCollection` (`config.go:152`, per-collection, as `*rawQuery` so
   "unset" is distinguishable from "set to default").
3. **File:** `internal/config/config.go` — resolve in `buildCollection`
   (`config.go:307`): merge collection `query:` over the project `query:`
   over the built-in defaults, key by key, then validate each value against
   its allowed set (mirror `normDiscovery`/`formatExts`, `config.go:356`).
   Thread the project-level `rawQuery` into `loadCollections`/`buildCollection`
   (it currently only sees the collections block).
4. **Gate:** `go test ./internal/config/...` green.

### Phase 2 — Raw frontmatter region

**Goal:** Expose the raw frontmatter bytes so grep can target them.

1. **File:** `internal/frontmatter/frontmatter_test.go` *(failing first)* —
   assert `Parse` sets `Frontmatter` to the raw YAML block (no fences) for a
   document with frontmatter, and to `nil` when `HasFrontmatter` is false.
2. **File:** `internal/frontmatter/frontmatter.go` — add `Frontmatter []byte`
   to `Document` (`frontmatter.go:47`) and set it from `yamlBlock`
   (`frontmatter.go:86`) in the `Parse` return (`frontmatter.go:106`).
   Document it: the raw block between the fences, for text search; `Meta` is
   the parsed form.
3. **Gate:** `go test ./internal/frontmatter/...` green.

### Phase 3 — Query engine (`internal/query`)

**Goal:** A pure, table-tested package that runs the
filter→grep→sort→skip→limit pipeline over in-memory records.

1. **File:** `internal/query/query.go` *(new)* — define the engine input and
   options, decoupled from `project`/`config`/`checks`:
   ```go
   type Record struct {
       ID          string
       Status      int            // violation count; for the "status" sort key
       Meta        map[string]any // parsed frontmatter ("" when unparseable)
       Raw         []byte         // whole file
       Body        []byte
       Frontmatter []byte
   }
   type Options struct {
       Filters      []Predicate
       Greps        []*regexp.Regexp
       GrepIn       Region // RegionAll | RegionBody | RegionFrontmatter
       Sorts        []SortKey
       Skip, Limit  int
       TypeMismatch string // "skip" | "error"
       SortMissing  string // "last" | "lowest"
   }
   ```
2. **File:** `internal/query/filter.go` *(new)* + `internal/query/filter_test.go`
   *(failing first)* — `ParseFilter(string) (Predicate, error)`. Scan for the
   longest operator (`>=`,`<=`,`!=`,`=~`,`>`,`<`,`=`); a leading `!` with no
   operator is "absent", a bare field is "exists". Dot-path field lookup into
   `Meta`. YAML-scalar typing of the RHS via a local `scalar(string) any`
   helper (`yaml.Unmarshal` into `any` — duplicated from `parseAssignment`,
   not imported; see Architecture Decisions). Comma RHS on `=`/`!=` →
   in/nin. Comparison rules: numeric, string, bool; type mismatch returns
   `ErrTypeMismatch` from the predicate so the caller can skip-or-error.
   Tests cover every operator, exists/absent, dot paths, in/nin, regex, and
   mismatch.
3. **File:** `internal/query/sort.go` *(new)* + `internal/query/sort_test.go`
   *(failing first)* — `ParseSort(string) ([]SortKey, error)` (split on
   commas; leading `-` = descending; keys `id`, `status`, or a dot-path).
   `sortRecords` is stable, breaks ties by `id` ascending, and places
   missing fields per `SortMissing`. Tests: single/multi-key, descending,
   `status` key, missing `last` vs `lowest`, tie-break determinism.
4. **File:** `internal/query/query.go` + `internal/query/query_test.go`
   *(failing first)* — `Apply(recs []Record, opts Options) ([]Record, error)`
   running the pipeline in order: filter (AND; on `ErrTypeMismatch` skip or
   return a usage-style error per `opts.TypeMismatch`), grep (AND; match the
   `GrepIn` region), sort, skip, limit (`Limit<=0` = no cap). Tests assert
   ordering of stages (e.g. limit applies after sort) and that an empty
   result is a non-error empty slice.
5. **Gate:** `go test ./internal/query/...` green.

### Phase 4 — `item list` wiring

**Goal:** Expose the flags on `item list` and run the engine.

1. **File:** `cmd/item_test.go` *(failing first)* — drive `runRoot` against a
   `t.TempDir()` project with several items. Cover the spec's CLI checklist:
   `--filter` (each operator, exists/absent, in, dot path, multiple ANDed),
   `--grep` + `--grep-in` + `-i`, `--sort`/`--skip`/`--limit`, the
   configurable behaviors via both config and the `--on-type-mismatch` /
   `--sort-missing` flags, empty result → exit 0, and bad
   filter/sort/regex/negative-limit → exit 2.
2. **File:** `cmd/item.go` — add flags to `newItemListCmd` (`item.go:41`):
   `--filter` (`StringArrayVar`), `--grep` (`StringArrayVar`), `--grep-in`,
   `-i/--ignore-case`, `--sort` (`StringArrayVar`), `--skip`, `--limit`,
   `--on-type-mismatch`, `--sort-missing`. Parse `--filter`/`--sort`/`--grep`
   through the `internal/query` parsers; any parse/regexp error →
   `usageErr(...)` (exit 2, `cmd/check.go:161`). `-i` rewrites each grep to
   `(?i)` form; it does not touch filter `=~`.
3. **File:** `cmd/item.go` — build `[]query.Record` from `e.proj.Items(c)`:
   read the file once (`mustRead`, `item.go:330`), `parseItem` for `Meta`,
   `Body`, `Frontmatter`; `itemStatus` for `Status` (treat a parse error as
   today does — surface it, don't drop the item silently). Resolve the
   effective `TypeMismatch`/`SortMissing` as flag-or-`c.Query`. Call
   `query.Apply`, then print the surviving records through the existing
   `tabwriter` as `id <tab> statusLabel(Status)`. Output columns unchanged.
4. **Gate:** `go test ./cmd/...` green; `make all` green.

### Phase 5 — Docs & graduation

**Goal:** Document the flags and retire the spec.

1. **File:** `README.md` — under `katalyst item ...` (around the `item list`
   line, README:172), add the filter/grep/sort/limit flags with one example
   line each, matching the existing terse style.
2. **File:** `docs/content/reference/commands.md` — expand the `item list`
   entry with the flag reference and the filter operator table.
3. **File:** `docs/content/reference/configuration.md` — document the
   `query:` block (project + per-collection) and the resolution precedence.
4. **File:** `product/specs/cli-spec.md` — update the `item list` entry
   (`cli-spec.md:134`) to note the query flags, or link to the durable docs.
5. **Graduation:** set the spec/plan Status to **done** and delete
   `item-list-query-spec.md` + `item-list-query-plan.md` per
   `how-we-plan.md` (rationale now lives in the docs).
6. **Gate:** `make all` green; the explanation/reference docs build.

## Key Files

| File | Role |
|---|---|
| `internal/config/config.go` | `QuerySettings` type + `Collection.Query`; parse and resolve the `query:` block (edited) |
| `internal/config/config_test.go` | Query resolution + precedence coverage (edited) |
| `internal/frontmatter/frontmatter.go` | `Document.Frontmatter` raw block (edited) |
| `internal/frontmatter/frontmatter_test.go` | Raw-block assertion (edited) |
| `internal/query/query.go` | `Record`, `Options`, `Apply` pipeline (new) |
| `internal/query/filter.go` | `ParseFilter` + predicate evaluation (new) |
| `internal/query/sort.go` | `ParseSort` + stable sort with missing-placement (new) |
| `internal/query/*_test.go` | Engine unit tests (new) |
| `cmd/item.go` | `item list` flags, record assembly, engine call (edited) |
| `cmd/item_test.go` | CLI behavior coverage (edited) |
| `README.md`, `docs/content/reference/{commands,configuration}.md`, `product/specs/cli-spec.md` | Graduation targets (edited) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Engine location | New `internal/query` package, pure functions over `[]Record` | Matches the existing `internal/*` split; table-testable with no cobra/disk, per AGENTS TDD style |
| Engine input | A flat `Record` (id, status, meta, raw/body/frontmatter bytes) | Decouples the pipeline from `project`/`checks`/`frontmatter`; cmd assembles records, engine stays dependency-free |
| Scalar typing in query | Local `scalar()` helper, not importing `cmd.parseAssignment` | `cmd` is not importable; the helper is three lines of `yaml.Unmarshal` — honest duplication per AGENTS over a cross-package dep |
| Raw frontmatter | Add `Document.Frontmatter` rather than derive in cmd | The parser already has the block (`yamlBlock`); slice arithmetic against `Body` in cmd is fragile across BOM/fence handling |
| Query-setting resolution | Resolve collection-over-project-over-default at config load into `Collection.Query` | One resolution point; cmd only layers the CLI flag on top, mirroring the `--schema` override pattern |
| Type-mismatch = error | Surfaced from `Apply` as a usage error (exit 2) | Consistent with the spec's exit-code table; keeps the default (`skip`) silent |
| `-i` scope | Rewrites `--grep` patterns to `(?i)`, never filter `=~` | Spec decision; one obvious meaning for the flag, inline `(?i)` covers filter regexes |

## Out of Scope

- **Projection** (`--fields`) — output stays `id <tab> status`. Strongest
  follow-up; deferred per spec.
- **`OR`/nested boolean logic, array-index paths (`tags.0`), `$size`/`$type`**
  and other Mongo operators beyond the spec's table.
- **Extending filter/grep/sort to `collection list` / `schema list`** — this
  plan touches only `item list`.
- **Machine-readable output (`--json`), watch mode** — unchanged from
  `cli-spec.md` out-of-scope.
- **Changing `itemStatus`/check resolution** — records reuse it as-is; the
  check engine is not modified.
