# katabridge

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
go install github.com/katabase-ai/katabridge@latest
```

Or from source:

```
git clone https://github.com/katabase-ai/katabridge
cd katabridge
make build  # produces ./bin/katabridge
```

## Quickstart

```bash
mkdir my-notes && cd my-notes
katabridge init                  # scaffolds katabridge.yaml, schemas/, notes/
katabridge validate notes/example.md
```

Both files are picked up automatically: `validate` discovers the nearest
`katabridge.yaml` walking up from the working directory, then matches the
file against the config's glob rules.

## Configuring

A `katabridge.yaml` at your repo root maps schemas to globs:

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

### `katabridge validate [paths...]`

Validate each file's frontmatter against its resolved schema.

```
$ katabridge validate notes/dune.md
notes/dune.md: OK

$ katabridge validate notes/bad.md
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

### `katabridge fmt [paths...]`

Normalize frontmatter: top-level keys sorted alphabetically, default
block style, exactly one trailing newline. Body preserved verbatim.

```
katabridge fmt notes/**/*.md                  # rewrites in place
katabridge fmt --check notes/**/*.md          # CI mode: no writes, exit 1 if any change
```

`fmt` has no flags besides `--check` on purpose — see
[`product/decisions.md`](product/decisions.md) D4.

### Filesystem-style operations

The CLI also supports shell-like operations for local workflows:

```bash
katabridge mkdir -p notes/archive
katabridge cp notes/a.md notes/archive/a.md
katabridge mv notes/archive/a.md notes/archive/a-renamed.md
katabridge set notes/archive/a-renamed.md title="New title" year=2026
katabridge rm notes/archive/a-renamed.md
```

Implemented commands:

- `katabridge cp <src> <dst>`
- `katabridge mkdir <dir> [dir...]`
- `katabridge mv <src> <dst>`
- `katabridge rm <path> [path...]`
- `katabridge set <path> key=value [key=value...]`

Validation behavior for write-affecting commands:

- Default is strict validation before write.
- `cp` validates markdown destination files (`*.md`) before writing.
- `set` validates the resulting markdown document before writing.
- `--no-validate` bypasses this check.
- `--schema` overrides config-based schema resolution (same precedence rules as `validate`).

### `katabridge schema list` / `katabridge schema show <name>`

```
$ katabridge schema list
book    schemas/book.json
person  schemas/person.json

$ katabridge schema show book
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  ...
}
```

### `katabridge init [--dir <path>]`

Scaffold a minimal working repo: `katabridge.yaml`, one schema, one
example document. Refuses to overwrite anything that already exists.

### Shell completion

Cobra provides scripts for bash, zsh, fish, and powershell:

```bash
source <(katabridge completion zsh)
katabridge completion zsh > "${fpath[1]}/_katabridge"   # persistent
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
cmd/                  cobra commands (root, init, validate, schema, fmt, cp/mkdir/mv/rm/set)
internal/config       katabridge.yaml loader + glob-based schema resolution
internal/frontmatter  YAML frontmatter parser + formatter, with line tracking
internal/validator    JSON Schema validation (wraps santhosh-tekuri/jsonschema)
product/              roadmap, resolved decisions, open questions
```

See [`AGENTS.md`](AGENTS.md) for conventions on tests, fixtures, and code style.
