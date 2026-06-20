+++
title = "Manifesto (Draft)"
weight = 10
+++

# Katalyst Manifesto

Traditional data management often forces teams into binary choices:
structured or unstructured, rigid or chaotic. Katalyst is an experimental
framework aimed at enabling fast, low-risk evolution through progressive typing
in the storage layer.

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

## Introducing Katalyst

Katalyst is an experimental framework for progressive typing in the storage
layer, designed for AI-readiness:

- Validate content across structured and unstructured forms.
- Apply one validation model across filesystems and databases.
- Use validation rules not only for checks, but as primitives for migration and
  system evolution.

This manifesto is intentionally directional. The implementation in this
repository represents an early, practical slice of that broader goal.
