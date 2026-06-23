# Domain model terminology matrix

Working reference for the [domain model cleanup
spec](./domain-model-cleanup-spec.md). One row per concept, matched across the
five places terminology lives. The point is to see, at a glance, where the same
concept wears different names (synonyms to reconcile), where the name already
agrees (leave alone), and where a source has no term at all (a gap to fill or a
concept to drop).

This is a snapshot taken 2026-06-23, not a generated artifact. Regenerate by
hand when the code or docs move.

## Sources

| Column | What it captures |
|---|---|
| **Internal code** | Package names and exported Go identifiers under `internal/`. |
| **CLI** | Command/subcommand names and the user-facing nouns in their help text (`cmd/`). |
| **Domain model** | Terms as used in `docs/content/deep-dives/domain-model.md` (katalyst-specific). |
| **Core concepts** | Terms as used in `docs/content/deep-dives/core-concepts.md` (tool-agnostic). |
| **Glossary** | Entries in `docs/content/reference/glossary.md` (the intended single source of truth). |

`—` means the source has no term for that concept. **Bold** marks a term that
source defines or owns; plain text marks an incidental or prose-only mention.

## A. General data model (tool-agnostic layer)

| Concept | Internal code | CLI | Domain model | Core concepts | Glossary |
|---|---|---|---|---|---|
| Data interface / backend | `storage.StorageType`, `config.StorageInstance` | `.katalyst/storage/` | — | **Data interface** | **StorageType** / **StorageInstance** |
| Collection | `config.Collection`, `storage.CollectionDefinition` | `collection` | **Collection** | **Collection** | **Collection** |
| Item | `storage.Item`, `project` resolution | `item`, `<collection>/<item>` | **Item** | **Item** | **Item** |
| Attribute / field | `checks.Field`, `inspect.ObjectFields` | "frontmatter keys", `key=value` | "field" (prose) | **Attribute** | — |
| Operation | — (the commands) | the verbs (check/fix/get/list/…) | "operation" (prose) | **Operation** (Read/List/Query/Aggregate/…) | — |
| Granularity | `storage.Granularity` | — | — | — | **Granularity** |

## B. Katalyst entities (filesystem instantiation)

| Concept | Internal code | CLI | Domain model | Core concepts | Glossary |
|---|---|---|---|---|---|
| Document | `frontmatter.Document` | "Print an **item** (frontmatter + body)" | **Markdown document** | (folded into "item") | **Document** |
| Frontmatter | `internal/frontmatter`, `frontmatter.Kind` | "frontmatter keys" | **frontmatter** | — | **Frontmatter** |
| Metadata | `Document.Meta` | — | **Meta** | "document metadata" | **Metadata** |
| Body | `Document.Body` | "body" | **Body** | — | **Body** |
| Selector | `project.Selector` | `[selector ...]` | **Selector** | — | **Selector** |
| Coordinates / reference | `storage.Reference` | — | **coordinates** | — | — |
| Project | `internal/project`, `project.Project` | "whole project", `init` | "the whole project" | — | (in Selector row only) |
| Repo root | `config.Config.Root` | — | **repo root** | — | **Repo root** |

## C. Schema & resolution

| Concept | Internal code | CLI | Domain model | Core concepts | Glossary |
|---|---|---|---|---|---|
| Schema | `checks.Schema`, `checks.SchemaRef` | `schema` | **Schema** | Schema (in Implications) | **Schema** |
| Schema directive | resolver / `jsonschema.Resolve` | inline `schema:` | **Schema directive** | — | **Schema directive** |
| Resolver | `project.Resolution`, `jsonschema.Resolve` | — | **Resolver** | — | **Resolver** |
| CheckLibrary | `checks.Library` / `CheckLibrary` / `SchemaLibrary` | — | **CheckLibrary** | "external check library" → CheckLibrary | **CheckLibrary** |

## D. Checks

| Concept | Internal code | CLI | Domain model | Core concepts | Glossary |
|---|---|---|---|---|---|
| Check type | `config.CheckType`, `checks.Descriptor` | `check-types`, `kind:` | "check type" (in families) | **check type** | **Check type** |
| Check instance | `config.CheckInstance` | YAML under `checks:` | "Check" | **check instance** | **Check instance** |
| Check (run / interface) | `checks.Check` | `check` | **Check** | **Check** | **Check** (shorthand) |
| Collection-scoped check | `checks.CollectionCheck` | — | "collection-scoped" | — | **Collection-scoped check** |
| Family | `checks.Family`, `inspect.Family` | "grouped by family" | "families" (Object/Markdown/FS) | — | (in CheckLibrary row only) |
| Engine | `cmd/engine.go` | "the **engine** can run/enforce" | — | — | (in Check-type row only) |
| Variant / Discriminator | `config.CollectionVariant` | — | "variant" (invariant 4) | — | **Variant** / **Discriminator** |
| Target / Span / Text rule | `checks.Text*`, target consts | — | "target" (FS only) | — | **Target** / **Span** / **Text rule** |
| Violation | `checks.Violation` | output line | **Violation** | "violations" | **Violation** |
| Validation result | `Result` (diagram) | `path: OK` | **Validation result** | — | — |

## E. Inspectors

| Concept | Internal code | CLI | Domain model | Core concepts | Glossary |
|---|---|---|---|---|---|
| Inspector | `inspect.SourceInspector` / `CollectionInspector` | `inspect`, `inspectors` | **Inspector** | **Inspector** | **Inspector** |
| Evidence | `inspect.Evidence` | — | **evidence** | **evidence** | **Evidence** |
| Raw-source layer | `inspect.SourceView`, `inspect.Layer` | "source" (dir/help) | **Raw-source** | **Raw-source** | **Raw-source layer** |
| Collection layer | `inspect.CollectionView` | "collection" | **Collection** layer | **Collection** layer | **Collection layer** |
| Aggregate (operation) | — | — | "aggregate operation" | **Aggregate** | — |
| Measurement primitive | `inspect.ObjectFields` / `MarkdownBody` | — | "measurement primitives" | — | **Measurement primitive** |
| Profile / Fingerprint / Profile class | `inspect.Profile`, `DocumentShape` | — | — | — | **Fingerprint** / **Profile class** |

## F. Config & query

| Concept | Internal code | CLI | Domain model | Core concepts | Glossary |
|---|---|---|---|---|---|
| Config / `.katalyst` | `config.Config` | `init`, `.katalyst/` | **Config** | — | ".katalyst/" (in notes) |
| Query / filter | `internal/query`, `config.QuerySettings` | `item list --filter` | "Query" (**out of scope**) | **Query** (an operation) | (in Discriminator row) |

## Conflicts and gaps the matrix exposes

Ordered roughly by how much they hurt. Resolutions are recorded in the
[spec](./domain-model-cleanup-spec.md); `OPEN` marks the two still unresolved.

1. **`item` vs `document`.** The single biggest collision. Core concepts and the
   CLI say *item*; domain model, glossary, and code say *document*
   (`frontmatter.Document`). **Resolved:** *item* is primary and general;
   *document* is the markdown file-form specialization, used only where parsing,
   the body/frontmatter structure, or the raw file is the subject.
2. **"Data interface" vs "Storage\*".** Core concepts coined *data interface*;
   everything downstream went with *StorageType* / *StorageInstance*. **Resolved:**
   *data interface* is deprecated; storage vocabulary wins everywhere.
3. **"engine" is undefined everywhere.** It appears only in CLI help text and
   `cmd/` code ("the engine can run"), yet no doc defines it. **`OPEN`** (spec
   OQ 2): lean toward dropping it from user-facing copy.
4. **"Query" contradicts itself.** Core concepts lists it as a supported
   operation; domain model lists it as explicitly *out of scope*; meanwhile
   `internal/query` and `item list --filter` exist. **`OPEN`** (spec OQ 1): owner
   investigating; lean toward "partially shipped."
5. **Glossary gaps for tool-agnostic terms.** *Attribute*, *Operation*, and
   *Aggregate* are core-concepts-only abstractions with no glossary entry.
   **Resolved:** glossary is canonical, so each gains an entry.
6. **`attribute` vs `field`.** Core concepts says *attribute*; code and CLI say
   *field*; the glossary names neither. **Resolved:** *attribute* is the general
   umbrella (any named characteristic, including filename/path); *field* is the
   object-key specialization. Field ⊂ attribute.
7. **"source" vs "raw-source".** Code and CLI shortened it; the docs did not.
   **Resolved:** *source* wins; rename the docs.
8. **`family` vs `library` vs `kind`.** Three orthogonal axes that only the
   glossary tries to pin, and even it relegates *family* to a sub-clause instead
   of its own entry. **Resolved:** glossary gains a standalone *family* entry
   alongside *library* and *kind*.
9. **"Validation result" has no glossary entry**, though both the domain model
   and the code (`Result`) name it. **Resolved:** gains a glossary entry.
