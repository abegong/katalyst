# cmd/testdata

Reusable fixtures for `cmd/` CLI tests. Loaded via `//go:embed` in
[`../fixtures_test.go`](../fixtures_test.go) so tests can reference them
even after `chdir`-ing into a temp directory.

See [`AGENTS.md`](../../AGENTS.md) for the project-wide policy on when
to use a fixture vs an inline literal.

## Fixtures

### `schemas/`

| File              | Used by                                                  | Purpose                                                          |
|-------------------|----------------------------------------------------------|------------------------------------------------------------------|
| `book.json`       | `check_test.go`, `item_test.go`, `schema_test.go`        | Minimal `{title, year}` schema; carries JSON-Schema `title: book`. |
| `person.json`     | `schema_test.go`, `collection_test.go`                   | Minimal object schema; carries JSON-Schema `title: person`.       |
| `strict-book.json`| `check_test.go`, `item_test.go`                          | Adds required `isbn` with a regex `pattern`. Used to test that an inline `schema:` key in a doc's frontmatter overrides the collection's configured schema. |

These are deliberately simpler than `internal/validator/testdata/schemas/book.json`,
which exists to exercise the validator itself.

### `help/`

Top-level CLI help snapshots used by `root_test.go` and
`help_snapshot_test.go`.

| File | Used by | Purpose |
|---|---|---|
| `root-noargs.txt` | `root_test.go` | Exact output of `katalyst` with no args |
| `inspect-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst inspect --help` |
| `init-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst init --help` |
| `check-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst check --help` |
| `fix-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst fix --help` |
| `collection-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst collection --help` |
| `item-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst item --help` |
| `schema-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst schema --help` |
| `check-types-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst check-types --help` |
| `inspectors-help.txt` | `help_snapshot_test.go` | Exact output of `katalyst inspectors --help` |

Project layout (the `.katalyst/` directory, schema files, and collection
files) is scaffolded inline by the `writeProject` helper in
[`../helpers_test.go`](../helpers_test.go); collection bodies are small
enough to live as literals in each test rather than as fixtures. Because
the shared schema fixtures are JSON, test projects set `schemas: { format:
json }` in their `.katalyst/config.yaml` (the `schemaFormatJSON` helper).

## Adding a fixture

1. Drop the file under the relevant subfolder (`schemas/`, `help/`, or a
   new subfolder when needed).
2. Embed it in `../fixtures_test.go` with `//go:embed`.
3. Add a row to the relevant table above.
