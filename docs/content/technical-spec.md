+++
title = "Technical Spec (Draft)"
+++

## Scope

The core of Katalyst is a framework for content validation. Where practical, it
should build on existing systems rather than replacing them (for example JSON
Schema and other established validators).

Unlike most validation frameworks, Katalyst is intended to:

- Span both unstructured and structured content.
- Support deterministic and non-deterministic rule evaluation.

## Why this matters

Katalyst is designed as infrastructure for AI harnesses and agentic systems.
Representative use cases include:

- Guardrails for agents creating or updating content, including memory stores
  and shared knowledge bases.
- Validation across mixed content types, rather than only typed records.
- Support for storage-layer migrations (for example markdown to SQLite or other
  backends) without losing validation guarantees.

## Multiple form factors

The same DSL is intended to be exposed through several form factors:

1. A linter that applies Katalyst rules to files in a filesystem.
2. A CLI that enforces rules on write operations in a filesystem (markdown,
   YAML, CSV, and related formats).
3. A server that enforces rules on write operations for SQL and NoSQL stores
   (for example SQLite, PostgreSQL, MongoDB).

These form factors share one core idea: schemas and linters are closely related
and should compose across storage boundaries.

## Roadmap directions

Katalyst is expected to expand in three directions:

1. A richer and more expressive validation language.
2. Application to more storage backends.
3. Tooling that uses validation rules to support additional operations,
   especially content and storage migrations.

## DSL direction

The DSL is expected to support validation for:

- Object-like data (YAML, JSON, SQL-backed records, and similar structures).
- Markdown content.
- File and directory structures.

## Current implementation status

The current implementation in this repo is intentionally narrower than the full
spec direction:

- Filesystem-first CLI.
- Markdown frontmatter validation via JSON Schema.
- Config-driven schema resolution through `katalyst.yaml`.

Treat this page as directional scope, not a final frozen specification.
