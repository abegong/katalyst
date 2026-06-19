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

Project layout (the `.katalyst/` directory, schema files, and collection
files) is scaffolded inline by the `writeProject` helper in
[`../helpers_test.go`](../helpers_test.go); collection bodies are small
enough to live as literals in each test rather than as fixtures. Because
the shared schema fixtures are JSON, test projects set `schemas: { format:
json }` in their `.katalyst/config.yaml` (the `schemaFormatJSON` helper).

## Adding a fixture

1. Drop the file under `schemas/` (add a new subfolder if a new kind of
   fixture earns one).
2. Embed it in `../fixtures_test.go` with `//go:embed`.
3. Add a row to the relevant table above.
