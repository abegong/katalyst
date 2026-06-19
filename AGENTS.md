# AGENTS.md

Conventions for anyone — human or AI — making changes in this repo.

For *what* the project does and how to use the CLI, see [`README.md`](README.md).
For *why* the design is the way it is, see [`product/decisions.md`](product/decisions.md).
For *how we plan and document* changes, see [`product/how-we-plan.md`](product/how-we-plan.md)
and [`product/how-we-document.md`](product/how-we-document.md).

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
cmd/                  cobra commands (root, init, validate, schema, fmt)
internal/config       katalyst.yaml loader + glob-based schema resolution
internal/frontmatter  YAML frontmatter parser + formatter, with line tracking
internal/validator    JSON Schema validation (wraps santhosh-tekuri/jsonschema)
product/              roadmap, resolved decisions, open questions
```

Production code stays in `internal/` unless something genuinely needs to be
importable from outside the module.

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
  shared `testutil` package — duplication of a five-line helper is cheaper
  than a cross-package dependency.
- **CLI tests drive the real Cobra root.** Build the command with
  `cmd.NewRootCmd()`, capture output via `SetOut` / `SetErr`, and invoke
  via `SetArgs` + `Execute`. Don't shell out to a built binary.

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
See `internal/frontmatter/frontmatter_test.go` for the canonical example.

**Honest duplication is fine** when two layers genuinely test different
contracts. The book schema fixtures in `cmd/testdata/schemas/book.json`
and `internal/validator/testdata/schemas/book.json` differ on purpose: one
exercises CLI plumbing, the other exercises validator rule coverage.

Per-`testdata/` READMEs list what each fixture is for. Update them when
you add a fixture.

## Adding code

- Run `gofmt -w .` (or `make fmt`) before committing.
- Don't add comments that just narrate what the code does. Comments
  explain *why* — non-obvious intent, trade-offs, constraints — not *what*.
- Production code that needs a new test fixture: add it under the
  consuming package's `testdata/`, embed it in `fixtures_test.go`, and
  note it in that package's `testdata/README.md`.
