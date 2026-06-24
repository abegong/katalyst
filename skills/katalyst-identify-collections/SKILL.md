---
name: katalyst-identify-collections
description: >-
  Identify the collections in a katalyst project — the recurring kinds of item
  the content is made of — and declare them in `.katalyst/`. Use when a user
  knows roughly what content they have and wants to define its object types, set
  up collections, or structure a knowledge base. Step 1 of the Define stage;
  points forward to katalyst-define-schemas for each collection's schema.
---

# Identify collections

A **collection** is a named group of like items — the recurring object type a
knowledge base has many instances of (meeting notes, people, endpoints). This
skill names those types and declares them in the project's `.katalyst/` config,
so katalyst knows what the content is made of. Defining each collection's *shape*
(fields, constraints) is the next step, owned by **katalyst-define-schemas**.

If the CLI is missing, run `./bootstrap.sh`. If there is no `.katalyst/` yet, run
`katalyst init` to scaffold the project.

## Find the object types

Work from the catalog evidence (see **katalyst-catalog**) or the content itself.
A good collection is a set of items that are *instances of the same thing*:

- They answer the same question ("who is this person?", "what happened in this
  meeting?").
- They share a body shape and a core of common frontmatter fields.
- They live together or follow a naming/location convention you can express as a
  directory and a filename pattern.

Split when two groups have genuinely different shapes; merge when an apparent
distinction is just an optional field. Prefer a handful of clear collections over
many near-duplicates.

## Declare them

Each collection is an entry in the project config: a directory, a filename
`pattern` for its items, and (later) the checks its items must pass. Add one
entry per object type you identified. Use `katalyst collection` to inspect what
is configured, and `katalyst check` to confirm items resolve into the collections
you expect before going further.

## Next

For each collection you declared, continue to **katalyst-define-schemas** to
formalize the fields its items must have and the constraints they obey. That
skill treats this one as its prerequisite: identify the collections first, then
define each one's schema.
