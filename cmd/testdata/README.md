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

### `configs/`

All configs use the v0 named `collections:` map (see
`product/cli-spec.md`).

| File                       | Used by                              | Purpose                                                            |
|----------------------------|--------------------------------------|--------------------------------------------------------------------|
| `book-and-person.yaml`     | `schema_test.go`, `collection_test.go` | Two-schema config with `books` and `people` collections. Drives `schema`/`collection` tests. |
| `strict-book.yaml`         | `check_test.go`                      | Two-schema config (`book` + `strict-book`) with a single `notes` collection bound to `book`, so an inline `schema: strict-book` key has something to override. |
| `object-check.yaml`        | (reference)                          | `notes` collection using `checks:` with `kind: object`. |
| `markdown-check.yaml`      | `check_test.go`                      | `notes` collection using `checks:` with `kind: markdown_title_matches_h1`. |
| `filesystem-check.yaml`    | (reference)                          | `notes` collection using `checks:` with `kind: filesystem_filename_matches_slug`. |

## Adding a fixture

1. Drop the file under `schemas/` or `configs/` (add a new subfolder if a
   new kind of fixture earns one).
2. Embed it in `../fixtures_test.go` with `//go:embed`.
3. Add a row to the relevant table above.
