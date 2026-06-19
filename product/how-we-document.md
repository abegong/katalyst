# How we document

## Goals

- Agents get the context they need to work in any part of the repo.
- Humans get scannable reference without wading through agent-oriented detail.
- One source of truth per topic — no drift between duplicate files.
- Docs live close to what they describe.

## Where each kind of doc lives

katalyst has four homes for documentation, each with a distinct audience.

### `AGENTS.md` — operational conventions

Rules for anyone *writing code* in the repo: commands, layout, testing
style, code style. **What goes here:** naming conventions, required
patterns, gotchas, and the *why* behind a constraint. **What doesn't:**
conceptual explanations of how the system works (→ `product/`), design
rationale (→ `product/decisions.md`), or API-level detail (→ Go doc
comments).

katalyst keeps a **single root `AGENTS.md`**. Add a subdirectory `AGENTS.md`
only when a package has rules that genuinely don't belong at the root.
Because there's one file, we don't need a namespaced rule-slug system —
revisit that only if per-package `AGENTS.md` files proliferate. Examples
live in tests, not a separate examples file: a `*_test.go` is the canonical,
executable example (see `AGENTS.md` → Testing).

### `product/` — design & reference

Internal docs about how the system works and why it's built that way. For
the development team (human and agent). Subtypes:

- **Architecture / domain model** (evergreen): [`domain-model.md`](domain-model.md),
  [`domain-model-mapping.md`](domain-model-mapping.md),
  [`general-model.md`](general-model.md),
  [`progressive-operations.md`](progressive-operations.md),
  [`connectors.md`](connectors.md). *"How does the system work?"*
- **Decision log:** [`decisions.md`](decisions.md) (resolved, with
  D-numbers and rationale) and [`decisions-to-make.md`](decisions-to-make.md)
  (open questions).
- **Specs & plans** (design docs for in-flight or shipped work) live in
  `product/specs/`: e.g. [`cli-spec.md`](specs/cli-spec.md). See
  [`how-we-plan.md`](how-we-plan.md).
- **Roadmap:** [`roadmap.md`](roadmap.md) — sequencing and what's next.
- **Process:** this file and [`how-we-plan.md`](how-we-plan.md).

**What doesn't go here:** operational rules (→ `AGENTS.md`), user-facing
usage (→ `docs/`), code-level API docs (→ Go doc comments).

### `docs/` — user-facing site (Hugo)

Everything a *user* needs: getting started, command reference,
configuration, the rules reference, the manifesto. Built and served by Hugo
(`make docs-serve` / `make docs-build`). Typically the last docs to update,
after the surface stabilizes.

### `README.md` — project overview

Install, quickstart, the command surface at a glance, and the dev layout.
Only content that isn't just a subset of `AGENTS.md` or `docs/`.

### Go doc comments — code-level API docs

Package- and symbol-level documentation lives in the code as Go doc
comments (the `godoc` convention), not in Markdown. This is what a JS
project would put in JSDoc.

## What goes where (quick matrix)

| You're documenting… | It goes in… |
|---|---|
| A rule for writing code here | `AGENTS.md` |
| How a subsystem works / a domain concept | `product/` (architecture) |
| Why a choice was made | `product/decisions.md` |
| An open design question | `product/decisions-to-make.md` |
| A feature/architecture change being designed | a spec in `product/specs/` |
| How to use the CLI | `docs/` + `README.md` |
| What a package/function does | Go doc comments |

## Guidelines

- **Keep `AGENTS.md` lean** — rules and conventions, prose not walls of text.
- **Don't repeat root standards** in subdirectory docs; document only what's
  specific to that location.
- **Update docs in the same change** that establishes a convention or ships
  a feature.
- **Vocabulary is shared.** Use the terms defined in
  [`domain-model.md`](domain-model.md) (and the general model) consistently
  across code, docs, and user-facing copy.

## Tool-specific files

`AGENTS.md` is the source of truth. Other tools read their own files; keep
them thin and pointed at `AGENTS.md` rather than duplicating it.

- **`.cursor/skills/`** — reusable skills (e.g. `add-katalyst-rule`). Skills
  are *actions*, not conventions; conventions stay in `AGENTS.md`.
- **`.claude/`** — Claude Code local settings, not a documentation source.
