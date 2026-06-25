+++
title = "Collections"
weight = 42
+++

# Collections

The `internal/project` loader (`loader.go`) is the orchestration hub: it loads a
project's `.katalyst/` directory, resolves named schemas, and assembles bases
and their collections. Each object type parses its own config: the base
registry validates a declared `type`, and a collection parses its own block in
`storage/collection`. It decides which schema applies to a given item, and the
`check` lifecycle is driven from here.

This page is the model and the *why*; for the key-by-key surface see the
[configuration reference]({{< relref "../../reference/configuration.md" >}}).

## The `.katalyst/` directory

Configuration lives in a `.katalyst/` directory, discovered by walking **up**
from the working directory to the nearest ancestor that contains one. That
ancestor becomes the repo root for all path resolution.

The directory holds an optional `config.yaml`, one schema file per definition
under `schemas/`, and one base file per definition under `bases/`. A directory
rather than one big file keeps each schema and base in its own reviewable file
and lets the name fall out of the filename by convention. A nearest-ancestor
lookup mirrors `.git`, `.editorconfig`, and `go.mod`: familiar and predictable.
Discovery resolves symlinks on both the root and the input path, because on
macOS `$TMPDIR` lives behind `/var` to `/private/var` and relative-path
resolution would otherwise produce garbage.

`config.yaml` is YAML; schema and base files default to YAML/JSON and the
accepted format is set per kind there. Default discovery is **convention**: one
file per definition. A kind can be switched to **explicit** to list its
definitions inline in `config.yaml` instead.

Collections are declared *inside* a [base]({{< relref "base.md" >}}), which owns
the base-to-collection mapping. This page covers the collection model and
schema resolution; the base page covers how a base maps a backend source onto
those collections.

## Terms

| Term | Meaning |
|---|---|
| **Collection** | A group of items that share structure: a directory of similar files, a relational table, a Mongo collection, or a family of API resources. Collections are the unit that owns checks and that users address by name. |
| **Item** | One unit of data in a collection: a markdown file, a table row, a Mongo document, or one API resource. In the filesystem base, an item is one file matching the collection's `pattern`; its **id** is the filename stem (`notes/books/dune.md` gives `dune`). |
| **Attribute** | A named characteristic of an item: a column, a frontmatter key, a response field, its filename, its path, or another backend-derived property. A key in a structured object specifically is a **field**. |
| **Selector** | How commands (`check`, `fix`, the `item` subcommands) name what to operate on, broad to narrow: *(none)* is the whole project, `<collection>` is one collection, `<collection>/<item>` is a single item. |
| **Schema** | A JSON Schema (draft 2020-12 by default) describing the legal shape of an item's parsed `Meta`. A schema has two identities: a **path** on disk and a **name** (its filename stem under `.katalyst/schemas/`). The name is the stable public handle; paths can change. `--schema <path>` bypasses the name layer entirely. |
| **Schema directive** | A per-document `schema:` frontmatter key that opts the document into a specific schema. It is **metadata about katalyst, not user data**: the resolver reads it to choose a schema, then strips it from `Meta` before validating, so a schema with `additionalProperties: false` is not tripped by katalyst's own key. |

The `config.Config` loaded from disk is the single source of truth for "what
schemas exist and what each collection checks." It is validated at load: every
collection's object schema must reference a known schema, and a collection must
configure at least one check via the `schema:` shorthand or an explicit
`checks:` list.

## Collections across backends

The collection model is intentionally broader than "a directory of markdown
files." A collection is the named group Katalyst can list, select, inspect, and
check, even when the backing base has a different native vocabulary.

| System               | Base          | Collection      | Item       | Attribute        |
|----------------------|---------------|-----------------|------------|------------------|
| Postgres             | The database  | A table         | A row      | A column         |
| MongoDB              | The database  | A collection    | A document | A field          |
| A directory of CSVs  | The directory | A CSV file      | A row      | A column         |
| A REST API           | The API       | A resource type | A resource | A response field |
| An S3 bucket of JSON | The bucket    | A key prefix    | An object  | A JSON key       |

An operation defined against this vocabulary, such as checking an attribute or
aggregating over a collection, applies to every base that can support it. The
base still decides the mechanics: a filesystem may list files and parse
frontmatter in memory, while a database may push filtering and aggregation into
queries. The collection name stays the user's handle either way.

## Why schema resolution has three tiers

When `check` validates an item against an object schema, it resolves which
schema, highest precedence first:

| # | Source                        | When it wins |
|---|-------------------------------|--------------|
| 1 | `--schema <path>` flag        | Always, for every item in the invocation |
| 2 | Inline `schema: <name>` in FM | When (1) absent and `Meta["schema"]` is a known name |
| 3 | The collection's object check | When (1) and (2) absent |
| - | None                          | The item simply runs no object check |

Command-line beats inline beats config because that orders the sources from most
specific intent to most general: the flag is the operator's override for this
run, the file's author has the most local information about what the file is,
and the collection is the bulk-association default. Markdown and filesystem
checks are *not* subject to this precedence; they always come from the
collection, since they describe the item's place in the project rather than its
object shape.

Resolution runs through a per-invocation **resolver** that owns this policy and a
compiled-schema cache keyed by absolute path, so "check 10,000 files against the
same schema" costs one compile.

## Why variants discriminate by metadata, not path

A collection's `variants:` run extra checks on a subset of items, chosen by the
item's metadata. The discriminator (`when`) reuses the `item list --filter`
predicate grammar (`internal/storage/collection/predicate`), validated at load
via `predicate.Parse` so a bad expression fails fast. A variant's `schema:`
folds into a leading object check exactly like a collection's, so the engine
compiles base and variant through one path.

The discriminator is metadata, not a glob, on purpose: metadata is the one
property every item yields on every backend (frontmatter for a file, columns for
a future row), so routing stays portable and the engine never depends on the
base type. Selecting by *path* is a base-type-scoped condition, deferred. The
base page covers [how variants route checks rather than
membership]({{< relref "base.md" >}}).

## Why a file inside a collection must match

A file that sits inside a collection's directory but does not match its
`pattern` is reported as an **error**, not silently skipped. Silent skips hide
config drift: a typo'd pattern or a misfiled document would simply disappear
from validation. Opt-outs (`--allow-unmatched` and a config knob) are deferred
until real usage shows the need. The base page frames the same decision as
[unmatched references being first-class]({{< relref "base.md" >}}).

## Why named collections replaced the old `rules:` list

Earlier versions used a flat, ordered `rules:` list of `{paths: <glob>, schema:
<name>}` pairs, where the *first matching glob wins*. Named collections replaced
it for three reasons:

- **Identity.** A collection has a name, so commands can address it (`check
  books`, `item list books`). An anonymous glob rule cannot be named or
  selected.
- **No precedence puzzles.** Glob ordering made the active rule for a file
  depend on the order of unrelated entries. A file now belongs to exactly one
  collection - the one whose directory contains it - so there is no "first match
  wins" to reason about.
- **More than schemas.** A collection carries a whole `checks:` list (markdown
  and filesystem checks, not just an object schema), which the old `{paths,
  schema}` shape could not express cleanly.

The `schema: <name>` shorthand is the one piece of the old model that survived:
sugar for a single leading `object` check.

## Lifecycle of `check`

The data flow per item, end to end:

1. **Load config** (or take the `--schema` flag). Discover the `.katalyst/`
   directory from the working directory; failing to find one is a usage error.
2. **Resolve selectors to items.** No selector means every collection;
   `<collection>` means all its items; `<collection>/<item>` means one. Files
   inside a collection directory that do not match its `pattern` are unmatched
   references (errors).
3. **Read file bytes.** Read errors are reported per item but do not abort the
   run; exit-1 status accumulates.
4. **Parse frontmatter.** Malformed YAML/TOML/JSON is a per-item failure; no
   frontmatter is itself an error.
5. **Resolve the object schema** via the precedence above, then **strip the
   `schema:` directive** so user schemas with `additionalProperties: false`
   are not tripped by katalyst's own metadata.
6. **Build the check list** from the resolved object check plus the collection's
   markdown and filesystem checks.
7. **Run checks** (see [Checks]({{< relref "checks.md" >}})).
8. **Format output**: `path:line: /pointer: message` per violation; valid items
   print `path: OK`.

## Invariants

1. **Schema names are stable; paths can move.** The `.katalyst/` config is the
   only place that maps names to paths.
2. **The `schema:` directive is katalyst metadata, not user data.** It
   influences resolution but never reaches the validator.
3. **A collection owns its checks; an item belongs to one collection.** There is
   no glob-ordering "first match wins" - an item's checks are those of the
   collection whose directory contains it.
4. **Unmatched is an error, not a warning.** Silent skips hide config drift.

## See also

- The [configuration reference]({{< relref "../../reference/configuration.md" >}})
  for the precise `.katalyst/` surface.
- The [base]({{< relref "base.md" >}}) for how a backend source maps onto
  collections, and the base model.
- The [domain model]({{< relref "_index.md" >}}) for the cross-subsystem
  entity map and invariants.
- `go doc ./internal/project` for the code-level contract.
