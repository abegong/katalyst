+++
title = "Structured object"
weight = 40
+++

# Structured object

Structured object is the data surface that exposes named fields and values. Today,
for filesystem markdown collections, it comes from parsed frontmatter metadata.
The domain-model term is broader on purpose: future bases may provide rows,
documents, or API resources directly as structured objects.

## Terms

| Term | Meaning |
|---|---|
| **Structured object** | A map-like representation of an item's structured data. |
| **Field** | A key in the structured object. A field is an attribute; not every attribute is a field. |
| **Metadata** | The parsed markdown frontmatter shape used as the structured object today. |
| **Schema directive** | The inline `schema:` key that opts an item into a named schema before validation. |

## Model

In the current filesystem backend, the structured-object surface is
`Document.Meta` from [Markdown body text]({{< relref "markdown-body-text.md" >}}).
It is normalized to `map[string]any` no matter whether the source frontmatter
was YAML, TOML, or JSON.

Structured-object checks validate fields and schema-backed object shape. They
are the right fit when a check needs to ask about named values: required fields,
field type, field length, enum membership, uniqueness, sentence case, or JSON
Schema validation.

The `schema:` directive is Katalyst metadata. It selects a configured schema for
the item and is removed before the item is validated against that schema.

## Invariants

1. **Field checks read normalized metadata.** They do not branch on YAML, TOML,
   or JSON syntax.
2. **A field is narrower than an attribute.** Filenames and path segments are
   attributes, but they are not structured-object fields.
3. **Schema selection is separate from validation.** The directive chooses the
   schema; the object check validates the resulting structured object.

## See also

- [Structured object check types]({{< relref "../check-types/structured-object/_index.md" >}})
- [Markdown body text]({{< relref "markdown-body-text.md" >}})
- [Configs]({{< relref "../configs/checks.md#object-schema-resolution-precedence" >}})
- [Collections]({{< relref "../../deep-dives/domain-model/collections.md" >}})
