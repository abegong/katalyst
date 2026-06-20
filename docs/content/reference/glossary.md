+++
title = "Glossary"
weight = 50
+++

# Glossary

The canonical vocabulary for Katalyst. Use these terms consistently in code,
docs, and user-facing copy. They are defined conceptually in the [domain
model]({{< relref "../explanation/domain-model.md" >}}); this page is the
quick lookup.

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
| **Check** | A single rule run against an item (object, markdown, or filesystem family). |
| **Violation** | One failed check, reported as `path:line: /pointer: message`. |
| **Repo root** | The directory containing `katalyst.yaml`; the base for all path resolution. |
| **Resolver** | The runtime object that decides which object schema applies to an item and caches compiled schemas. |
| **Connector** | (Future) the two-way mapping between a backend store and the domain model. The filesystem is the only one today. |

## Usage notes

- Prefer **schema** for what users author and **validator** only for the
  runtime check itself — never "validator" as a thing users write.
- Use **frontmatter** for the on-disk block and **metadata** for the parsed
  structure; they are not interchangeable.
- Say **`katalyst.yaml`** or "the config" rather than an unqualified
  "config" when ambiguous.
