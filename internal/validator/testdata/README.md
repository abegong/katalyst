# internal/validator/testdata

Fixtures for the JSON Schema validator. Embedded via `//go:embed` in
[`../validator_test.go`](../validator_test.go).

See [`AGENTS.md`](../../../AGENTS.md) for the project-wide policy on when
to use a fixture vs an inline literal.

## Fixtures

### `schemas/`

| File        | Used by             | Purpose                                                          |
|-------------|---------------------|------------------------------------------------------------------|
| `book.json` | `validator_test.go` | Rich schema exercising `required`, `minLength`/`minimum`, array `items`, and `additionalProperties: false`. Drives every `TestValidate_*` case. |

This is intentionally richer than `cmd/testdata/schemas/book.json`. The
CLI tests only need a schema that accepts/rejects a frontmatter blob;
these tests need every constraint the validator can apply.

Tiny malformed snippets (e.g. `{ "type": 123 }` in `TestLoad_invalidSchema`)
stay inline — the exact bytes are the assertion.

## Adding a fixture

1. Drop the file under `schemas/`.
2. Embed it in `../validator_test.go` with `//go:embed`.
3. Add a row to the table above.
