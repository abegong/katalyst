+++
title = "Schemas"
weight = 20
+++

# Schemas

Each file under `.katalyst/schemas/` is a JSON Schema. Its **name**, the
filename stem, is the stable public handle used by `schema get <name>`, by
an inline `schema: <name>` key in a document's frontmatter, and by a
collection's `schema:` shorthand. The path can move; the name should not.

Schemas are stored flat; the check library that compiles a schema is determined
by the referencing check type's `kind` (the `object` check uses JSON Schema).

See [Checks]({{< relref "checks.md" >}}) for object-schema resolution precedence.
