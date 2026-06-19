+++
title = "Roadmap"
weight = 40
+++

# Roadmap

A living document of where `katalyst` is heading. Items near the top are
soonest; items further down are more speculative.

## v0.1 — MVP: validate YAML frontmatter against JSON Schema ✅

Shipped.

- Parse YAML frontmatter from markdown files (`---` fences).
- Validate parsed frontmatter against a JSON Schema file.
- `katalyst init` — scaffold a config file, schema, and example doc.
- `katalyst check [selector ...]` — config-driven schema discovery with a
  `--schema` override; an inline `schema:` frontmatter key takes precedence
  over the collection's schema.
- `katalyst schema list` — list registered schemas.
- `.katalyst/` project config with named schemas and collections.
- Stdlib `testing`; CI runs `go vet`, race-enabled `go test`, and `go build`.

## v0.2 — Authoring ergonomics ✅

Shipped.

- `katalyst schema show <name>` — pretty-print a registered schema.
- File:line pointers in `check` error output, walking up to ancestor paths
  when the leaf has no source location.
- `katalyst fix` — normalize frontmatter (sort top-level keys, trailing
  newline, block style); supports `--check` for CI.
- Named collections, the `<collection>/<item>` selector grammar, and the
  `collection` and `item` subcommands.
- An 18-check engine across object, markdown, and filesystem families, with
  a generated rule reference.

Safe value-injecting mutation was deliberately left out of `fix` — see the
[formatting rationale]({{< relref "../explanation/formatting.md" >}}).

## v0.3 — Safer mutation

- `katalyst patch <file> --set key=value` (working name) — targeted,
  user-driven mutation rather than an opaque value-injecting fix.
- `katalyst schema check` — sanity-check schema files themselves
  (valid JSON, no dangling `$ref`s, meta-schema conformance).
- `--allow-unmatched` flag and corresponding config knob.
- `katalyst explain <selector>` — single-item deep-dive (which schema
  matched, why, the check result, frontmatter dump).

## v0.4 — Mongo-inspired schema interactions

The MongoDB validation API isn't just "is this document valid?" — it's a
toolkit for *evolving* and *querying* schemas. Borrow these ideas:

- `katalyst schema diff <a> <b>` — structural diff between two schemas
  (added/removed fields, tightened constraints, etc.).
- `katalyst schema infer <selector ...>` — synthesize a draft schema from
  existing documents (à la `mongo`'s sampling validators).
- `katalyst query '<jsonpath or mongo-style filter>'` — find documents
  matching a structural query across the repo.
- `katalyst schema migrate` — apply a transformation across documents
  when a schema changes (rename field, change type, default value).

## v0.5 — More frontmatter formats

- TOML frontmatter (`+++` fences).
- JSON frontmatter (`{ ... }` fenced or whole-file).
- Auto-detect format per file; let the schema/config opt into a subset.

## v0.6+ — Bridges (the "bridge" in katalyst)

- A **connector** layer mapping non-filesystem backends (SQLite, CSV
  directories, S3, hosted APIs) onto collections and items. SQLite is the
  intended first stress test. See
  [Connectors]({{< relref "../explanation/connectors.md" >}}).
- Export validated frontmatter as a queryable index (SQLite, DuckDB, JSON).
- Watch mode for editor integration.
- LSP server so editors can show schema errors inline.
- Relations between notes (foreign-key-style refs between documents).

## Housekeeping / infra

- Separate the docs Hugo module from the Go module so `go mod tidy` stays
  clean — spec + plan in
  [`specs/docs-module-separation.md`](specs/docs-module-separation.md)
  (status: planning).

## Non-goals (for now)

- Being a general-purpose markdown linter (use `markdownlint` etc.).
- Rendering or transforming markdown body content.
- Replacing JSON Schema — we want to interoperate, not reinvent.
