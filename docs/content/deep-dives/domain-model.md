+++
title = "Domain model"
weight = 40
+++

# Domain model

What `katalyst` is *about*: the concepts it manipulates, how they relate,
and which invariants hold across the system. This is the conceptual map.

This page is katalyst-specific. For the general, tool-agnostic model these
concepts instantiate, the same vocabulary applied to a Postgres table or a
MongoDB collection, see [core concepts]({{< relref "core-concepts.md" >}}).

For *what* the commands do, see the [getting-started
tutorial]({{< relref "../getting-started.md" >}}) and the
[configuration reference]({{< relref "../reference/configuration.md" >}}).
For *why* specific design choices were made, see the per-package `README.md`
files under `internal/` (for example `internal/config`) and the
[commands]({{< relref "../reference/commands.md" >}}) reference. For
*how the code is laid out*, see the repo's `AGENTS.md` files.

## At a glance

```mermaid
flowchart LR
    subgraph Disk["On disk"]
        MD["Markdown file<br/>(frontmatter + body)"]
        SF["Schema file<br/>(JSON Schema)"]
        CF[".katalyst/<br/>(config)"]
    end

    subgraph Parsed["In memory"]
        DOC["Document<br/>Meta + Body + Lines"]
        SCH["Schema<br/>(compiled)"]
        CFG["Config<br/>Schemas + Collections"]
    end

    subgraph Decide["Schema selection"]
        FLAG["--schema flag"]
        INLINE["inline 'schema:' key"]
        COL["collection's object check"]
        RES["Resolver"]
    end

    MD --> DOC
    SF --> SCH
    CF --> CFG

    FLAG --> RES
    INLINE --> RES
    COL --> RES
    CFG --> COL

    DOC -- "Meta (minus schema directive)" --> VAL["Run checks"]
    RES -- "selected Schema" --> VAL
    VAL --> RESULT["Result<br/>(OK + Violations)"]
    DOC -- "Lines" --> RESULT
```

## Entities

### Markdown document

The unit of work. A file on disk with two optional regions:

- A **frontmatter** block at the very top of the file, in YAML (`---`
  fences), TOML (`+++` fences), or JSON (`{ … }`): the formats Hugo,
  Obsidian, and Jekyll emit. The source format is detected on read and
  preserved by `fix`.
- A **body**, everything after the closing fence.

A document *may* have no frontmatter, in which case `check` reports it as an
error (the file claimed no metadata, so we couldn't check anything).

When parsed, a markdown document becomes a `frontmatter.Document`:

| Field            | Meaning |
|------------------|---------|
| `HasFrontmatter` | Did the file open with `---`? |
| `Meta`           | Parsed YAML, normalized to `map[string]any` |
| `Body`           | Bytes after the closing fence, **never modified** except by `fix` |
| `Lines`          | JSON-pointer-path → 1-indexed source line |

The `Lines` index is what makes error messages locatable. It accounts for
the opening `---` fence offset, so `Lines["/title"] = 2` means the
`title:` key is on line 2 of the original file.

### Schema

A JSON Schema (draft 2020-12 by default; the library supports 4 through
2020-12) describing the legal shape of a document's `Meta`.

A schema has two identities:

- A **path** on disk, where the JSON lives.
- A **name**, by default its filename under `.katalyst/schemas/` (e.g.
  `book.json` → `book`). The name is the stable public handle used by inline
  `schema:` directives and `schema show`. Paths can change; names should not.

`--schema <path>` bypasses the name layer entirely, useful for ad-hoc
runs but skips name-based identity.

In memory, schemas are `validator.Schema`: a thin wrapper around
`santhosh-tekuri/jsonschema/v6`, kept so the rest of the codebase doesn't
depend on the library directly.

### Config

The single source of truth for "what schemas exist and what each
collection checks." Lives in a `.katalyst/` directory at the **repo root**:
an optional `config.yaml`, plus one file per schema under `schemas/` and one
per collection under `collections/`.

Discovery walks upward from the current working directory looking for
the nearest ancestor that contains a `.katalyst/` directory (cf. `.git`,
`.editorconfig`, `go.mod`). The discovered directory *is* the repo root for
all subsequent path resolution.

A `config.Config` has:

| Field         | Meaning |
|---------------|---------|
| `Root`        | Absolute, symlink-resolved repo-root directory |
| `Schemas`     | Schema name → absolute file path |
| `Collections` | The named collections, in name order |

Validated at load time: every collection's object schema must reference a
known entry in `Schemas`, and a collection must configure at least one
check (via `schema:` shorthand or an explicit `checks:` list).

### Collection

A **named** group of items backed by a directory. It is the unit you select
on the command line and the unit that owns a set of checks. Each collection
is one file under `.katalyst/collections/`, its filename the collection name:

```yaml
# .katalyst/collections/books.yaml
path: notes/books   # directory, relative to the repo root
pattern: "*.md"      # filename glob; default "*.md"
schema: book         # shorthand for a single object check
checks:              # any additional checks
  - kind: markdown_title_matches_h1
```

`path` defaults to the collection name; `pattern` defaults to `*.md`. The
`schema:` shorthand is sugar for a leading `object` check. A collection
without an explicit `path` looks for a directory named after itself.

### Item

A single member of a collection: one file matching the collection's
`pattern`. Its **id** is the filename stem (`notes/books/dune.md` →
`dune`). On the command line an item is addressed by a **selector**,
`<collection>/<item>` (see below).

### Selector

How commands name what to operate on. Three shapes, from broad to narrow:

| Selector | Scope |
|---|---|
| *(none)* | the whole project, every collection |
| `<collection>` | one collection, all its items |
| `<collection>/<item>` | a single item |

Selectors are shared by `check`, `fix`, and the `item` subcommands. They are
the present-day, degenerate case of the
[coordinates]({{< relref "storage.md" >}}) idea from the storage layer: today an
item is named by one coordinate (its stem); richer layouts grow into more.

### Schema directive (`schema:` in frontmatter)

A per-document opt-in to a specific schema. Treated as **metadata about
katalyst itself, not user data**: the resolver reads it to choose a
schema, then strips it from `Meta` before passing to the validator. This
matters when a schema uses `additionalProperties: false`, the document
can still "name itself" without the directive becoming a validation
violation.

### Resolver

Not a persistent entity, a per-`check`-invocation object. Owns:

1. The object-schema selection policy (which schema applies to an item?).
2. A compiled-schema cache keyed by absolute path. The cache makes
   "check 10,000 files against the same schema" cost one compile.

The selection policy, highest precedence first:

| # | Source                          | When it wins |
|---|---------------------------------|--------------|
| 1 | `--schema <path>` flag          | Always, for every item in the invocation |
| 2 | Inline `schema: <name>` in FM   | When (1) absent and `Meta["schema"]` is a known name |
| 3 | The collection's object check   | When (1) and (2) absent |
|, | None                            | The item simply runs no object check |

Markdown and filesystem checks always come from the collection, regardless
of how the object schema is resolved.

### Check

A single check run against an item: a type constraint, a heading requirement,
a filename convention. Each comes from one of the check types Katalyst ships,
in three families:

- **Object**: `object` (full JSON Schema), plus targeted
  `object_required_field`, `object_field_type`, `object_field_enum`,
  `object_number_range`, `object_string_length`.
- **Markdown**: `markdown_title_matches_h1`, `markdown_requires_h1`,
  `markdown_single_h1`, `markdown_no_heading_level_jumps`,
  `markdown_required_section`, `markdown_code_fence_language_required`.
- **Filesystem**: name and path conventions built on a shared **target**
  (`filename`, `filename-ext`, `parent-dir`, `path-segments`): `name_case`,
  `name_matches_field`, `name_affix`, `name_regex`, `name_length`,
  `path_charset`, `path_depth`, `parent_dir_matches_field`, `extension_in`,
  `parent_dir_in`, `referenced_files_exist`, plus **collection-scoped** checks
  that reason across an entire collection (`unique_filename`, `unique_field`,
  `index_file_required`).

Most check types implement the `checks.Check` interface (`Run(Context)
[]Violation`), one item at a time. **Collection-scoped** check types implement
`checks.CollectionCheck` (`RunCollection(CollectionContext) []Violation`) and
run once per collection over all its items, so a single-item selector still
re-scans the whole collection. Each check type is documented in the generated
[check types reference]({{< relref "../reference/check-types/_index.md" >}});
the per-check-type descriptors in `internal/checks/registry.go` are the source
of truth, so a new check type cannot ship undocumented.

### Validation result

The product of running an item's checks. Two states:

- **Valid**: nothing to print except the conventional `path: OK`.
- **Invalid**: a flat list of violations, each with a JSON pointer `Path`
  and a `Message`. JSON Schema's raw error tree is nested and unhelpful for
  line-level reporting, so it is flattened.

When combined with `Document.Lines`, a violation becomes a
`path:line: /pointer: message` user-visible line. If the exact pointer has
no recorded line (e.g. for "missing required property" errors), the resolver
walks up to the nearest ancestor that does, pointing at the parent object
is better than pointing at nothing.

### Inspector

The descriptive dual of a check, in `internal/inspect`. A check asks "does
this item satisfy a predicate?" and returns violations; an **inspector** asks
"what is the distribution of this aspect across the corpus?" and returns
**evidence:** counts and distributions with the file count `n` as the
denominator. It realizes the `aggregate` operation from the [core
concepts]({{< relref "../deep-dives/core-concepts.md" >}}).

Inspectors are read-only and never recommend. They report that `status` is
present in 142 of 142 items with three distinct values; deciding that this
warrants a `required` field with an `enum` is *judgment* that belongs to
whoever reads the evidence, a human or an agent, not to the inspector.

Inspectors come in **two layers**, distinguished by how they reference the
data, the same seam as [storage]({{< relref "storage.md" >}}):

- **Raw-source** inspectors profile a backend store directly, before any
  collection configuration, addressed by backend-native reference (a path
  today). This is the onboarding case: "what's in this directory?" `file_tree`,
  `file_tree_content`, and `document_shape` live here.
- **Collection** inspectors profile a configured collection's items, addressed
  by domain identity (collection + item id) through the
  [`CollectionDefinition`]({{< relref "storage.md" >}}), and probe items through
  the same substrate the checks use. `object_fields` and `markdown_body` live
  here.

The [`inspect`]({{< relref "../reference/commands.md" >}}) command infers the
layer from its argument, a configured collection name runs the collection
layer, anything else is a path for the raw-source layer, and renders the
evidence as Markdown (default) or JSON. Both layers are built from a few
reusable measurement primitives (`object_fields`, `markdown_body`,
file-metadata). Inspectors have their own registry and per-layer parity test,
mirroring checks, so none ships undocumented. The design rationale
(evidence-not-verdicts, the determinism dividing line, the two-layer split)
lives in the package's own docs, `go doc ./internal/inspect`, next to the
code; see [the reference]({{< relref "../reference/inspectors/_index.md" >}})
for the inspector set.

## Lifecycle of `check`

The data flow per item, end-to-end:

1. **Load config (or take the `--schema` flag).** Discover the `.katalyst/`
   directory from the working directory; failing to find one is a usage error
   (exit 2).
2. **Resolve selectors to items.** No selector means every collection; a
   `<collection>` selector means all its items; `<collection>/<item>` means
   one. Files inside a collection directory that do not match its `pattern`
   are reported as unmatched references (errors).
3. **Read file bytes.** Read errors are reported per-item but don't abort
   the run; we accumulate exit-1 status and continue.
4. **Parse frontmatter.** Errors here (malformed YAML, unterminated fence)
   are per-item failures too. No frontmatter is itself an error.
5. **Resolve the object schema** via the precedence policy above, then
   **strip the `schema:` directive** so user schemas with
   `additionalProperties: false` aren't tripped by katalyst's own metadata.
6. **Build the check list** from the resolved object check plus the
   collection's markdown/filesystem checks.
7. **Run checks.** The object check normalizes Go integer types to JSON
   `float64` before validating (yaml.v3 produces native ints; the JSON
   Schema library expects JSON-shaped numbers).
8. **Format output.** Violations get the `path:line: /pointer: message`
   treatment. Valid items print `path: OK`.

## Lifecycle of `fix`

Much simpler. For each item:

1. Read bytes.
2. Parse to `Document`.
3. If no frontmatter, return verbatim, `fix` never invents structure.
4. Marshal `Meta` with top-level keys sorted alphabetically, yaml.v3
   default block style.
5. Re-assemble: `---\n<sorted yaml>\n---\n<body>`. Body bytes are
   preserved verbatim; one trailing newline is enforced on the file.
6. Compare against the original. If unchanged, do nothing. Otherwise
   atomically rewrite (temp file + rename), or, with `--check`, print the
   path and accumulate exit-1 status.

See [Commands]({{< relref "../reference/commands.md" >}}) for why `fix` is
opinionated and why it refuses to inject missing values.

## Invariants

Properties the codebase should always uphold. Most are protected by
tests; a few are protected only by code review and convention.

1. **Body bytes are sacred.** No command except `fix` modifies them. Even
   `fix` only normalizes trailing whitespace and the leading separator;
   interior body bytes round-trip exactly.
2. **Schema names are stable; paths can move.** The `.katalyst/` config is the
   only place that knows how names map to paths.
3. **The `schema:` directive is katalyst metadata, not user data.** It
   influences resolution but never reaches the validator.
4. **A collection owns its checks; an item belongs to one collection.**
   An item's collection is unambiguous, the one whose directory contains it
, and never decided by glob ordering *across* collections. *Within* a
   collection, an item's checks are the base checks plus those of the first
   [variant]({{< relref "../reference/configuration.md" >}}#variants) whose
   `when` predicates its metadata satisfies; first-match-wins applies only
   among one collection's own variants, never between collections.
5. **Line numbers are file-relative and 1-indexed.** The opening `---`
   fence is line 1, so the first YAML key is typically line 2.
6. **Unmatched is an error, not a warning.** Silent skips hide config
   drift. Escape hatches (`--allow-unmatched`) are deferred to v0.3.
7. **Schema compilation happens once per process per absolute path.** The
   resolver's cache is the bottleneck, not the JSON Schema library.
8. **Config discovery uses symlink-resolved paths on both sides.** On
   macOS, `$TMPDIR` lives under `/var → /private/var`. Without
   `EvalSymlinks` on both root and input, relative-path resolution produces
   garbage.
9. **Production code lives in `internal/`.** Anything exported from `cmd/`
   or a hypothetical `pkg/` should be a deliberate choice with stability
   promises attached.

## Vocabulary

The canonical definitions of frontmatter, metadata, schema, collection,
item, selector, check, and the rest live in the
[glossary]({{< relref "../reference/glossary.md" >}}). Use those terms
consistently in code, docs, and user-facing copy.

## Out of scope (today)

These are absences worth being explicit about; they shape what the
domain currently is *not*:

- **Relations between documents.** A schema can constrain one document at a
  time. No `$ref` to other documents, no foreign keys. Planned.
- **Schema evolution.** No "this field was renamed in v2" migrations.
  Planned.
- **Query.** No "find all docs where year > 1980." Planned.
- **Derived state.** No index, no cache file, no `.katalyst/` directory.
  Every run is stateless.
