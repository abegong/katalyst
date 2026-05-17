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
| `book.json`       | `validate_test.go`, `schema_test.go`                     | Minimal `{title, year}` schema; carries JSON-Schema `title: book`. |
| `person.json`     | `schema_test.go`                                         | Minimal object schema; carries JSON-Schema `title: person`.       |
| `strict-book.json`| `validate_config_test.go`                                | Adds required `isbn` with a regex `pattern`. Used to test that an inline `schema:` key in a doc's frontmatter overrides config rules. |

These are deliberately simpler than `internal/validator/testdata/schemas/book.json`,
which exists to exercise the validator itself.

### `configs/`

| File                       | Used by                          | Purpose                                                            |
|----------------------------|----------------------------------|--------------------------------------------------------------------|
| `book-and-person.yaml`     | `schema_test.go`                 | Two-schema config with one `books/**` rule. Drives `schema list`/`schema show` tests. |
| `strict-book.yaml`         | `validate_config_test.go`        | Two-schema config (`book` + `strict-book`) with a single rule pointing all `notes/**/*.md` at `book`, so an inline `schema: strict-book` key has something to override. |

## Adding a fixture

1. Drop the file under `schemas/` or `configs/` (add a new subfolder if a
   new kind of fixture earns one).
2. Embed it in `../fixtures_test.go` with `//go:embed`.
3. Add a row to the relevant table above.
