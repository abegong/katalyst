---
name: write-domain-model
description: >-
  Drafts or revises a domain model document — the shared vocabulary for a
  domain's nouns and verbs, defined the way users and the development team think
  about them, independent of code. Use when the user asks to "write a domain
  model", "define the terms for X", "document the domain of Y", wants to capture
  the working vocabulary for a feature, or is reviewing an existing
  domain-model doc.
---

# Writing a domain model

A domain model document is how the team agrees on what to call the objects in a
domain and what those objects are. It exists prior to code: the same terms it
pins down get used in design discussions, code review, error messages, and
product copy. If two people say "collection" and mean different things, the
domain model is the doc that fixes that.

## What belongs

For each domain object:

- The **name** the team uses — the actual vocabulary, not whatever the code or
  config happens to call it.
- A **one-sentence definition** of what it is.
- Optionally, **what it does or what can be done to it** — when the verb is part
  of the term's meaning ("a collection can be checked against its schema"). Skip
  when the noun stands on its own.
- **How it relates** to other objects, when the relationship is load-bearing.

Group related objects together. Lead with the most important one.

## What does not belong

- Type signatures (`type Foo struct{…}`), interfaces, JSON Schemas.
- Identifier formats, file paths, package or function names.
- Config keys, glob mechanics, resolver/cache internals.
- API verbs that map 1:1 onto domain verbs — write the domain verb ("an item
  can be added to a collection"), not the command (`item add`).

If a sentence would change when the implementation changes, it's probably not
domain.

## Voice

Match `.cursor/skills/write-docs/SKILL.md`:

- **Define, don't describe.** "A Check is one rule asserting a condition on an
  item or its attributes." — not "Checks are used to…".
- **Active voice on the verbs.** "A rule binds a glob to a set of checks." — not
  "Checks are bound by rules."
- **You, not we.** No editorial _we_.
- **No padding.** One sentence is the default; a second adds what the thing
  does; a third only when each carries distinct weight.
- **No hedging.** Cut _typically_, _usually_, _can be_, _tends to_. State the
  rule; note the exception once, sharply.

## Shape

```markdown
# Domain model: {domain name}

{One-paragraph framing: what it covers, what falls outside.}

## Core objects

**Term:** Definition. Optional second sentence on what it does.

- **Verb the term.** One-line rule, directly under the term.

**Term:** Definition. ...

## Relationships

{ASCII tree, table, or short prose connecting the objects.}

## {Specialized subsections, only as needed}

{Cross-cutting invariants that don't belong on a single term.}
```

**Verbs belong with their noun** — as bullets under that object's definition,
not in a standalone "Verbs" or "Lifecycle" section. That keeps each object
readable as a unit and stops the doc from splitting into a noun-list-then-verb-
list shape that reads like an API. Reach for a separate section only when an
operation truly spans the whole domain.

Cut sections that don't earn their tokens. A domain model with three objects and
one relationship table is a great domain model.

katalyst keeps its model in two homes: `docs/deep-dives/core-concepts.md` (the
abstract, cross-system model — data interface, item, collection, attribute,
operation, check) and `docs/reference/glossary.md` (the concrete katalyst
vocabulary). For a new large domain, split similarly — an abstract model plus a
concrete vocabulary — or one file per coherent sub-domain plus a short framing
doc.

## Workflow

1. **Find the existing vocabulary first.** Read the relevant specs, product
   copy, existing `product/` docs, and the session. Note the actual words the
   team uses. If two terms compete, surface it — don't silently pick. Prefer
   reusing existing vocabulary to coining new terms.
2. **List the objects.** Five to fifteen per doc is typical. More → split into
   sub-domains.
3. **Define each one.** One sentence: what it *is*, not how it's stored. Read it
   back and ask "would this still be true if we rewrote katalyst in another
   language?" If no, rewrite.
4. **Add jobs where they earn it.** A noun whose meaning is partly verbal gets a
   second sentence; a noun that's just a noun doesn't.
5. **Draw the relationships.** Use the simplest form that works.
6. **Strip the implementation.** Final pass: search for type signatures, config
   keys, file/function names, cache mechanics. Delete or rewrite each.

## Exemplars

- `docs/reference/glossary.md`: the canonical katalyst vocabulary:
  bold-term-colon definitions used across code, docs, and copy.
- `docs/deep-dives/core-concepts.md` — the abstract, cross-system model (data
  interface, item, collection, attribute, operation, check), with an examples
  table mapping it onto Postgres, MongoDB, CSVs, and more.

## Anti-patterns

- Opening with "This document describes the domain model for …".
- Defining a term by what code does with it.
- Type signatures or config keys inside a definition.
- Bullets where a sentence is shorter.
- A relationships section that restates each definition.
- A standalone "Verbs" or "Lifecycle" section that lifts behavior off its noun.
