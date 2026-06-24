---
name: katalyst-define-schemas
description: >-
  Define each collection's schema in a katalyst project — the fields its items
  must have and the constraints they must satisfy — plus the checks that enforce
  them. Use when a user has collections and wants to declare their structure,
  add required fields, types, or validation rules. Step 2 of the Define stage;
  its prerequisite is katalyst-identify-collections.
---

# Define schemas

Once the collections exist (see **katalyst-identify-collections** — its
prerequisite), give each one a **schema**: the declared shape its items must
hold. A schema is JSON Schema today — required fields, types, enums, numeric
ranges — named in `.katalyst/schemas/` and bound to a collection through its
checks. This is what turns "these files are sort of alike" into a contract
katalyst enforces.

If the CLI is missing, run `./bootstrap.sh`.

## Author a schema per collection

For each collection, decide the contract from its items' real fields (the catalog
evidence is a good starting point):

- **Required vs optional** — which fields every item must carry, which are
  allowed but not required.
- **Types and constraints** — string/number/boolean, enums for closed value
  sets, ranges for numbers, formats where they matter.
- **Body structure** — required sections, a single H1, naming conventions, where
  those belong (the markdown and filesystem check families, not the object
  schema).

Write the schema under `.katalyst/schemas/<name>.json` and reference it from the
collection. `katalyst schema` inspects configured schemas; `katalyst check-types
list` shows the check kinds available to enforce structure beyond the object
schema.

## Verify against real items

Run checks and read the violations — they are the feedback loop for getting the
schema right:

```bash
katalyst check <collection>
```

If items you expect to be valid fail, the schema is too strict (or the content
needs fixing); if junk passes, it is too loose. Tighten and re-run until the
schema accepts exactly the items it should. Use `katalyst fix <selector>` to
canonicalize frontmatter as you go.

## Next

With schemas defined and checks passing, make enforcement automatic so the
contract holds as content changes: continue to **katalyst-deploy**.
