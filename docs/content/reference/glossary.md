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
| **Frontmatter** | The on-disk metadata block at the top of a markdown file, in YAML (`---`), TOML (`+++`), or JSON (`{ … }`). |
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
| **Collection-scoped check** | A check type that runs once per collection over all its items (e.g. `filesystem_unique_filename`), rather than per item. It re-scans the full collection even under a single-item selector. |
| **Variant** | A discriminated check group inside a collection (one entry of `variants:`): a `when` discriminator plus the schema/checks added for items that match it. An item runs the base checks plus the first matching variant's. |
| **Discriminator** | The `when` predicate that selects a variant — a list of `item list --filter` expressions over an item's metadata, ANDed together. |
| **Target** | The slice of a path a filesystem name/path check type tests: `filename`, `filename-ext`, `parent-dir`, or `path-segments` (every directory segment plus the basename). For a text rule, the slice of body it tests — see Span. |
| **Text rule** | A `text_*` check (`text_requires`, `text_forbids`, `text_denylist`) that tests the body as raw text — a regex or a literal denylist — independent of markdown structure. Applies to plain-text items too. |
| **Span** | The slice of body text a text rule is evaluated against, chosen by its `target`: the whole `body`, each `line`, the `first-line`, or `matched-lines` (lines matching a `select` regex). |
| **Violation** | One failed check, reported as `path:line: /pointer: message`. |
| **Inspector** | A read-only operation that measures content and returns evidence. The descriptive dual of a check: a check asserts a predicate, an inspector reports the distribution. Inspectors come in two layers. |
| **Raw-source layer** | Inspectors that profile a backend store directly, before any collection configuration — addressed by backend-native reference (a path today). The onboarding case: "what's in this store?" |
| **Collection layer** | Inspectors that profile a configured collection's items, addressed by domain identity (collection + item id) and probing through the same substrate the checks use. |
| **Measurement primitive** | A reusable engine the inspectors are built from: `object_fields` (a data dictionary over object maps), `markdown_body` (body structure), and file-metadata. |
| **Evidence** | The structured result of one inspector: counts and distributions with the unit count `n` as denominator. Never a recommendation or verdict. |
| **Fingerprint** | A file's composite signature — frontmatter keys, body section skeleton, and file type/naming — that `document_shape` clusters into candidate collections. |
| **Profile class** | A group of near-identical profiles the summarizer collapses together, so output is proportional to the number of distinct profiles, not directories. |
| **Repo root** | The directory containing the `.katalyst/` config directory; the base for all path resolution. |
| **Resolver** | The runtime object that decides which object schema applies to an item and caches compiled schemas. |
| **StorageType** | A known backend kind capable of holding collections and items (`filesystem` today; `sqlite`, `postgresql`, `mongodb` later). |
| **StorageInstance** | A configured instance of a StorageType plus how to reach it (for `filesystem`, a root directory). Declared under `.katalyst/storage/`; it embeds the collections it maps. |
| **CollectionDefinition** | The two-way mapping from a StorageInstance's contents to collections and items. Yields one or more collections; the filesystem is the only backend today. See [storage layer]({{< relref "../deep-dives/storage.md" >}}). |
| **Granularity** | The level — item vs. collection — at which a StorageType attaches a store's units to the domain model (a markdown file is an item; a SQL table is a collection). |

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
- Say **`.katalyst/`** or "the config" rather than an unqualified
  "config" when ambiguous.
