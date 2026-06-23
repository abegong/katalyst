# CLI test strategy — hybrid snapshots + property tests

> **Status: planning.** Defines how the `cmd/` test suite splits between
> fixture/snapshot tests for user-facing text contracts and property tests for
> behavior, and the reusable snapshot harness that makes the snapshot half
> cheap to write and review. Implements [issue #53](https://github.com/abegong/katalyst/issues/53).

## Overview

The `cmd/` suite proves two different things about the CLI: that its output
*text* reads a certain way, and that its *behavior* (exit codes, side effects,
precedence, query semantics) is correct. Today both are asserted the same way —
scattered `strings.Contains` probes — which is weak for text (a passing
`Contains` says nothing about layout, ordering, or wording around the match)
and noisy for behavior. This spec adopts a hybrid: snapshot fixtures own the
text contracts, focused property tests own the semantics, and a single
snapshot harness backs every fixture comparison.

## Value

- **Stronger text contracts.** A golden file pins the whole output — columns,
  ordering, blank lines, punctuation — so a layout regression fails loudly
  instead of slipping past a substring match.
- **Reviewable copy.** Help text and list/show output are product copy
  (`cmd/AGENTS.md` § "Help text copy"). A reviewer reads the fixture as plain
  text in the diff rather than reverse-engineering it from assertions.
- **Less brittle behavior coverage.** Property tests stop carrying incidental
  wording, so a copy tweak touches one fixture instead of a dozen `Contains`
  lines.

## Current state

One snapshot pattern already exists and works:

- `cmd/root_test.go` and `cmd/help_snapshot_test.go` compare full stdout against
  fixtures under `cmd/testdata/help/*.txt`, loaded through `mustHelpFixture`
  (`cmd/fixtures_test.go`) which reads from a `//go:embed` FS so the fixture
  survives the per-test `chdir` into `t.TempDir()`.
- The comparison is hand-rolled in each test (`if stdout != want { t.Errorf(…) }`)
  and there is **no update mechanism** (fixtures are edited by hand) and **no
  normalization** (help output is path-free, so none was needed).

Everything else asserts text with partial probes:

- `cmd/collection_test.go` — `collection list` / `get` checked with
  `strings.Contains` for name, path, count, schema columns.
- `cmd/schema_test.go` — `schema list` ordering/line-count and `schema show`
  content probed inline.
- `cmd/check_types_test.go`, `cmd/inspectors_test.go` — `list` / `show` /
  `list --family|--layer` text checked with `Contains`, `lineContaining`,
  breadcrumb/intro/sibling probes. (Their `--json` tests already parse and
  assert shape — those stay; see Design.)
- `cmd/item_test.go` — `item list` status table and `item get`
  default/`--frontmatter`/`--body` probed inline; the `--filter`/`--grep`/
  `--sort`/`--skip`/`--limit` tests assert *semantics*.
- `cmd/inspect_test.go` — the Markdown report (`# Inspection report:`,
  `### document_shape`) probed with `Contains`.
- `cmd/check_test.go` — stderr diagnostics (`/year` pointer, `path:line:`,
  `unmatched`, the `em dash` writing-tell warning) probed inline, alongside the
  exit-code assertions they belong with.

Two facts shape the harness design:

1. **Tests `chdir` into temp dirs.** `chdir` / `writeProject` / `storageLocal`
   (`cmd/helpers_test.go`) scaffold a `.katalyst/` tree in `t.TempDir()` and
   `chdir` in. Reads must not depend on the cwd (hence `//go:embed`); writes (an
   update mode) must target the *source* `testdata/`, not the temp cwd.
2. **Some output embeds the temp path.** `check` diagnostics print the absolute
   item path (`<tmpdir>/notes/bad.md:3:`), and `inspect <path>` prints the
   inspected path in its `# Inspection report: <scope>` header
   (`internal/inspect/render.go:40`). A snapshot of either is non-deterministic
   without path normalization. Pure-text surfaces (help, `collection list`,
   `schema list`, `check-types`/`inspectors`, `item list`/`get`) are already
   deterministic — paths in them are project-relative.

> **Aside — recently moved pieces.** Collections now live *inside* a storage
> instance file (`.katalyst/storage/local.yaml`, scaffolded by `storageLocal`),
> not the `.katalyst/collections/*.yaml` layout `cli-spec.md` still describes;
> and `check` now runs body-text/markdown lint that emits advisory warnings
> (the `markdown_writing_tells` em-dash tell) to stderr. The harness consumes
> these as-is — it scaffolds via `storageLocal` and snapshots the lint
> diagnostics — but neither is changed by this work.

## Design

### The snapshot harness

One helper in `cmd/helpers_test.go` (or a new `snapshot_test.go`), replacing the
hand-rolled comparison and generalizing `mustHelpFixture`:

```go
// snapshot asserts got equals the fixture at testdata/snapshots/<name>,
// after applying any normalizers to got. With -update, it rewrites the
// fixture instead of asserting. name is a slash path, e.g. "collection/list".
func snapshot(t *testing.T, name, got string, norm ...func(string) string)
```

Properties:

- **Reads via `//go:embed`** (`testdata/snapshots/**`), so a comparison works
  after `chdir`, exactly as `mustHelpFixture` does today.
- **Writes via the source path** under `-update`. The package dir is recovered
  with `runtime.Caller` (the test file's directory), *not* `os.Getwd`, because
  the cwd is a temp dir mid-test. `-update` is a `flag.Bool` registered once in
  the test package (`go test ./cmd -run TestX -update`) — the canonical Go
  golden-file pattern, no new dependency.
- **Normalizers** are `func(string) string` applied to `got` before compare and
  before writing, so fixtures store the normalized form. The common one,
  `normTmp(dir)`, rewrites the test's temp dir to a stable token (`<project>`).
  Stderr/inspect snapshots pass it; pure-text snapshots pass nothing.
- **Failure output** shows a unified got/want diff and the `-update` hint.

`mustHelpFixture`, `root_test.go`, and `help_snapshot_test.go` collapse into
`snapshot(...)` calls — the issue's "merge root help assertion into centralized
snapshot harness" item. Help fixtures move `testdata/help/` → a `help/`
subgroup under `testdata/snapshots/` (or stay put and the harness reads both;
moving is cleaner and is the recommendation).

### Fixture layout

```
cmd/testdata/snapshots/
  help/root.txt              # was testdata/help/root-noargs.txt
  help/check.txt             # was testdata/help/check-help.txt …
  collection/list.txt
  collection/get.txt
  schema/list.txt
  schema/show-book.txt
  check-types/list.txt
  check-types/list-family-markdown.txt
  check-types/show-object_required_field.txt
  inspectors/list.txt
  inspectors/show-object_fields.txt
  item/list.txt
  item/get.txt
  item/get-frontmatter.txt
  inspect/source-report.txt  # normTmp applied
  check/invalid-pointer.txt  # stderr, normTmp applied
  check/unmatched.txt        # stderr
  check/writing-tell.txt     # stderr warning voice
```

One fixture per surface; stdout and stderr are separate fixtures (a test that
asserts both calls `snapshot` twice). `testdata/AGENTS.md` gains a `snapshots/`
section and the per-fixture table moves there.

### What becomes a snapshot

Text contracts — full stdout (or normalized stderr) pinned as a fixture:

| Surface | From | Fixture group |
|---|---|---|
| root help, every `--help` | `root_test.go`, `help_snapshot_test.go` | `help/` |
| `collection list`, `collection get` | `collection_test.go` | `collection/` |
| `schema list`, `schema show` | `schema_test.go` | `schema/` |
| `check-types list`, `list --family`, `show` | `check_types_test.go` | `check-types/` |
| `inspectors list`, `list --layer`, `show` | `inspectors_test.go` | `inspectors/` |
| `item list`, `item get` (+`--frontmatter`/`--body`) | `item_test.go` | `item/` |
| `inspect <path>` Markdown report | `inspect_test.go` | `inspect/` (normTmp) |
| selected `check` stderr diagnostics | `check_test.go` | `check/` (normTmp) |

### What stays a property test

Behavior the fixture can't express, asserted directly:

- **Exit codes** `0/1/2` via `errors.As(&coded)` / `coded.Code()` — every
  current usage stays.
- **Precedence:** `--schema` override, inline `schema:` key, variant routing /
  first-match / exhaustive (`check_test.go`).
- **Side effects:** `item add` writes / refuses overwrite / writes-nothing on
  validation failure; `update` merges and leaves body untouched; `delete`
  removes (`item_test.go`).
- **Collection-scoped semantics:** single-item selector still rescans the whole
  collection and names both colliding files (`check_test.go`).
- **JSON outputs:** `check-types --json`, `inspectors --json`, `inspect --json`
  stay parse-and-assert (unmarshal, count vs registry, snake_case keys,
  non-empty fields). See Open Question 3.
- **Query semantics:** `item list` `--filter`/`--grep`/`--sort`/`--skip`/
  `--limit`, type-mismatch, sort-missing, bad-query (`item_test.go`).

### Hybrid tests keep both halves

A test that today asserts an exit code *and* a message keeps the exit-code
assertion and moves only the wording to a snapshot. Example:
`TestCheck_invalidItem_exit1WithPointer` keeps `Code()==1` and replaces its
`/year` + `path:line` probes with `snapshot(t, "check/invalid-pointer", stderr,
normTmp(dir))`. **A snapshot existing for a surface does not justify deleting a
behavior assertion** — only redundant *text* probes are removed.

### Cleanup (pruned because a snapshot subsumes them)

Removed outright: `TestCollectionList_showsNamePathCountSchema`,
`TestCollectionGet_showsDetail`, `TestSchemaShow_printsSchemaContents`,
`TestCheckTypesShow_showsDetail`, `TestCheckTypesShow_showsFamilyContextAndSiblings`,
`TestInspectorsShow_showsDetail`, and similar pure-`Contains` text probes whose
every assertion is a substring the snapshot now pins. `root_test.go` and
`help_snapshot_test.go` are rewritten onto the harness, not deleted.

Kept even though they overlap a snapshot: ordering/structure tests that assert a
*computed property* rather than literal bytes (e.g. `schema list` "sorted, two
lines"; `check-types list` "families in `Families()` order") — but reframed to
lean on the snapshot for layout and keep only the invariant. When the invariant
is fully implied by the fixture, the test goes.

## Open Questions

1. **Update mechanism: `-update` flag vs `UPDATE_SNAPSHOTS` env var.**
   **Context.** Golden fixtures need a regeneration path so a deliberate copy
   change is one command, not a hand-edit of N files. Two stdlib-only options.
   **Choices & tradeoffs.** A registered `flag.Bool("update", …)` is the
   canonical Go pattern (`go test ./cmd -run TestThing -update`), discoverable
   and scoped per run, but adds one package-level flag var. An env var
   (`UPDATE_SNAPSHOTS=1 go test ./cmd`) needs no flag registration and is
   trivially settable in a Make target, but is less idiomatic and easy to leave
   exported in a shell. **Recommendation:** the `-update` flag — it's the
   convention Go reviewers expect and reads clearly in CI logs. Your call.

2. **Fixture granularity for `item list` query results.**
   **Context.** `item list` has a plain listing (snapshot candidate) and the
   `--filter`/`--grep`/`--sort`/`--limit` family (semantics). The query tests
   currently assert *which ids appear in what order* — which is also exactly
   what a snapshot pins. **Choices & tradeoffs.** Snapshot each query result and
   the test becomes "ran the query, output matches" — strong on layout, but the
   *intent* ("`--limit 2` returns the first two by sort key") is no longer
   legible in the test, it's implicit in the fixture. Keep them as property
   tests and layout drift in the query table goes uncaught there (the plain
   `item list` snapshot still guards the table shape). **Recommendation:**
   snapshot only the plain `item list`; keep the query-flag tests as property
   tests asserting ids/order in code, since the semantic is the point. Worth a
   second look if the query table layout proves fragile.

3. **Golden JSON for `--json` outputs?**
   **Context.** `check-types --json` / `inspectors --json` / `inspect --json`
   currently unmarshal and assert shape. A canonicalized golden JSON file would
   also be reviewable as a diff. **Choices & tradeoffs.** Golden JSON pins the
   exact wire bytes (key order, formatting) and reads well in review, but
   couples the test to incidental serialization and re-raises the determinism
   question (registry order is stable, but timestamps/paths in `inspect --json`
   are not). Parse-and-assert tolerates formatting and targets the contract
   (every descriptor present, snake_case keys, no empty fields). The issue
   explicitly lists JSON under "keep as property tests." **Recommendation:**
   keep `--json` as property tests; do **not** add golden JSON. Listed so the
   decision is on the record, not silent.

## Documentation updates

- **`cmd/testdata/AGENTS.md`** — add a `snapshots/` section: the harness, the
  group layout, the `-update` (or env) workflow, and how to review a fixture
  diff. Fold/replace the current `help/` table.
- **`cmd/AGENTS.md`** — a "Testing the CLI" note: snapshot for text contracts,
  property test for behavior, hybrid tests keep both; point at the harness and
  the normalizer for path-bearing output.
- **Root `AGENTS.md` § Testing** — one line under "Style" pointing CLI text
  contracts at the snapshot harness, consistent with "standard library only"
  and the existing `//go:embed` fixture guidance.
- **`cmd/testdata/AGENTS.md` "Adding a fixture"** — extend the steps to cover a
  snapshot fixture (run with `-update`, review the generated file, embed group).
- User-facing Hugo docs: **none** — this is an internal test-strategy change.

## Test checklist (harness self-coverage)

- [ ] `snapshot` asserts equality against an embedded fixture and passes on match
- [ ] mismatch prints a got/want diff and the update hint, fails the test
- [ ] `-update` rewrites the source fixture (verified by writing to the file the
      next read loads), and writes the *normalized* form
- [ ] reads succeed after a test `chdir`s into `t.TempDir()` (embed, not cwd)
- [ ] `normTmp(dir)` rewrites the temp path to `<project>` in stderr and
      `inspect` snapshots; a pure-text snapshot needs no normalizer
- [ ] migrated surfaces (help, `collection`, `schema`, `check-types`,
      `inspectors`, `item list`/`get`, `inspect`, `check` stderr) each have a
      fixture and a passing assertion
- [ ] every retained property test still guards its exit code / side effect /
      precedence / query semantic after the text probes are removed
