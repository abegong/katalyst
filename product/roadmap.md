# Roadmap

A living document of where `katabridge` is heading. Items near the top are
soonest; items further down are more speculative.

## v0.1 — MVP: validate YAML frontmatter against JSON Schema ✅

Shipped.

- Parse YAML frontmatter from markdown files (`---` fences).
- Validate parsed frontmatter against a JSON Schema file.
- `katabridge init` — scaffold a config file, schema, and example doc.
- `katabridge validate [paths...]` — config-driven schema discovery
  with `--schema` override; inline `schema:` frontmatter key takes
  precedence over config rules.
- `katabridge schema list` — list registered schemas.
- `katabridge.yaml` config maps glob patterns (via doublestar) to
  named schemas.
- Stdlib `testing`; CI runs `go vet`, race-enabled `go test`, and `go build`.

## v0.2 — Authoring ergonomics ✅

Shipped.

- `katabridge schema show <name>` — pretty-print a registered schema.
- File:line pointers in `validate` error output, walking up to ancestor
  paths when the leaf has no source location.
- `katabridge fmt` — normalize frontmatter (sort top-level keys,
  trailing newline, block style); supports `--check` for CI.

`validate --fix` was moved to v0.3 — see `product/decisions.md` (D3).

## v0.3 — Safer mutation

- `katabridge patch <file> --set key=value` (working name) — targeted,
  user-driven mutation rather than an opaque `--fix`.
- `katabridge schema check` — sanity-check schema files themselves
  (valid JSON, no dangling `$ref`s, meta-schema conformance).
- `--allow-unmatched` flag and corresponding config knob.
- `katabridge ls` — list files with the schema each matched against;
  great for debugging association rules.
- `katabridge explain <path>` — single-file deep-dive (which schema
  matched, why, validation result, frontmatter dump).

## v0.4 — Mongo-inspired schema interactions

The MongoDB validation API isn't just "is this document valid?" — it's a
toolkit for *evolving* and *querying* schemas. Borrow these ideas:

- `katabridge schema diff <a> <b>` — structural diff between two schemas
  (added/removed fields, tightened constraints, etc.).
- `katabridge schema infer <paths...>` — synthesize a draft schema from
  existing documents (à la `mongo`'s sampling validators).
- `katabridge query '<jsonpath or mongo-style filter>'` — find documents
  matching a structural query across the repo.
- `katabridge schema migrate` — apply a transformation across documents
  when a schema changes (rename field, change type, default value).

## v0.5 — More frontmatter formats

- TOML frontmatter (`+++` fences).
- JSON frontmatter (`{ ... }` fenced or whole-file).
- Auto-detect format per file; let the schema/config opt into a subset.

## v0.6+ — Bridges (the "bridge" in katabridge)

- Export validated frontmatter as a queryable index (SQLite, DuckDB, JSON).
- Watch mode for editor integration.
- LSP server so editors can show schema errors inline.
- Relations between notes (foreign-key-style refs between documents).

## Non-goals (for now)

- Being a general-purpose markdown linter (use `markdownlint` etc.).
- Rendering or transforming markdown body content.
- Replacing JSON Schema — we want to interoperate, not reinvent.
