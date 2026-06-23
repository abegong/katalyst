# internal/checks/jsonschema/testdata

Fixtures for the JSON Schema check library. Embedded via `//go:embed` in
[`../jsonschema_test.go`](../jsonschema_test.go).

See [`AGENTS.md`](../../../../AGENTS.md) for the project-wide policy on when
to use a fixture vs an inline literal.

## Fixtures

### `schemas/`

| File        | Used by              | Purpose                                                          |
|-------------|----------------------|------------------------------------------------------------------|
| `book.json` | `jsonschema_test.go` | Rich schema exercising `required`, `minLength`/`minimum`, array `items`, and `additionalProperties: false`. Drives every `TestCheck_*` case. |

This is intentionally richer than `cmd/testdata/schemas/book.json`. The
CLI tests only need a schema that accepts/rejects a frontmatter blob;
these tests need every constraint the library can apply.

Tiny malformed snippets (e.g. `{ "type": 123 }` in `TestCompile_invalidSchema`)
stay inline, the exact bytes are the assertion.

## Adding a fixture

1. Drop the file under `schemas/`.
2. Embed it in `../jsonschema_test.go` with `//go:embed`.
3. Add a row to the table above.
