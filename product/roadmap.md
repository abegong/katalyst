# Roadmap

A living document of where `katabridge` is heading. Items near the top are
soonest; items further down are more speculative.

## v0.1 — MVP: validate YAML frontmatter against JSON Schema

- Parse YAML frontmatter from markdown files (`---` fences).
- Validate parsed frontmatter against a JSON Schema file.
- CLI commands:
  - `katabridge init` — scaffold a config file and an example schema.
  - `katabridge validate [paths...]` — validate files, non-zero exit on failure.
  - `katabridge schema list` — list schemas registered in the config.
- Config file (`katabridge.yaml`) maps glob patterns to schema files.
- Stdlib `testing` for tests; CI runs `go test ./...` and `go vet`.

## v0.2 — Authoring ergonomics

- `katabridge schema show <name>` — pretty-print a registered schema.
- `katabridge validate --fix` (where safe) — e.g. add missing required keys
  with sentinel values, normalize key order.
- Better error formatting (file:line pointers into the frontmatter).
- `katabridge fmt` — normalize frontmatter (key order, quoting, trailing
  newline).

## v0.3 — More frontmatter formats

- TOML frontmatter (`+++` fences).
- JSON frontmatter (`{ ... }` fenced or whole-file).
- Auto-detect format per file; let the schema/config opt into a subset.

## v0.4 — Mongo-inspired schema interactions

The MongoDB validation API isn't just "is this document valid?" — it's a
toolkit for *evolving* and *querying* schemas. Borrow these ideas:

- `katabridge schema diff <a> <b>` — structural diff between two schemas
  (added/removed fields, tightened constraints, etc.).
- `katabridge schema infer <paths...>` — synthesize a draft schema from
  existing documents (à la `mongo`'s sampling validators).
- `katabridge query '<jsonpath or mongo-style filter>'` — find documents
  matching a structural query across the repo.
- `katabridge schema migrate` — apply a transformation across documents when
  a schema changes (rename field, change type, default value).
- `katabridge schema check --strict` — additionalProperties: false enforcement
  with helpful "did you mean?" suggestions.

## v0.5+ — Bridges (the "bridge" in katabridge)

- Export validated frontmatter as a queryable index (SQLite, DuckDB, JSON).
- Watch mode for editor integration.
- LSP server so editors can show schema errors inline.
- Relations between notes (foreign-key-style refs between documents).

## Non-goals (for now)

- Being a general-purpose markdown linter (use `markdownlint` etc.).
- Rendering or transforming markdown body content.
- Replacing JSON Schema — we want to interoperate, not reinvent.
