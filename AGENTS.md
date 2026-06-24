# AGENTS.md

Conventions for anyone, human or AI, making changes in this repo.

For *what* the project does and how to use the CLI, see [`README.md`](README.md)
and the user docs under `docs/`.
For *why* the design is the way it is, see the deep-dive pages under
`docs/deep-dives/`, rationale lives on each topic's page; there is no central
decisions log.
For *how we plan and document* changes, see
[`docs/contributing/how-we-plan.md`](docs/contributing/how-we-plan.md) and
[`docs/contributing/how-we-document.md`](docs/contributing/how-we-document.md).

## Commands

```
make test    # go test ./...
make vet     # go vet ./...
make build   # produces ./bin/katalyst
make all     # vet + test + build
```

Tests should always pass on `main`. Run `make test` before sending a PR.

## Layout

```
cmd/                  cobra commands (root, init, check, fix, inspect, collection, item, schema, rules)
internal/project      project domain layer: the whole workspace, selectors, item enumeration
internal/project/config            .katalyst/ loader: schemas + storage instances (which embed their collections)
internal/storage      backend-kind registry: StorageType, Known, Granularity, Reference
internal/storage/collection            the read stack: CollectionDefinition + the thin Item
internal/storage/collection/query      query/filter predicate grammar (item list --filter, collection variants)
internal/storage/collection/document   markdown codec: Parse/Encode (frontmatter + body), with line tracking
internal/storage/collection/filesystem the filesystem backend: structural read (glob/locate) + atomic persist
internal/fix          fix transform engine: canonical form + text fixes (decides what to write; no IO)
internal/checks       check engine: per-family check types, the registry, and CheckLibrary providers
internal/checks/jsonschema  the JSON Schema library (wraps santhosh-tekuri/jsonschema); provides the object check type
internal/inspect      corpus profiling: inspectors return descriptive evidence (dual of checks)
cmd/gendocs           generates reference/check-types/ and reference/inspectors/ from the registries
docs/                 Hugo docs site — users + contributors (content in docs/content/)
product/specs/        in-flight specs only (deleted when their work lands)
```

The docs are a **separate Hugo module** so the application's `go.mod` stays
`go mod tidy`-clean. Never add the Hugo theme to the root `go.mod`; it lives
in `docs/go.mod` and is managed by `make docs-deps` (`hugo mod get`).

Katalyst **dogfoods itself on those docs.** The repo-root `.katalyst/`
directory configures a single `pages` collection over `docs/content/`, and the
CI `docs` job runs `./bin/katalyst check` after the Hugo build. A docs change
that breaks the page frontmatter contract (`schemas/page.json`, `title`
required; `weight`/`draft`/`bookCollapseSection`/`aliases` typed) fails CI, so
run `make build && ./bin/katalyst check` after editing docs. `.katalyst/` sits
at the repo root (not under `docs/content/`) so the collection's recursive
unmatched-file scan never walks the config dir itself.

Production code stays in `internal/` unless something genuinely needs to be
importable from outside the module.

The path ⇄ item-identity translation passes through
`internal/storage/collection.CollectionDefinition` (forward discovery + reverse
reconstruction), implemented per backend under `storage/collection/<backend>`
(filesystem today). Don't inline filesystem assumptions (globbing, stem-as-id,
path joins) elsewhere, a second backend (SQLite) attaches by implementing that
interface. `internal/project/config` owns the `.katalyst/` *vocabulary* (it
validates the storage `type` against a parse-time allowlist); it imports only
the `query` grammar from the collection subtree (for variant predicates), never
the readers — those depend on `config`, not the reverse. That `config → …/query`
edge is a known cross-tree compromise the config-distribution spec retires.

Per-item check *routing* (collection variants) lives in the check engine
(`engine.checksFor`), keyed on the item's parsed metadata via
`query.Predicate.Matches`, never on its path. Keep it that way: discrimination
by metadata is portable across backends and leaves the storage seam untouched.

## Testing

The project follows TDD. New behavior arrives with a failing test first.

### Style

- **External test packages.** Every `_test.go` file uses `package <pkg>_test`,
  so tests can only touch the exported API.
- **Standard library only.** No testify, no gomock, no fixtures framework.
  Just `t.Fatalf` / `t.Errorf` and small table-driven slices where useful.
- **Naming.** `TestSubject_behavior`, e.g. `TestLoad_rejectsUnknownSchemaInRule`,
  `TestValidateCmd_invalidFile_returnsExitCode1`.
- **Filesystem isolation.** Anything that touches disk scaffolds into
  `t.TempDir()`. Nothing writes into the repo at test time.
- **Helpers are per-file** and start with `t.Helper()`. Don't reach for a
  shared `testutil` package, duplication of a five-line helper is cheaper
  than a cross-package dependency.
- **CLI tests drive the real Cobra root.** Build the command with
  `cmd.NewRootCmd()`, capture output via `SetOut` / `SetErr`, and invoke
  via `SetArgs` + `Execute`. Don't shell out to a built binary.
- **Snapshot CLI text contracts.** Pin user-facing output (help, list/show,
  diagnostics) with the `cmd` snapshot harness — golden files under
  `cmd/testdata/snapshots/`, embedded and regenerated with `-update` — and keep
  exit codes, side effects, and query semantics as property tests. See
  `cmd/AGENTS.md`.

### Fixtures (`testdata/`)

Per-package `testdata/` directories hold reusable inputs. The Go tool
ignores anything under `testdata/` during build, so it's free to add.

Fixtures are loaded via `//go:embed` (see `cmd/fixtures_test.go`) rather
than `os.ReadFile`. Embeds resolve at compile time, so they survive tests
that `chdir` into a temp directory.

**Use a fixture when**: the content is reused across tests, or a single
test needs to scaffold a realistic multi-file repo.

**Keep it inline when**: the input is one-off, byte-exact (BOM, CRLF,
malformed YAML, exact line numbers), or a tiny one-liner. In those cases
the literal bytes *are* the assertion and a file would only add indirection.
See `internal/storage/collection/document/document_test.go` for the canonical example.

**Honest duplication is fine** when two layers genuinely test different
contracts. The book schema fixtures in `cmd/testdata/schemas/book.json`
and `internal/validator/testdata/schemas/book.json` differ on purpose: one
exercises CLI plumbing, the other exercises validator rule coverage.

Per-`testdata/` READMEs list what each fixture is for. Update them when
you add a fixture.

## Adding code

- Run `gofmt -w .` (or `make fmt`) before committing.
- Don't add comments that just narrate what the code does. Comments
  explain *why* (non-obvious intent, trade-offs, constraints) not *what*.
- The `markdown_writing_tells` check surfaces likely AI-writing tells (em
  dashes, decorative emoji, stock phrases) as warnings for review; see
  `docs/content/contributing/how-we-document.md`.
- Production code that needs a new test fixture: add it under the
  consuming package's `testdata/`, embed it in `fixtures_test.go`, and
  note it in that package's `testdata/README.md`.
- Each check type lives in its own file in a per-family package under
  `internal/checks/` (`structuredobject`, `markdownbodytext`, `filesystem`,
  `plaintext`), holding its struct, `Run`, `Descriptor`, and an `init()` that
  registers it through the package's `register` helper (in `library.go`), which
  stamps `Descriptor.Library`. The core `checks` package owns the shared types
  and registry and imports none of the families; callers blank-import
  `internal/checks/all` to wire them all in. To add a check type, add one file
  (and a `config.CheckType` constant + `normalizeCheck` case, which `checks`
  can't own because it imports `config`).
- The check registry (populated by those `Register` calls) is the single source
  of truth for check types: `cmd/engine` builds the runnable list by registry
  lookup (`Build`/`BuildCollection`), and both `cmd/gendocs` and `katalyst
  check-types list` read `Descriptors()`/`Families()`. `registry_test.go` fails
  if a dispatched check type has no descriptor, a new check type ships with its
  descriptor. The `json:` tags on `Descriptor`/`Field` are the published wire
  contract for `katalyst check-types list --json`; keep them stable.
- A check type's **family** groups it by source-data kind, and is orthogonal to
  its granularity: a collection-scoped check is filed by the data it reads
  (`unique_field` → `structuredObject`, `unique_filename` → `fileSystem`). The
  `kind` id is the wire contract and never changes, even when the family does.
- A **CheckLibrary** (`internal/checks`, `CheckLibrary`/`SchemaLibrary`) is the
  *provider* behind a check type. Every check type has one: the four native
  families each register a library in their `library.go` (with `Name()`,
  `Available()` returning nil, and the `register` helper that stamps
  `Descriptor.Library`); a **schema-backed** library
  (`internal/checks/jsonschema`, the only one today) also implements
  `SchemaLibrary` to compile a named schema and probes `Available()` (an
  out-of-process tool checks its binary). `registry_test.go` enforces that every
  check type names a registered library. **Library is provenance, orthogonal to
  family** (source-data kind): `object` (json-schema) and `object_required_field`
  (structuredobject) are both the `structuredObject` family. The engine resolves
  a kind's library via `checks.LibraryFor` and fails the run on `Available()`
  error. The `object` schema-selection precedence lives in `jsonschema.Resolve`,
  not the engine; schemas stay flat under `.katalyst/schemas/`, resolved to a
  library by the binding's `kind`.
- Filesystem name/path check types share a **target × rule** shape: a `target`
  (`filename`, `filename-ext`, `parent-dir`, `path-segments`) resolved by
  `resolveTarget` in `internal/checks/filesystem/common.go`, against which a rule runs.
  Targets that span directories (`path-segments`, `path_depth`, `path_charset`)
  resolve relative to `Context.CollectionRoot`, populated by the per-item check
  pass, don't assume `FilePath` alone is enough.
- **Text check types** (`text_requires`/`text_forbids`/`text_denylist`) lint the
  body as raw text over a **span selector** (`target`:
  `body`/`line`/`first-line`/`matched-lines`), sharing `textSpans` in
  `internal/checks/plaintext/common.go`. Their regex is compiled **unanchored**, the
  deliberate divergence from `filesystem_name_regex`'s `^…$`. `text_forbids` may
  carry an opt-in `fix` template, applied to the body by `cmd/fix.go`, which then
  re-checks its own work; this is the one place `fix` rewrites the body rather
  than only reformatting frontmatter. Because text rules read only the body,
  `check` runs every configured check on frontmatter-less items too (no
  "no frontmatter" rejection).
- **Collection-scoped check types** implement `checks.CollectionCheck`
  (`RunCollection(CollectionContext)`), not `Check`. They register a
  `CollectionBuilder` (not a per-item builder); `engine.collectionChecksFor`
  builds them via `checks.BuildCollection` and a second pass in `cmd/check.go`
  re-scans the *whole* collection via `project.Items`, independent of the
  selector, a uniqueness verdict is only correct against every item. Mark such
  types in `config.collectionScopedTypes` and set `Scope: "collection"` on their
  descriptor.
