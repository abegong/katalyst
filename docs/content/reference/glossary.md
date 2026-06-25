+++
title = "Glossary"
weight = 50
+++

# Glossary

The canonical vocabulary for Katalyst. Use these terms consistently in code,
docs, and user-facing copy. The general, backend-agnostic vocabulary is
introduced in the [domain model]({{< relref "../deep-dives/domain-model/_index.md" >}});
how each term maps onto today's code is documented in the per-package
`AGENTS.md` files under `internal/`. This page is the quick lookup.

| Term | Meaning |
|---|---|
| **Aggregate** | The descriptive operation an inspector realizes: measuring a distribution across a collection's items rather than fetching or asserting. See **Inspector**. |
| **Attribute** | A named characteristic of an item: a frontmatter key, but also its filename, path, or extension. The general term; a key in the structured object specifically is a **Field**. |
| **Base** | One configured backend store plus the operations Katalyst can perform on its content. A raw base gives Katalyst backend-native access; a collectionized base adds collection definitions. |
| **BaseInstance** | A configured instance of a BaseType plus how to reach it (for `filesystem`, a root directory). Declared under `.katalyst/bases/`; it embeds the collections it maps. |
| **BaseType** | A known backend kind capable of holding content Katalyst can operate on (`filesystem` today; `sqlite`, `postgresql`, `mongodb` later). |
| **Body** | Everything after the closing frontmatter fence. Preserved verbatim except by `fix`. |
| **Check** | Shorthand for a check instance when context is unambiguous. |
| **Check instance** | One configured check attached to a collection: a check type plus its arguments (one YAML object under `checks:`). It runs against each item (object, markdown, or filesystem family). |
| **Check type** | The reusable definition of a constraint: one entry in katalyst's check registry (`object_required_field`, `markdown_single_h1`, ...), selected by its `kind:` id. `katalyst check-types list` lists them. |
| **CheckLibrary** | The provider behind a check type. Native libraries (`filesystem`, `plaintext`, `markdownbodytext`, `structuredobject`) wrap hand-written checks; schema-backed libraries (`json-schema`, Vale next) compile a named schema and run items against it, and report their own availability. A library is provenance, orthogonal to the source-data family (`structuredObject`, `markdownBodyText`, `fileSystem`, `plainText`) the check reads. |
| **Collection** | A named entry in `collections:`: a directory, a filename `pattern`, and the checks its items must pass. |
| **Collection layer** | Inspectors that profile a configured collection's items, addressed by domain identity (collection + item id) and probing through the same substrate the checks use. |
| **Collection-scoped check** | A check type that runs once per collection over all its items (e.g. `filesystem_unique_filename`), rather than per item. It re-scans the full collection even under a single-item selector. |
| **CollectionDefinition** | The two-way mapping from a BaseInstance's contents to collections and items. Yields one or more collections; the filesystem is the only backend today. See [Bases]({{< relref "../deep-dives/domain-model/storage.md" >}}). |
| **Config** | A **Project**'s configuration: the schemas, bases, and collection definitions that declare what the project contains and how its items are checked. Katalyst's config is the `.katalyst/` directory; it is loaded by the `project` package's loader (`internal/project/loader.go`). Each object type owns the parse of its own config: the storage registry validates a declared `type`, and a collection parses its own block in `storage/collection`. |
| **Discriminator** | The `when` predicate that selects a variant: a list of `item list --filter` expressions over an item's metadata, ANDed together. |
| **Document** | The markdown file-form of an **Item**: a parsed markdown file (frontmatter metadata + body + a line map). Use it where parsing or the on-disk file is the subject; elsewhere prefer **Item**. |
| **Evidence** | The structured result of one inspector: counts and distributions with the unit count `n` as denominator. Never a recommendation or verdict. |
| **Field** | A key in an item's structured object (its frontmatter map). A field is an **Attribute**; a filename is an attribute but not a field. The term used wherever object or frontmatter keys are meant (`object_field_type`, `name_matches_field`). |
| **Fingerprint** | A file's composite signature (frontmatter keys, body section skeleton, and file type/naming) that `document_shape` clusters into candidate collections. |
| **Frontmatter** | The on-disk metadata block at the top of a markdown file, in YAML (`---`), TOML (`+++`), or JSON (`{ … }`). |
| **Inspector** | A read-only operation that measures content and returns evidence. The descriptive dual of a check: a check asserts a predicate, an inspector reports the distribution. Inspectors come in two layers. |
| **Item** | The unit of data in a collection, addressed by a selector and operated on by `check`, `fix`, and the `item` subcommands. In the filesystem backend an item is one file matching the collection's pattern, its id the filename stem; its markdown file-form is a **Document**. |
| **Measurement primitive** | A reusable building block the inspectors are built from: `object_fields` (a data dictionary over object maps), `markdown_body` (body structure), and file-metadata. |
| **Metadata** | The parsed, in-memory structure of the frontmatter (a `map[string]any`). |
| **Operation** | Something a base lets you do with its data: read, list, query, aggregate, write. Each has a scope (item, collection, across collections) and structural requirements the backend must satisfy. See [progressive operations]({{< relref "../deep-dives/progressive-operations.md" >}}). |
| **Profile class** | A group of near-identical profiles the summarizer collapses together, so output is proportional to the number of distinct profiles, not directories. |
| **Project** | The whole katalyst workspace: a repo root with a `.katalyst/` **Config** that declares the bases, collections, and checks katalyst operates over. The top-level scope an empty selector addresses, and what `katalyst init` creates. Collections live within a project; the `project` package (`internal/project`) is its code home, holding the `.katalyst/` loader while the collection layer lives under `storage/`. |
| **Raw-source layer** | Inspectors that profile a backend store directly, before any collection configuration, addressed by backend-native reference (a path today). The onboarding case: "what's in this store?" |
| **Repo root** | The directory containing the `.katalyst/` config directory; the base for all path resolution. |
| **Resolver** | The runtime object that decides which object schema applies to an item and caches compiled schemas per `(library, path)`. |
| **Schema** | The definition of a collection's shape, expressed in a CheckLibrary's format (JSON Schema today; a Vale style config later). Named in `schemas:`; located by path. The katalyst concept, not the JSON Schema document specifically. |
| **Schema directive** | The inline `schema:` key inside a document's frontmatter, opting it into a named schema. |
| **Selector** | How a command names what to operate on: nothing (whole project), `<collection>`, or `<collection>/<item>`. |
| **Scope** | The level an operation or backend mapping applies to: item, collection, project, or across collections. In the base layer, scope answers whether one matched backend unit becomes an item or a collection. |
| **Span** | The slice of body text a text rule is evaluated against, chosen by its `target`: the whole `body`, each `line`, the `first-line`, or `matched-lines` (lines matching a `select` regex). |
| **Target** | The slice of a path a filesystem name/path check type tests: `filename`, `filename-ext`, `parent-dir`, or `path-segments` (every directory segment plus the basename). For a text rule, the slice of body it tests, see Span. |
| **Text rule** | A `text_*` check (`text_requires`, `text_forbids`, `text_denylist`) that tests the body as raw text, a regex or a literal denylist, independent of markdown structure. Applies to plain-text items too. |
| **Validation result** | The product of running an item's checks: either `path: OK`, or a flat list of violations. |
| **Variant** | A discriminated check group inside a collection (one entry of `variants:`): a `when` discriminator plus the schema/checks added for items that match it. An item runs the base checks plus the first matching variant's. |
| **Violation** | One failed check, reported as `path:line: /pointer: message`. |

## Usage notes

- A **check type** is the definition; a **check instance** is that check type
  configured in a collection and run against a specific item, and a
  **violation** is a check that failed. The [check types
  reference]({{< relref "check-types/_index.md" >}}) and `katalyst check-types
  list` enumerate check types.
- Prefer **schema** for what users author. The runtime check is the `object`
  check type, provided by the JSON Schema **CheckLibrary**; "validator" is not a
  thing users write.
- Use **frontmatter** for the on-disk block and **metadata** for the parsed
  structure; they are not interchangeable.
- Say **`.katalyst/`** or "the config" rather than an unqualified
  "config" when ambiguous.
- **Default to the general term; use the specific one only where the form is the
  subject.** *Item* and *attribute* are the general terms; *document* (an item's
  markdown file-form) and *field* (an attribute that is a structured-object key)
  apply only where parsing, the on-disk file, or the object map is specifically
  what you mean. A document is an item and a field is an attribute; the reverse
  does not hold.
