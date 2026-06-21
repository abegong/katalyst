+++
title = "Glossary"
weight = 50
+++

# Glossary

The canonical vocabulary for Katalyst. Use these terms consistently in code,
docs, and user-facing copy. The general, backend-agnostic vocabulary is
introduced in [core concepts]({{< relref "../deep-dives/core-concepts.md" >}});
how each term maps onto today's code is documented in the per-package
`README.md` files under `internal/`. This page is the quick lookup.

| Term | Meaning |
|---|---|
| **Frontmatter** | The on-disk YAML block delimited by `---` fences at the top of a markdown file. |
| **Metadata** | The parsed, in-memory structure of the frontmatter (a `map[string]any`). |
| **Body** | Everything after the closing frontmatter fence. Preserved verbatim except by `fix`. |
| **Document** | A parsed markdown file: frontmatter metadata + body + a line map. |
| **Schema** | A JSON Schema document. Named in `schemas:`; located by path. |
| **Schema directive** | The inline `schema:` key inside a document's frontmatter, opting it into a named schema. |
| **Collection** | A named entry in `collections:` — a directory, a filename `pattern`, and the checks its items must pass. |
| **Item** | One file in a collection that matches its pattern. Its id is the filename stem. |
| **Selector** | How a command names what to operate on: nothing (whole project), `<collection>`, or `<collection>/<item>`. |
| **Check type** | The reusable definition of a constraint — one entry in the engine's check registry (`object_required_field`, `markdown_single_h1`, …), selected by its `kind:` id. `katalyst check-types list` lists them. |
| **Check instance** | One configured check attached to a collection: a check type plus its arguments (one YAML object under `checks:`). It runs against each item (object, markdown, or filesystem family). |
| **Check** | Shorthand for a check instance when context is unambiguous. |
| **Violation** | One failed check, reported as `path:line: /pointer: message`. |
| **Inspector** | A read-only operation that measures a corpus and returns evidence. The descriptive dual of a check: a check asserts a predicate, an inspector reports the distribution. |
| **Evidence** | The structured result of one inspector: counts and distributions with the file count `n` as denominator. Never a recommendation or verdict. |
| **Corpus** | The set of markdown files under an inspected path, parsed once and shared across inspectors. |
| **Fingerprint** | The sorted set of a file's frontmatter keys, used by `frontmatter_shape` to group files into candidate collections. |
| **Repo root** | The directory containing `katalyst.yaml`; the base for all path resolution. |
| **Resolver** | The runtime object that decides which object schema applies to an item and caches compiled schemas. |
| **Connector** | (Future) the two-way mapping between a backend store and the domain model. The filesystem is the only one today. |

## Usage notes

- A **check type** is the definition; a **check instance** is that check type
  configured in a collection and run against a specific item, and a
  **violation** is a check that failed. The [check types
  reference]({{< relref "check-types/_index.md" >}}) and `katalyst check-types
  list` enumerate check types.
- Prefer **schema** for what users author and **validator** only for the
  runtime check itself — never "validator" as a thing users write.
- Use **frontmatter** for the on-disk block and **metadata** for the parsed
  structure; they are not interchangeable.
- Say **`katalyst.yaml`** or "the config" rather than an unqualified
  "config" when ambiguous.
