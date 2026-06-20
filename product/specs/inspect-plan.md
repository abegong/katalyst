# Inspect — plan

> Spec: [Inspect — profiling a directory into a draft schema](./inspect-spec.md)
>
> **Status: planning.** Builds the inspector layer (`internal/inspect`) and the
> `inspect` command. The counterfactual half of `check` (`--try`, `check
> --json`) is **deferred to a follow-up spec** (see Out of Scope), and the agent
> orchestration that drives these instruments is the harness's job, not katalyst
> code.

## Current State

- **Checks are evaluative, registered, and self-documenting.**
  `internal/checks/checks.go` defines `Check` (`Run(Context) []Violation`),
  `Context{FilePath, Doc, Meta}`, and `RunAll`. `internal/checks/registry.go`
  holds `Descriptor`/`Family` and `Descriptors()`/`Families()`; a parity test
  asserts every dispatched kind has a descriptor, and `cmd/gendocs/main.go`
  renders `docs/content/reference/rules/` from it. There is **no descriptive
  counterpart** — nothing aggregates across a collection to *describe* it.
- **Frontmatter parsing is reusable.** `internal/frontmatter` `Parse(src)`
  returns `*Document{HasFrontmatter, Meta map[string]any, Body, Lines}`.
  `cmd/write_validation.go` `parseItem(path)` wraps read+parse.
- **Item enumeration exists.** `internal/project/project.go` `Items(c)` globs a
  collection directory with its pattern; `Unmatched(c)` lists non-matching
  files. Both stat the dir and no-op when it is absent.
- **`check` exists but isn't reused here.** `cmd/check.go` and `cmd/engine.go`
  hold the evaluative path; the counterfactual extensions to them are deferred
  (see Out of Scope), so this plan touches neither.
- **Commands attach to a constructor root.** `cmd/root.go` `NewRootCmd()` adds
  each subcommand; CLI tests drive it via `SetArgs`/`Execute` (`AGENTS.md`).

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Inspector core | `internal/inspect`: `Corpus`, `Inspector`, `Evidence`, registry + parity test, first two inspectors |
| 2 | Inspector set | remaining inspectors across all families, incl. `frontmatter_shape` grouping |
| 3 | Rendering | Markdown (default) + JSON renderers over `[]Evidence` |
| 4 | `inspect` command | wire to root; `--inspector` / `--json` / `-o`; read-only profile |
| 5 | Docs & graduation | glossary, explanation pages, `cli-spec.md` scope move, README, generated inspector reference; retire spec/plan |

Phases 1–4 build and expose the descriptive engine bottom-up; Phase 5
graduates. Each phase is **tests-first internally**: write the failing test
sub-step, then the code that makes it pass — a single up-front scaffolding phase
doesn't work in Go, where a test referencing an unbuilt symbol breaks the whole
package's compilation.

## Phases

### Phase 1 — Inspector core

**Goal:** A parsed `Corpus`, an `Inspector` interface, an `Evidence` value, and
a self-documenting registry — proven by two inspectors.

1. **File:** `internal/inspect/inspect_test.go` *(new, failing first)* — build a
   tiny in-memory `Corpus` of files (some with frontmatter, one parse failure)
   and assert: `WalkParse` reports total/parsed/failed counts matching `n`;
   `ObjectFieldFrequency` present-counts each key correctly over `n`. Use
   `package inspect_test` (external), stdlib only.
2. **File:** `internal/inspect/corpus.go` *(new)* — define
   `File{Rel string; Doc *frontmatter.Document; ParseErr error}` and
   `Corpus{Scope string; Files []File}`. Add `Load(root string) (Corpus, error)`
   that walks `*.md` under `root` and parses each via `frontmatter.Parse`,
   capturing per-file `ParseErr` rather than aborting. This is the **parse-once**
   substrate every inspector reads (spec: determinism + caching).
3. **File:** `internal/inspect/inspect.go` *(new)* — define
   `Evidence{Inspector, Scope string; N int; Data map[string]any}` and
   `Inspector interface { Name() string; Inspect(Corpus) Evidence }`. A single
   `Data map[string]any` payload keeps one renderer pair serving every
   inspector (no per-inspector structs).
4. **File:** `internal/inspect/registry.go` *(new)* — mirror
   `internal/checks/registry.go`: `Descriptor{Name, Family, Summary}`,
   `Families()` (structural / object / markdown / filesystem, ordered), and
   `Descriptors()`. Add `All() []Inspector` returning instances, and
   `ByName(name string) (Inspector, bool)`.
5. **File:** `internal/inspect/inspectors_object.go` *(new)* — implement
   `WalkParse` (structural) and `ObjectFieldFrequency` (object) against
   `Corpus`. `Inspect` returns `Evidence` with counts only — **no
   recommendations** (no `→ required`), per the spec invariant.
6. **File:** `internal/inspect/registry_test.go` *(new, failing first)* —
   parity: every `All()` inspector has a `Descriptors()` entry and vice versa;
   names are unique. Mirrors `internal/checks/registry_test.go`.
7. **Gate:** `go test ./internal/inspect/...` green.

### Phase 2 — Inspector set

**Goal:** The full initial inspector set across all four families.

1. **File:** `internal/inspect/inspectors_*_test.go` *(new, failing first)* —
   per inspector, assert evidence shape against a small fixed corpus:
   `ObjectFieldValues` (cardinality + value set under a size cap),
   `ObjectFieldTypes` (mixed-type key reported as mixed, not first-wins),
   `ObjectFieldNumericRange`, `ObjectFieldStringLength`,
   `MarkdownHeadingShape` (single-H1 rate, H1==title rate, level-jump
   presence), `MarkdownSections` (recurring headings + frequency),
   `MarkdownCodeFences`, `FilesystemNaming` (casing histogram, spaces,
   extensions), and `FrontmatterShape`.
2. **File:** `internal/inspect/inspectors_object.go`,
   `inspectors_markdown.go` *(new)*, `inspectors_filesystem.go` *(new)* —
   implement the object/markdown/filesystem inspectors. Markdown inspectors
   read `Doc.Body`; filesystem inspectors read `File.Rel`. Reuse existing
   heading/fence parsing from `internal/checks` where it is exported; otherwise
   keep the scan local to the inspector (don't export check internals just for
   this).
3. **File:** `internal/inspect/inspectors_structural.go` *(new)* —
   `FrontmatterShape`: per file, the **sorted key-set** is the fingerprint
   identity; it also groups files that share an identical fingerprint and emits
   observed per-key types as *adjacent* evidence (spec: key-set identity, types
   alongside). Grouping is deterministic and lives here — the only aggregation
   in the initial set.
4. **File:** `internal/inspect/registry.go` — register every new inspector in
   `All()` and `Descriptors()` (parity test from Phase 1 guards this).
5. **Gate:** `go test ./internal/inspect/...` green.

### Phase 3 — Rendering

**Goal:** Render `[]Evidence` to Markdown (default) and JSON, from one source.

1. **File:** `internal/inspect/render_test.go` *(new, failing first)* — assert
   `RenderMarkdown` groups records by family and includes `n` and each
   record's counts; `RenderJSON` round-trips a record with `inspector`,
   `scope`, `n`, `evidence`; both derive from the same `[]Evidence` (one source
   of truth). Markdown assertions match on substrings, not byte-exact layout.
2. **File:** `internal/inspect/render.go` *(new)* — `RenderMarkdown([]Evidence)
   string` and `RenderJSON([]Evidence) ([]byte, error)`. JSON marshals the
   `Evidence` struct (`Data` → `evidence`). Markdown is a projection: a section
   per record, grouped by family in `Families()` order.
3. **Gate:** `go test ./internal/inspect/...` green.

### Phase 4 — `inspect` command

**Goal:** `katalyst inspect <path>` runs inspectors over a scope and renders
their evidence; writes nothing.

1. **File:** `cmd/inspect_test.go` *(new, failing first)* — drive
   `NewRootCmd()` with `inspect <tmpdir>` over a scaffolded corpus in
   `t.TempDir()`. Assert: default output is Markdown and names the inspectors;
   `--json` emits valid JSON for the same run; `--json` and default derive from
   the same evidence; `--inspector object_field_frequency` narrows to one;
   `-o <file>` writes bytes identical to stdout; the command writes no files
   under the scope (read-only); exit 0 on a readable path, exit 2 on a missing
   path.
2. **File:** `cmd/inspect.go` *(new)* — `newInspectCmd()`: positional `<path>`
   (required), flags `--inspector` (repeatable; default all), `--json`, and
   `-o/--output <file>`. Body: `inspect.Load(path)` → select inspectors via
   `ByName` → run each → `RenderMarkdown` (default) or `RenderJSON`. Write to
   `-o` when set, else `cmd.OutOrStdout()`. Reuse the `exitUsage`/`exitOK`
   codes from `cmd/check.go`. **No `--write`, no `--strictness`** (spec).
3. **File:** `cmd/root.go` — add `newInspectCmd()` to `NewRootCmd()`'s
   `AddCommand` list.
4. **Gate:** `go test ./cmd -run TestInspect` green.

### Phase 5 — Docs & graduation

**Goal:** Reconcile durable docs, generate the inspector reference, retire the
spec.

1. **File:** `cmd/gendocs/main.go` — extend to also render
   `docs/content/reference/inspectors/` from `inspect.Descriptors()` /
   `inspect.Families()`, mirroring the rules generation. Run `make docs-gen`.
   (Generated pages are never hand-edited — `AGENTS.md`.)
2. **File:** `docs/content/explanation/general-model.md` — note that an
   **inspector** realizes the long-listed `aggregate` operation: a descriptive
   read that reports a distribution, the dual of a check.
3. **File:** `docs/content/explanation/domain-model.md` — add the inspector
   concept and the `inspect` data flow; absorb the locked decisions
   (evidence-not-verdicts, determinism dividing line, Markdown default) into the
   prose, per `how-we-plan.md` (no separate decisions log).
4. **File:** `docs/content/reference/glossary.md` — add *inspector*, *evidence*,
   *corpus*, *fingerprint*.
5. **File:** `product/specs/cli-spec.md` — move `inspect` (was `infer`/`profile`)
   out of the v0 "out of scope" list, pointing at the shipped surface. Leave
   `check --json` deferred with the counterfactual follow-up.
6. **File:** `docs/content/reference/commands.md`,
   `docs/content/how-to/`, `README.md` — document `inspect`.
7. **Graduation:** set the spec Status to **done**, run the
   `how-we-plan.md` graduation checklist, delete spec + plan.
8. **Gate:** `make all` and `make docs-gen` clean; no stale references.

## Key Files

| File | Role |
|---|---|
| `internal/inspect/corpus.go` | Parse-once `Corpus`/`File`; `Load(root)` (new) |
| `internal/inspect/inspect.go` | `Inspector`, `Evidence` (new) |
| `internal/inspect/registry.go` | `Descriptor`/`Families`/`All`/`ByName`, mirrors checks registry (new) |
| `internal/inspect/inspectors_*.go` | The inspector set across families (new) |
| `internal/inspect/render.go` | `RenderMarkdown` (default) + `RenderJSON` (new) |
| `internal/inspect/*_test.go` | Inspector, registry parity, render coverage (new) |
| `cmd/inspect.go` | `inspect` command: flags, run, render (new) |
| `cmd/inspect_test.go` | `inspect` CLI behavior (new) |
| `cmd/root.go` | Attach `newInspectCmd()` (edited) |
| `cmd/gendocs/main.go` | Generate inspector reference (edited) |
| `docs/.../general-model.md`, `domain-model.md`, `glossary.md`, `commands.md`, `README.md`, `cli-spec.md` | Graduation targets (edited) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Inspector package | New `internal/inspect`, sibling to `internal/checks` | Inspectors are the descriptive dual of checks, not checks; a sibling mirrors the registry pattern without polluting the `Check` interface |
| Parse once | Command builds a `Corpus`; inspectors are pure `Inspect(Corpus) Evidence` | Spec's determinism + caching: repeated runs/renders never re-read disk |
| Evidence payload | One `Data map[string]any`, not per-inspector structs | A single Markdown/JSON renderer pair serves every inspector; new inspectors add no renderer code |
| Markdown default | Render Markdown unless `--json` | Spec: agents handle Markdown well, humans read it for free |
| Evidence carries no verdicts | Counts + `n` only | Keeps threshold judgment in the agent; evidence stays trustable |
| Fingerprint identity | Sorted key-set; types adjacent | Spec: cheaper, clusters aggressively, types available to split a group |
| Inspector reference | Generated from the registry, like rules | A new inspector cannot ship undocumented (mirrors the checks invariant) |

## Out of Scope

- **Counterfactual `check` (`--try`, `check --json`).** Deferred to a follow-up
  spec. The grammar is recorded in `inspect-spec.md`, but this plan touches
  neither `cmd/check.go` nor `cmd/engine.go` and adds no ephemeral-collection
  path. The inspectors stand on their own without it.
- **The agent orchestration loop.** Forming hypotheses, picking thresholds,
  clustering near-miss fingerprint groups, naming collections, and writing the
  draft `.katalyst/` files are the harness's job — not katalyst code. This plan
  ships the instruments, not the profiler.
- **`inspect --write` / schema generation.** `inspect` never writes a schema
  (spec). `-o` saves a copy of the report, nothing more.
- **Parameterized inspectors.** The initial set takes no descriptor options;
  the only aggregation is `frontmatter_shape`'s identical-fingerprint grouping.
  A `field:`-style parameter mechanism is deferred until an inspector needs it.
- **Evidence format versioning.** Deferred with the rest of katalyst's
  versioning question (spec).
- **Non-filesystem corpora and non-markdown items.** `inspect` reads `*.md`
  under a directory, consistent with v0's filesystem-only scope.
- **Fuzzy clustering.** Only exact identical-fingerprint grouping is in scope;
  near-miss boundary calls stay with the agent.

## Test checklist

The spec's [Test checklist](./inspect-spec.md) is the contract for this branch.
The pending tests are scaffolded across phases: inspector + evidence (1–2),
rendering (3), and the `inspect` command (4). The counterfactual `check`
checklist is deferred with that follow-up spec.
