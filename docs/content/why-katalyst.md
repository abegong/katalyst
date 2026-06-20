+++
title = "Why Katalyst?"
weight = 10
+++

# Why Katalyst?

Traditional data management often forces teams into binary choices:
structured or unstructured, rigid or chaotic. Katalyst is an experimental
framework aimed at enabling fast, low-risk evolution through progressive
typing in the storage layer.

## Database management is risky and rigid

Backend architecture and database changes are frequently high-risk operations.
Teams therefore wrap data systems in heavy governance for access control,
schema changes, and migrations. Those controls are necessary, but the cost can
make change rare. Rare change creates rigidity.

## AI-native apps want flexibility

AI-driven systems increasingly involve non-engineers designing prompts and
workflows that generate and transform data. To move quickly, teams often store
semi-structured or unstructured data.

Over time, familiar pain appears:

1. More frequent bugs.
2. Slower debugging and incident resolution.
3. Confusion around ownership and system boundaries.
4. Hard-to-read workflows and data paths.
5. Risky refactors and fragile changes.
6. Fewer bugs caught before production.
7. Growing dependence on manual QA.

## Move past the structured/unstructured dichotomy

When schema changes are rare and expensive, teams are pushed toward two bad
extremes: rigid systems with high coordination costs, or flexible systems with
weak guarantees.

Katalyst explores a third path: make schema changes fast and safe enough that
teams can progressively add structure as it becomes necessary.

This follows a proven pattern from the application layer (TypeScript, mypy,
Pydantic) and extends it deeper into storage systems.

## What Katalyst is

Katalyst is an experimental framework for progressive typing in the storage
layer, designed for AI-readiness:

- Validate content across structured and unstructured forms.
- Apply one validation model across filesystems and databases.
- Use validation rules not only for checks, but as primitives for migration
  and system evolution.

The core is a framework for **content validation**. Where practical, it builds
on existing systems rather than replacing them (for example JSON Schema and
other established validators). Unlike most validation frameworks, Katalyst is
intended to:

- Span both unstructured and structured content.
- Support deterministic and non-deterministic rule evaluation.

## Why this matters

Katalyst is designed as infrastructure for AI harnesses and agentic systems.
Representative use cases include:

- Guardrails for agents creating or updating content, including memory stores
  and shared knowledge bases.
- Validation across mixed content types, rather than only typed records.
- Support for storage-layer migrations (for example markdown to SQLite or
  other backends) without losing validation guarantees.

## Multiple form factors

The same DSL is intended to be exposed through several form factors:

1. A linter that applies Katalyst rules to files in a filesystem.
2. A CLI that enforces rules on write operations in a filesystem (markdown,
   YAML, CSV, and related formats).
3. A server that enforces rules on write operations for SQL and NoSQL stores
   (for example SQLite, PostgreSQL, MongoDB).

These form factors share one core idea: schemas and linters are closely
related and should compose across storage boundaries. The conceptual basis —
why each backend tier unlocks new operations — is in
[Progressive operations]({{< relref "deep-dives/progressive-operations.md" >}}) and the
[core concepts]({{< relref "core-concepts.md" >}}).

## Current implementation status

The current implementation in this repo is intentionally narrower than the
full direction above:

- Filesystem-first CLI.
- Markdown frontmatter validation via JSON Schema.
- Config-driven schema resolution through `katalyst.yaml`.

The DSL is expected to grow to support validation for object-like data (YAML,
JSON, SQL-backed records), markdown content, and file/directory structures,
along with more storage backends and tooling that reuses validation rules for
additional operations (especially migrations).

This page is intentionally directional — treat it as scope and rationale, not
a frozen specification. The implementation here represents an early, practical
slice of the broader goal.
