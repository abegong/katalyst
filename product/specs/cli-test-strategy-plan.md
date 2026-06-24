# CLI test strategy — plan

> Spec: [CLI test strategy](./cli-test-strategy-spec.md)
>
> **Status: implementing.** All five phases landed, one commit each; `make all`
> is green. Spec is resolved (no open questions).

## Current State

- **A working snapshot pattern, hand-rolled.** `cmd/root_test.go` and
  `cmd/help_snapshot_test.go` compare full stdout against `cmd/testdata/help/*.txt`
  via `mustHelpFixture` (`cmd/fixtures_test.go:24`), which reads a
  `//go:embed testdata/help/*.txt` FS (`cmd/fixtures_test.go:21`) so the fixture
  survives the per-test `chdir`. Each comparison is inline
  (`if stdout != want { t.Errorf(…) }`); there is no update flag and no
  normalization.
- **Text contracts asserted with partial probes.** `cmd/collection_test.go`,
  `cmd/schema_test.go`, `cmd/check_types_test.go`, `cmd/inspectors_test.go`,
  `cmd/item_test.go`, `cmd/inspect_test.go` check list/show/report output with
  `strings.Contains` / `lineContaining` against substrings.
- **Behavior already isolated where it counts.** Exit codes go through
  `errors.As(&coded)` / `coded.Code()`; `--json` tests unmarshal and assert
  shape; `item list` query-flag tests assert ids/order; `check` precedence and
  side-effect tests stand alone.
- **Path-bearing output.** `check` diagnostics print `<tmpdir>/…:line:`
  (`cmd/check_test.go:85`); `inspect <path>` prints the scope in
  `# Inspection report: %s` (`internal/inspect/render.go:40`). Both need
  normalization to snapshot; every other text surface is project-relative and
  already deterministic.
- **Scaffolding helpers.** `cmd/helpers_test.go` holds `runRoot`, `chdir`,
  `writeProject`, `storageLocal`, `writeConfigDir`; tests `chdir` into
  `t.TempDir()`. Collections live in `.katalyst/storage/local.yaml` via
  `storageLocal`.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Snapshot harness | `snapshot()`, `normTmp()`, `-update` flag, self-coverage tests; relocate `testdata/help/` and rewrite `root_test.go`/`help_snapshot_test.go` onto it |
| 2 | `collection` + `schema` | Migrate `collection list`/`get` and `schema list`/`show` text tests to snapshots; prune subsumed probes |
| 3 | `check-types` + `inspectors` | Migrate `list`/`list --family|--layer`/`show` text tests; prune subsumed probes; leave `--json` property tests untouched |
| 4 | `item`, `inspect`, `check` stderr | Snapshot plain `item list` + `item get` variants, the `inspect` Markdown report (normTmp), and the error-voice stderr diagnostics (normTmp); keep query-flag/exit-code/side-effect assertions |
| 5 | Docs | `cmd/testdata/AGENTS.md`, `cmd/AGENTS.md`, root `AGENTS.md` |

Phase 1 ships the harness and proves it on the already-golden help surface, so
later phases are pure migration. Phases 2-4 are independent migrations in the
issue's order; each deletes only text probes a fixture now subsumes and keeps
every behavior assertion. Phase 5 records the convention.

Snapshot tests are still tests-first: write the `snapshot(t, name, got)` call
(fixture absent → test fails), generate the fixture with `go test ./cmd -run … -update`,
**review the generated file as the contract**, then re-run without `-update` to
confirm it asserts. "Review the fixture" is the gate, not the generation.

## Phases

### Phase 1 — Snapshot harness

**Goal.** One `snapshot()` helper backs every fixture comparison, with an
update flag and a path normalizer, proven on the existing help fixtures.

1. **File:** `cmd/snapshot_test.go` (new). Add the harness:
   - `var updateSnapshots = flag.Bool("update", false, "rewrite snapshot fixtures")`
     (registered once for the test binary).
   - `func snapshot(t *testing.T, name, got string, norm ...func(string) string)`:
     apply each `norm` to `got`; resolve the fixture path as
     `filepath.Join(pkgDir(), "testdata/snapshots", filepath.FromSlash(name))`
     where `pkgDir()` derives the package dir from `runtime.Caller` (not
     `os.Getwd`, which is a temp dir mid-test); under `-update`,
     `os.MkdirAll`+`os.WriteFile` the normalized `got` and return; otherwise read
     the embedded fixture (see step 2) and `t.Errorf` a got/want diff plus the
     `re-run with -update` hint on mismatch.
   - `func normTmp(dir string) func(string) string`: returns a closure replacing
     every occurrence of `dir` with `<project>`.
2. **File:** `cmd/fixtures_test.go`. Change the help embed to the snapshot tree:
   `//go:embed testdata/snapshots` → `var snapshotFixtures embed.FS`; the
   `snapshot()` reader uses `snapshotFixtures.ReadFile("testdata/snapshots/"+name)`.
   Remove `mustHelpFixture` (its callers move to `snapshot()` in step 4). Keep
   the schema-fixture embeds.
3. **File:** `cmd/testdata/snapshots/help/*.txt` (new). Move the ten
   `testdata/help/*.txt` fixtures here, renaming to the new scheme
   (`root-noargs.txt` → `help/root.txt`, `check-help.txt` → `help/check.txt`, …);
   `git mv` so history follows. Delete `testdata/help/`.
4. **File:** `cmd/root_test.go`, `cmd/help_snapshot_test.go`. Rewrite onto the
   harness: `snapshot(t, "help/root", stdout)` and, in the table test,
   `snapshot(t, "help/"+tc.name, stdout)`. Keep the empty-stderr assertions.
5. **File:** `cmd/snapshot_test.go` (same new file). Add harness self-coverage:
   a passing match against a tiny committed fixture; a mismatch path that records
   an error (run the assert body with a deliberately wrong `got` through a
   `testing.T` stub or a captured-failure helper); `normTmp` rewrites a temp path
   to `<project>`; an `-update` round-trip (write to a scratch dir via an
   injectable path seam, read it back). Cover the spec's harness checklist.

### Phase 2 — `collection` + `schema`

**Goal.** `collection list`/`get` and `schema list`/`show` text contracts are
fixtures; their behavior assertions stay.

1. **File:** `cmd/collection_test.go`. Replace the body of
   `TestCollectionList_showsNamePathCountSchema` and `TestCollectionGet_showsDetail`
   with `snapshot(t, "collection/list", stdout)` /
   `snapshot(t, "collection/get", stdout)` over a fixed `writeConfigDir` project
   plus one committed item, then delete the `strings.Contains` probes. Keep
   `TestCollectionGet_unknown_exit2` and `TestCollectionGet_wrongDepth_exit2`
   (exit-code property tests) unchanged.
2. **File:** `cmd/schema_test.go`. `TestSchemaShow_printsSchemaContents` →
   `snapshot(t, "schema/show-book", stdout)` (book schema is static JSON, no
   normalization). For `TestSchemaList_printsSortedNamesAndPaths`, add
   `snapshot(t, "schema/list", stdout)` and drop the per-line `HasPrefix`/`Contains`
   probes (the fixture pins order and paths); keep `TestSchemaList_noConfig` and
   `TestSchemaShow_unknownName` (exit-code / error-mentions-name) unchanged.
3. **File:** `cmd/testdata/snapshots/collection/*.txt`,
   `cmd/testdata/snapshots/schema/*.txt` (new). Generate with `-update`; review.

### Phase 3 — `check-types` + `inspectors`

**Goal.** The `list`/`show`/family/layer text surfaces are fixtures; `--json`
and ordering-invariant tests stay property tests.

1. **File:** `cmd/check_types_test.go`. Snapshot the text surfaces:
   `check-types list` → `check-types/list`; `list --family markdownBodyText`
   → `check-types/list-family-markdown`; `show object_required_field` →
   `check-types/show-object_required_field`; `show object_field_enum` (breadcrumb
   + family intro + siblings) → `check-types/show-object_field_enum`. Delete
   `TestCheckTypesShow_showsDetail` and `TestCheckTypesShow_showsFamilyContextAndSiblings`
   (fully subsumed). **Keep** `TestCheckTypes_listsEveryTypeGroupedByFamily` (it
   asserts the `Families()`-order invariant against the live registry, not
   literal bytes), the `--json` tests, `TestCheckTypesList_unknownFamily_exit2`,
   `TestCheckTypesShow_unknown_exit2`, and `TestCheckTypes_bare_printsHelpNotList`.
2. **File:** `cmd/inspectors_test.go`. Mirror Phase 3.1: snapshot
   `inspectors list`, `list --layer collection`, `show object_fields`,
   `show document_shape` (breadcrumb). Delete `TestInspectorsShow_showsDetail`
   and the show-context probe bodies subsumed by the fixture; keep the
   `--layer`/registry-coverage `--json` tests, the layer-order invariant test,
   the exit-2 tests, and `TestInspectors_bare_printsHelpNotList`.
3. **File:** `cmd/testdata/snapshots/check-types/*.txt`,
   `cmd/testdata/snapshots/inspectors/*.txt` (new). Generate; review.

### Phase 4 — `item`, `inspect`, `check` stderr

**Goal.** The remaining text surfaces (item listing/printing, the inspect
report, and the error-voice diagnostics) are fixtures; query semantics, exit
codes, and side effects stay property tests.

1. **File:** `cmd/item_test.go`. Snapshot the plain `item list` (the status
   table from `TestItemList_showsIdsAndStatus`) → `item/list`, and
   `item get` default / `--frontmatter` / `--body`
   (`TestItemGet_defaultPrintsFrontmatterAndBody`, `TestItemGet_frontmatterAndBodyFlags`)
   → `item/get`, `item/get-frontmatter`, `item/get-body`. **Do not** snapshot the
   query-flag tests (`_filter`/`_grep`/`_sortAndLimit`/…) or the exit/side-effect
   `add`/`update`/`delete` tests — leave them as-is.
2. **File:** `cmd/inspect_test.go`. `TestInspect_rawPathRunsSourceLayer` →
   `snapshot(t, "inspect/source-report", stdout, normTmp(dir))` over a fixed
   `inspectRepo` tree; keep the "not JSON" guard and the `--json` /
   collection-layer property tests. The report header
   (`# Inspection report: <project>`) normalizes the scope path.
3. **File:** `cmd/check_test.go`. Add a small set of stderr snapshots for the
   error/warning voice, alongside the existing exit-code assertions (do not
   remove those): `TestCheck_invalidItem_exit1WithPointer` → keep `Code()==1`,
   add `snapshot(t, "check/invalid-pointer", stderr, normTmp(dir))`;
   `TestCheck_unmatchedFileInCollectionDir_isError` → keep `Code()==1`, add
   `snapshot(t, "check/unmatched", stderr, normTmp(dir))`;
   `TestCheck_writingTells_warnButPass` → keep the stdout-OK and no-error
   assertions, add `snapshot(t, "check/writing-tell", stderr, normTmp(dir))` for
   the advisory `warning: … em dash` voice. Leave the variant/precedence/
   collection-scoped tests as property tests (their stderr is incidental to the
   semantic).
4. **File:** `cmd/testdata/snapshots/{item,inspect,check}/*.txt` (new).
   Generate; review — especially the normalized path tokens.

### Phase 5 — Docs

**Goal.** The snapshot/property split and the harness workflow are written down
where the next contributor will look.

1. **File:** `cmd/testdata/AGENTS.md`. Replace the `help/` table with a
   `snapshots/` section: the harness (`snapshot`, `normTmp`, `-update`), the
   group layout, the generate-then-review workflow, and how to read a fixture
   diff. Update the "Adding a fixture" steps to cover a snapshot fixture
   (write the test, `-update`, review, the embed is the whole tree so no per-file
   `//go:embed`).
2. **File:** `cmd/AGENTS.md`. Add a "Testing the CLI" note under the existing
   guidance: snapshot text contracts, property-test behavior, hybrid tests keep
   both halves, and point path-bearing output at `normTmp`.
3. **File:** `AGENTS.md` (root, § Testing → Style). One line directing CLI text
   contracts at the snapshot harness, consistent with "standard library only"
   and the `//go:embed` fixture guidance already there.

## Key Files

| File | Role |
|---|---|
| `cmd/snapshot_test.go` (new) | The harness: `snapshot()`, `normTmp()`, `-update` flag, self-coverage |
| `cmd/fixtures_test.go` | Embed `testdata/snapshots`; drop `mustHelpFixture` |
| `cmd/testdata/snapshots/**` (new) | All golden fixtures, grouped by command |
| `cmd/root_test.go`, `cmd/help_snapshot_test.go` | Rewritten onto `snapshot()` |
| `cmd/collection_test.go`, `cmd/schema_test.go` | Phase 2 migration |
| `cmd/check_types_test.go`, `cmd/inspectors_test.go` | Phase 3 migration |
| `cmd/item_test.go`, `cmd/inspect_test.go`, `cmd/check_test.go` | Phase 4 migration |
| `cmd/testdata/AGENTS.md`, `cmd/AGENTS.md`, `AGENTS.md` | Phase 5 docs |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Fixture reads vs writes | Read via `//go:embed`; write (`-update`) via `runtime.Caller`-derived package dir | Reads must survive the per-test `chdir` into `t.TempDir()` (why help fixtures already embed); writes must hit the source tree, and `os.Getwd` is a temp dir mid-test |
| Regeneration trigger | `-update` `flag.Bool` | Canonical Go golden-file pattern; stdlib-only; discoverable and scoped per run (spec Rejected alternatives) |
| Path normalization | `normTmp(dir)` rewrites temp dir → `<project>`, applied before compare and write | Only `check`/`inspect` embed the temp path; storing the normalized form keeps fixtures stable and reviewable |
| One fixture per stream | stdout and stderr are separate files; a hybrid test calls `snapshot` twice | Streams are separate contracts; keeps the error-voice fixtures isolated and small |
| Prune scope | Delete only text probes a fixture fully subsumes; keep every exit-code/side-effect/precedence/query/invariant assertion | The snapshot strengthens *text* coverage; it does not encode behavior, so behavior tests must remain (spec "Hybrid tests keep both halves") |

## Documentation updates

Carried from the spec, landing in Phase 5: `cmd/testdata/AGENTS.md` (snapshots
section + adding-a-fixture steps), `cmd/AGENTS.md` (testing-the-CLI note), root
`AGENTS.md` (§ Testing one-liner). No user-facing Hugo docs — this is an
internal test-strategy change.

## Out of Scope

- **Golden JSON for `--json` outputs.** Stays parse-and-assert (spec Rejected
  alternatives).
- **Snapshotting `item list` query results.** The `--filter`/`--grep`/`--sort`/
  `--skip`/`--limit` tests stay property tests.
- **Changing CLI output, config layout, or the `storageLocal` scaffolding.**
  This is a test-only change; any output edit is a separate change that
  regenerates the affected fixtures.
- **A shared cross-package `testutil`.** The harness lives in `cmd` test code
  only, per the root `AGENTS.md` "helpers are per-file / no `testutil`" rule.
- **`fix` / `init` output snapshots.** Not called out in the issue scope; add
  later if their text contracts grow.
