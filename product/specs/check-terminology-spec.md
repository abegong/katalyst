# Spec — check type and check instance terminology

> **Status: planning.** This spec defines a terminology shift from `rule`/`check` overlap to `check type`/`check instance` language across docs and code.

## Overview

Katalyst currently uses `rule` and `check` to describe overlapping concepts. The overlap is visible in docs, generated reference pages, and code symbols. This spec adopts a two-term model: **check type** for the reusable constraint definition, and **check instance** for one configured occurrence in a collection. The goal is one stable vocabulary that matches the CLI and removes ambiguity.

## Value

Clear terminology lowers contributor and user confusion when configuring checks, reading violations, and extending the engine. It also gives code and docs the same mental model, so naming drift does not reappear as the project grows.

## Current State

Today the project mixes terms in ways that force readers to translate mentally:

- `docs/content/deep-dives/core-concepts.md` defines a check as "a rule that asserts a condition."
- `docs/content/reference/glossary.md` defines check as "a single rule run against an item."
- `docs/content/reference/configuration.md` says each `checks:` entry has a `kind`, then links to a "rule reference."
- `docs/content/how-to/configure-rules.md` uses "rules" and "checks" in the same flow.
- `internal/config/config.go` models configured entries as `Check` with `CheckKind`.
- `internal/checks/registry.go` and `cmd/gendocs/main.go` generate a "Rules reference" and a per-page "Rule ID", even though the source records are check descriptors.
- `product/specs/cli-spec.md` already flags naming tension in its top-level status note.

The result is avoidable friction: users ask for the difference between rule and check, and contributors must infer which term is meant from context.

## Design

### Domain model terms

Katalyst standardizes on the following terms:

- **Check type** — the reusable definition of a constraint, identified by its config id (today the `kind:` value such as `object_field_type`).
- **Check instance** — one configured check attached to a collection (one YAML object inside `checks:`).
- **Check** — an allowed shorthand only when context is unambiguous; in formal docs and code-level API comments, prefer explicit `check type` or `check instance`.

This keeps `check` as the top-level concept while making "definition vs configured use" explicit.

### Public configuration surface

The configuration grammar remains backward-compatible in this change:

- Keep `checks:` as the list key.
- Keep `kind:` as the check type selector in YAML.
- Reframe docs to say "`kind` stores the check type id."

No config migrations are required in this spec's initial implementation.

### Code model alignment

Code symbols that model configuration should align with the same distinction:

- `CheckKind` → `CheckType`
- configured `Check` struct → `CheckInstance` (or `ConfiguredCheck` if package-local readability demands)
- local variables and helper names move from `kind`-centric phrasing to `checkType` where they refer to the conceptual type

This rename is mechanical, test-preserving, and should be done package-by-package to keep diffs reviewable.

### Docs and generated reference alignment

The generated reference remains generated from `internal/checks/registry.go`, but its labels move to check-type language:

- "Rules reference" → "Check types reference" (title/copy)
- "Object Rules / Markdown Rules / Filesystem Rules" → "Object Check Types / Markdown Check Types / Filesystem Check Types"
- "Rule ID" → "Check type ID"

To avoid link churn during rollout, the content path can stay `docs/content/reference/rules/` until a dedicated URL migration is planned.

### Scope and rollout

This spec covers terminology and naming, not behavior changes:

- No check semantics change.
- No command behavior change.
- No new check implementations.

Recommended rollout order:

1. Update glossary, deep-dive language, and configuration/how-to docs.
2. Update generated reference copy and regenerate docs.
3. Rename core code symbols (`internal/config`, `internal/checks`, `cmd/gendocs`) with tests green after each slice.
4. Sweep contributor docs (`docs/content/contributing/`) and `README.md` for final consistency.

## Open Questions

1. Should Katalyst add `type:` as an alias for `kind:` in check instances, or keep `kind:` as the single config key indefinitely?
2. Should the docs URL segment move from `/reference/rules/` to `/reference/check-types/`, with redirects, or remain stable and only change page copy?

## Rejected alternatives

- **Keep `rule` and `check` as-is and document the nuance better.** Rejected because the overlap is structural, not a documentation gap.
- **Rename config key `kind:` immediately.** Rejected for now to avoid unnecessary config churn in the same change as terminology cleanup.
