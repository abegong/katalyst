# Testing coverage report

A point-in-time audit of the katalyst test suite, produced for #38 (review the
tests and define an opinionated testing approach). It maps each architectural
seam to its boundary and current coverage, and records where coverage is thin.
The conventions it informs live in
[How we test](../docs/content/contributing/how-we-test.md); this report is the
supporting audit and goes stale as the suite changes. Snapshot date: 2026-06-24.

## Seams and coverage

A `katalyst check <selector>` run flows through these boundaries:

`config.Load` -> `project.Resolve` -> `storage.Items` -> `frontmatter.Parse` ->
`engine.checksFor` -> `checks.Check.Run` -> output, with `inspect.Inspect` and
`query.Predicate.Matches` as siblings.

| Seam | Boundary (identifier) | What flows across it | Test style | ~Tests | Where |
|---|---|---|---|---|---|
| Frontmatter | `frontmatter.Parse` / `Format` -> `*Document` | file bytes -> `{Meta, Body, Lines}` | unit, inline literals | 28 | `internal/frontmatter/*_test.go` |
| Config | `config.Load(start)` -> `*Config` | `.katalyst/` YAML -> resolved storage / collections / checks | component, scaffolded `.katalyst/` | 40 | `internal/config/config_test.go` |
| Storage | `storage.CollectionDefinition` (`Items` / `Unmatched` / `Reference`) | collection -> items / references | unit, temp dirs | 7 | `internal/storage/storage_test.go` |
| Project | `project.Project.Resolve(selectors)` -> `*Resolution` | selector strings -> items + collections to scan | component, scaffolded repo | 10 | `internal/project/project_test.go` |
| Engine | `engine.checksFor(collection, meta)` -> `[]checks.Check` | config + item meta -> runnable checks (schema precedence, variant routing) | component | engine_test | `cmd/engine_test.go` |
| Check | `checks.Check.Run(Context) []Violation` (+ `CollectionCheck`) | parsed doc -> violations | unit per family, via `checktest` | 54 | `internal/checks/*/` |
| Registry / library | `checks.Register` / `Build`, `CheckLibrary` / `SchemaLibrary` / `Schema` | check type <-> descriptor <-> library | parity guards | registry_test, library_test | `internal/checks/registry_test.go` |
| Validator | `jsonschema` `CompileSchema` + `Schema.Check` | schema + meta -> violations | unit, rich fixture | 9 | `internal/checks/jsonschema/jsonschema_test.go` |
| Inspector | `Source` / `CollectionInspector.Inspect(View, Params) Evidence` | view -> evidence | unit on evidence; registry parity | ~20 | `internal/inspect/*_test.go` |
| Query | `query.ParseFilter(s)` + `Predicate.Matches(meta)` | filter expr -> predicate -> bool | unit, table-driven | 25 | `internal/query/*_test.go` |
| CLI | `cmd.NewRootCmd()` end-to-end | args -> exit code + stdout / stderr | integration: snapshot text + property behavior | 121 | `cmd/*_test.go` |
| Generated docs | `cmd/gendocs` + `internal/examples` + `docs-gen-check` | engine output -> published docs | golden + drift gate | examples golden | `internal/examples/run_test.go` |
| Dogfood | `katalyst check` over `.katalyst/` in CI | the project's own docs corpus | acceptance | CI step | `.github/workflows/ci.yml` |

Test counts are approximate and current as of the snapshot date.

## CI gates beyond `go test`

- `go test -race -count=1 ./...` (race detector).
- `make build` (the Makefile entry point).
- `make docs-gen-check` (reference and example drift).
- `make docs-build` then `./bin/katalyst check` (Hugo build catches broken refs;
  dogfood validates the docs corpus).
- `go mod tidy` drift.

## Where coverage is thin

Priorities for #38 follow-on work, roughly in order:

1. **Storage error paths.** `internal/storage` is happy-path only: pattern
   matching, sorted listing, reference resolution. No coverage of permission
   errors, missing directories, or symlink loops.
2. **Inspector `Inspect()` integration.** Evidence computation is unit-tested
   (`fields`, `body`, `source`), but the `Inspect()` interface itself has a
   single integration test (`CollectionView`). Source-layer inspectors are not
   exercised end-to-end through the interface.
3. **Check composition.** Each check is unit-tested in isolation, but no test
   chains a base collection plus a matched variant plus multiple checks on one
   item, the path `engine.checksFor` actually produces.
4. **Project resolution with variants.** `Resolve` is tested for selector
   parsing and basic resolution, not for variant-predicate routing or the
   collection-scoped re-scan semantics.
5. **Config-example compilation.** Each check type ships a `ConfigExample`
   string in its descriptor; nothing asserts those examples actually parse and
   build.

## Method

Seams were identified by tracing one `katalyst check` invocation through the
packages and cataloguing the exported boundary at each hop. Coverage was read
from the `*_test.go` files per package. Both are point-in-time; regenerate this
report rather than editing counts piecemeal.
