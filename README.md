# katalyst

Define and enforce schemas for structured metadata (frontmatter) on markdown
files. Inspired by [JSON Schema][js] and the [MongoDB validation API][mv].

[js]: https://json-schema.org/
[mv]: https://www.mongodb.com/docs/manual/core/schema-validation/

> **Status:** v0.2. `init`, `validate`, `schema list/show`, and `fmt` are
> implemented. See [`product/roadmap.md`](product/roadmap.md) for what's
> next and [`product/decisions.md`](product/decisions.md) for what's
> already locked in.

## Install

```
go install github.com/katabase-ai/katalyst@latest
```

Or from source:

```
git clone https://github.com/katabase-ai/katalyst
cd katalyst
make build  # produces ./bin/katalyst
```

## Quickstart

```bash
mkdir my-notes && cd my-notes
katalyst init                  # scaffolds katalyst.yaml, schemas/, notes/
katalyst validate notes/example.md
```

Both files are picked up automatically: `validate` discovers the nearest
`katalyst.yaml` walking up from the working directory, then matches the
file against the config's glob rules.

## Configuring

A `katalyst.yaml` at your repo root maps schemas to globs:

```yaml
schemas:
  book:   ./schemas/book.json
  person: ./schemas/person.json

rules:
  - paths: "notes/books/**/*.md"
    schema: book
  - paths: "notes/people/**/*.md"
    schema: person
```

Schema resolution precedence, highest first:

1. `--schema <path>` on the command line (overrides everything).
2. An inline `schema: <name>` key inside a file's frontmatter (the name
   refers to an entry in `schemas:` above; the directive itself is
   stripped before validation, so you can have strict
   `additionalProperties: false` schemas without listing it).
3. The first matching entry in `rules`.

Files that don't resolve to any schema are reported as errors.

## Commands

### `katalyst validate [paths...]`

Validate each file's frontmatter against its resolved schema.

```
$ katalyst validate notes/dune.md
notes/dune.md: OK

$ katalyst validate notes/bad.md
notes/bad.md:3: /year: got string, want integer
notes/bad.md: /: missing property 'isbn'
```

Errors include `:line` when the source position is known. Missing-required
errors fall back to the nearest known ancestor line.

Exit codes:

| Code | Meaning                              |
|-----:|--------------------------------------|
| `0`  | All files valid                      |
| `1`  | One or more validation failures      |
| `2`  | Usage error or unreadable input      |

### `katalyst fmt [paths...]`

Normalize frontmatter: top-level keys sorted alphabetically, default
block style, exactly one trailing newline. Body preserved verbatim.

```
katalyst fmt notes/**/*.md                  # rewrites in place
katalyst fmt --check notes/**/*.md          # CI mode: no writes, exit 1 if any change
```

`fmt` has no flags besides `--check` on purpose — see
[`product/decisions.md`](product/decisions.md) D4.

### CRUD operations

The CLI supports basic item-level CRUD operations:

```bash
katalyst create notes/a.md title="New title" year=2026
katalyst read notes/a.md
katalyst update notes/a.md title="Updated title"
katalyst delete notes/a.md
```

Implemented commands:

- `katalyst create <path> [key=value ...]`
- `katalyst read <path>`
- `katalyst update <path> key=value [key=value...]`
- `katalyst delete <path> [path...]`

Validation behavior for write-affecting commands (`create`, `update`):

- Default is strict validation before write.
- `create` validates markdown destination files (`*.md`) before writing.
- `update` validates the resulting markdown document before writing.
- `--no-validate` bypasses this check.
- `--schema` overrides config-based schema resolution (same precedence rules as `validate`).

### `katalyst schema list` / `katalyst schema show <name>`

```
$ katalyst schema list
book    schemas/book.json
person  schemas/person.json

$ katalyst schema show book
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  ...
}
```

### `katalyst init [--dir <path>]`

Scaffold a minimal working repo: `katalyst.yaml`, one schema, one
example document. Refuses to overwrite anything that already exists.

### Shell completion

Cobra provides scripts for bash, zsh, fish, and powershell:

```bash
source <(katalyst completion zsh)
katalyst completion zsh > "${fpath[1]}/_katalyst"   # persistent
```

## Documentation site (Hugo)

User-facing docs live under `docs/` and are served/built by Hugo.

If `hugo` is on your `PATH`, `make` will use it directly. Otherwise the docs
targets automatically fall back to:

```bash
go run github.com/gohugoio/hugo@latest
```

Optional local install:

```bash
go install github.com/gohugoio/hugo@latest
```

Run docs locally:

```bash
make docs-serve
```

Build static docs:

```bash
make docs-build
```

## Development

This project follows TDD. Tests live next to the code they exercise, in
`*_test.go` files using only the standard library `testing` package.

```
make test    # go test ./...
make vet     # go vet ./...
make all     # vet, test, build
```

Layout:

```
cmd/                  cobra commands (root, init, validate, schema, fmt, create/read/update/delete)
internal/config       katalyst.yaml loader + glob-based schema resolution
internal/frontmatter  YAML frontmatter parser + formatter, with line tracking
internal/validator    JSON Schema validation (wraps santhosh-tekuri/jsonschema)
product/              roadmap, resolved decisions, open questions
```

See [`AGENTS.md`](AGENTS.md) for conventions on tests, fixtures, and code style.
