---
name: session-to-proposal
description: >-
  Drafts a concise decision proposal in the form "we should do X because Y" from
  the current chat or session topic. Use when the user asks to turn something
  into a proposal ("let's turn this into a proposal"), wants a proposal drafted
  from recent discussion, or when summarizing an agreed direction for a feature,
  refactor, code standard, or process change. May also be offered proactively
  when a clear decision statement would help — especially after discussion.
---

# Session → proposal

## Goal

Turn what was just discussed into a **proposal**: a concrete **X** (action) and
**Y** (reasoning grounded in shared context). Prefer speed and clarity over
polish.

## Triggers

- "Turn this into a proposal," "write this up as a proposal," "proposal for what
  we discussed."
- The thread has converged on a direction but it's still implicit — a short
  proposal makes the next step obvious.

If the ask is vague **and** the topic is unclear from context, ask **one or
two** focused questions before drafting.

## Default shape (short first)

Start small. **Shorter is better** unless complexity forces more.

1. **Proposal** — One sentence: what we should do (**X**). Often starts with
   "Proposal:" or "Let's …".
2. **Reasoning** — **2–5 bullets** that justify **X** (**Y**). Each ties to
   something the reader already shares — prior decisions, docs, goals,
   constraints — or flags what's new.

**Everything after Reasoning is optional.** Add further sections only when the
stakes warrant it. Do not pad.

## Optional sections (after reasoning)

- **Specifics** — More detail on the likely implementation path, to confirm
  alignment on direction (not to specify every detail).
- **Supporting artifact** — A link to a spec, doc section, PR, or sketch when X
  or Y has many moving parts.
- **Assumptions / new concepts** — Separate net-new ideas from shared grounding.
- **Alternatives considered** — Including "do nothing" / "defer," if the
  discussion actually weighed options.
- **Risks / mitigations** — When failure modes matter.
- **Open questions** — When Y is partly ungrounded; say what would resolve it.
- **Decision ask** — What you need from the reader (approve, choose A vs B), if
  not obvious.

## Where it lands

A proposal is the lightweight front end of katalyst's decision flow. When it's
accepted, graduate it: open questions become GitHub issues (or live in the
in-flight spec), and the locked rationale folds into the relevant
`docs/deep-dives/` page — there is no central decisions log. See
`docs/contributing/how-we-plan.md`.

## Principles

- **X** can be any follow-up: build, research, rename, adopt a standard, stop
  doing something.
- **Y** should lean on **shared** docs, decisions, and priorities; introduce new
  assumptions only when necessary and call them out.
- Substance (action + reason) matters more than formatting.

## Examples (short form)

**Example 1 — naming**

```markdown
Proposal: name the conformance verb `check`, not `validate`.

- The engine's primitive is already a `Check` (`internal/checks/`, `CheckType`) —
  `check` runs the checks; `validate` would be a second word for the same idea.
- It covers the non-schema rules too (filename, headings), which "validate"
  (schema-conformance) doesn't connote.
```

**Example 2 — config structure**

```markdown
Proposal: replace the anonymous `rules:` list in katalyst.yaml with named
`collections:`.

- Addressing (`notes/dune`) needs named collections to resolve against; the CLI
  spec depends on it (`product/specs/cli-spec.md`).
- It makes `collection` a first-class noun, matching the domain model
  (`docs/reference/glossary.md`) instead of an implicit glob.
```

## Workflow

1. Infer **X** and the strongest **Y** bullets from the session.
2. If **X** is ambiguous or **Y** would be mostly hand-wavy, ask brief
   clarifying questions.
3. Emit the **Proposal** line + **Reasoning** bullets first.
4. Add optional sections only if they reduce confusion or record something the
   session already established.
