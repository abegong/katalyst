# internal/config

Loads `katalyst.yaml`, resolves named schemas and collections, and decides
which schema applies to a given item. This is the orchestration hub: the
`check` lifecycle is driven from here.

For the precise key-by-key surface, see the
[configuration reference](../../docs/content/reference/configuration.md). This
file is the *why* and the conceptual model behind it.

## Why a single `katalyst.yaml` at the repo root

The config file is `katalyst.yaml`, discovered by walking **up** from the
working directory to the nearest ancestor that contains it. That directory
becomes the repo root for all path resolution.

YAML matches what users already write in frontmatter, so there is no second
format to learn. A nearest-ancestor lookup mirrors `.git`, `.editorconfig`,
and `go.mod` — familiar and predictable. Discovery resolves symlinks on both
the root and the input path, because on macOS `$TMPDIR` lives behind
`/var → /private/var` and relative-path resolution would otherwise produce
garbage.

Whether to also accept JSON config is deferred until someone asks; YAML is
the only supported format today.

## Why named collections

Configuration is two maps:

```yaml
schemas:
  book: ./schemas/book.json
collections:
  books:
    path: notes/books
    schema: book
```

`schemas` maps a **name** to a file path; the name is the stable public
handle (used by `schema show`, by inline `schema:` keys, and by a
collection's `schema:` shorthand) while the path is free to move. A
**collection** is a named directory with a filename `pattern` and the checks
its items must pass. Keeping `schemas` and `collections` separate lets one
schema back many collections without duplication.

### Why this replaced the old anonymous `rules:` list

Earlier versions used a flat, ordered `rules:` list of `{paths: <glob>,
schema: <name>}` pairs, where the *first matching glob wins*. Named
collections replaced it for three reasons:

- **Identity.** A collection has a name, so commands can address it
  (`check books`, `item list books`). An anonymous glob rule cannot be
  named or selected.
- **No precedence puzzles.** Glob ordering made the active rule for a file
  depend on the order of unrelated entries. A file now belongs to exactly
  one collection — the one whose directory contains it — so there is no
  "first match wins" to reason about.
- **More than schemas.** A collection carries a whole `checks:` list
  (markdown and filesystem checks, not just an object schema), which the old
  `{paths, schema}` shape could not express cleanly.

The `schema: <name>` shorthand on a collection is the one piece of the old
model that survived — it is sugar for a single leading `object` check.

## Why schema resolution has three tiers

When `check` validates an item against an object schema, it resolves which
schema, highest precedence first:

| # | Source                          | When it wins |
|---|---------------------------------|--------------|
| 1 | `--schema <path>` flag          | Always, for every item in the invocation |
| 2 | Inline `schema: <name>` in FM   | When (1) absent and `Meta["schema"]` is a known name |
| 3 | The collection's object check   | When (1) and (2) absent |
| — | None                            | The item simply runs no object check |

Command-line beats inline beats config because that orders the sources from
most specific intent to most general — the file's author has the most local
information about what it is; the collection is the bulk-association default.
Markdown and filesystem checks are *not* subject to this precedence: they
always come from the collection, since they describe the item's place in the
project rather than its object shape.

## Why unmatched files are errors

A file that sits inside a collection's directory but does not match its
`pattern` is reported as an **error**, not silently skipped. Silent skips
hide config drift — a typo'd pattern or a misfiled document would simply
disappear from validation. Users who want to opt out will get explicit
escape hatches (`--allow-unmatched` and a config knob) rather than implicit
silence; those are deferred until real usage shows the need.

## Entities

### Config

The single source of truth for "what schemas exist and what each collection
checks." Lives at `katalyst.yaml` in the **repo root**, discovered by walking
upward from the working directory. A `config.Config` has:

| Field         | Meaning |
|---------------|---------|
| `Root`        | Absolute, symlink-resolved repo-root directory |
| `Schemas`     | Schema name → absolute file path |
| `Collections` | The named collections, in name order |

Validated at load time: every collection's object schema must reference a
known entry in `Schemas`, and a collection must configure at least one check
(via `schema:` shorthand or an explicit `checks:` list).

### Collection

A **named** group of items backed by a directory. It is the unit you select
on the command line and the unit that owns a set of checks:

```yaml
collections:
  books:
    path: notes/books   # directory, relative to the repo root
    pattern: "*.md"      # filename glob; default "*.md"
    schema: book         # shorthand for a single object check
    checks:              # any additional checks
      - kind: markdown_title_matches_h1
```

`path` defaults to the collection name; `pattern` defaults to `*.md`. The
`schema:` shorthand is sugar for a leading `object` check.

### Item

A single member of a collection: one file matching the collection's
`pattern`. Its **id** is the filename stem (`notes/books/dune.md` → `dune`).
On the command line an item is addressed by a **selector**.

### Selector

How commands name what to operate on. Three shapes, broad to narrow:

| Selector | Scope |
|---|---|
| *(none)* | the whole project — every collection |
| `<collection>` | one collection — all its items |
| `<collection>/<item>` | a single item |

Selectors are shared by `check`, `fix`, and the `item` subcommands.

### Schema

A JSON Schema (draft 2020-12 by default) describing the legal shape of a
document's `Meta`. A schema has two identities: a **path** on disk and a
**name** registered in `katalyst.yaml`. The name is the stable public handle;
paths can change. `--schema <path>` bypasses the name layer entirely.

### Schema directive (`schema:` in frontmatter)

A per-document opt-in to a specific schema. Treated as **metadata about
katalyst itself, not user data**: the resolver reads it to choose a schema,
then strips it from `Meta` before passing to the validator. This matters when
a schema uses `additionalProperties: false` — the document can still "name
itself" without the directive becoming a validation violation.

### Resolver

Not a persistent entity — a per-`check`-invocation object. Owns the
object-schema selection policy (the three-tier precedence above) and a
compiled-schema cache keyed by absolute path. The cache makes "check 10,000
files against the same schema" cost one compile.

## Lifecycle of `check`

The data flow per item, end-to-end:

1. **Load config (or take the `--schema` flag).** Discover `katalyst.yaml`
   from the working directory; failing to find one is a usage error
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

## Invariants

1. **Schema names are stable; paths can move.** `katalyst.yaml` is the only
   place that knows how names map to paths.
2. **The `schema:` directive is katalyst metadata, not user data.** It
   influences resolution but never reaches the validator.
3. **A collection owns its checks; an item belongs to one collection.**
   There is no glob-ordering "first match wins" — an item's checks are the
   checks of the collection whose directory contains it.
4. **Unmatched is an error, not a warning.** Silent skips hide config drift.
   Escape hatches (`--allow-unmatched`) are deferred.
5. **Config discovery uses symlink-resolved paths on both sides.** On macOS,
   `$TMPDIR` lives under `/var → /private/var`. Without `EvalSymlinks` on
   both root and input, relative-path resolution produces garbage.
6. **Production code lives in `internal/`.** Anything exported from `cmd/` or
   a hypothetical `pkg/` should be a deliberate choice with stability
   promises attached.

## Out of scope (today)

Absences worth being explicit about; they shape what the domain currently
is *not*:

- **Relations between documents.** A schema can constrain one document at a
  time. No `$ref` to other documents, no foreign keys. Planned.
- **Schema evolution.** No "this field was renamed in v2" migrations.
  Planned.
- **Query.** No "find all docs where year > 1980." Planned.
- **Derived state.** No index, no cache file, no `.katalyst/` directory.
  Every run is stateless.
