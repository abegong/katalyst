# katabridge

Define and enforce schemas for structured metadata (frontmatter) on markdown
files. Inspired by [JSON Schema][js] and the [MongoDB validation API][mv].

[js]: https://json-schema.org/
[mv]: https://www.mongodb.com/docs/manual/core/schema-validation/

> **Status:** early v0.1 scaffolding. Only `validate` is wired up, and it
> requires an explicit `--schema` flag. See [`product/roadmap.md`](product/roadmap.md)
> for what's planned and [`product/decisions-to-make.md`](product/decisions-to-make.md)
> for open design questions.

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

Given a schema `schemas/book.json`:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["title", "year"],
  "properties": {
    "title": { "type": "string", "minLength": 1 },
    "year":  { "type": "integer", "minimum": 0 },
    "tags":  { "type": "array", "items": { "type": "string" } }
  }
}
```

And a markdown file `notes/dune.md`:

```markdown
---
title: Dune
year: 1965
tags: [sci-fi, classic]
---

# Dune

A story about spice.
```

Run:

```
katabridge validate --schema schemas/book.json notes/dune.md
```

Exit codes:

| Code | Meaning                              |
|-----:|--------------------------------------|
| `0`  | All files valid                      |
| `1`  | One or more validation failures      |
| `2`  | Usage error or unreadable input      |

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
cmd/                  cobra commands (root, init, validate, schema)
internal/frontmatter  YAML frontmatter extraction from markdown
internal/validator    JSON Schema validation (wraps santhosh-tekuri/jsonschema)
product/              roadmap and open design questions
```
