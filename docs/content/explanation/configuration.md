+++
title = "Configuration rationale"
weight = 50
+++

# Configuration rationale

*Why* Katalyst is configured the way it is. For the precise key-by-key
surface, see the [configuration
reference]({{< relref "../reference/configuration.md" >}}); this page is the
reasoning behind it.

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
  (markdown and filesystem rules, not just an object schema), which the old
  `{paths, schema}` shape could not express cleanly.

The `schema: <name>` shorthand on a collection is the one piece of the old
model that survived — it is sugar for a single leading `object` check.

## Why schema resolution has three tiers

When `check` validates an item against an object schema, it resolves which
schema, highest precedence first:

1. An explicit `--schema <path>` flag — so users can override config ad hoc.
2. An inline `schema: <name>` key in the item's frontmatter — the file's
   author has the most local information about what it is.
3. The collection's configured `object` check — the bulk-association
   default for everything else.

Command-line beats inline beats config because that orders the sources from
most specific intent to most general. Markdown and filesystem checks are not
subject to this precedence: they always come from the collection, since they
describe the item's place in the project rather than its object shape.

## Why unmatched files are errors

A file that sits inside a collection's directory but does not match its
`pattern` is reported as an **error**, not silently skipped. Silent skips
hide config drift — a typo'd pattern or a misfiled document would simply
disappear from validation. Users who want to opt out will get explicit
escape hatches (`--allow-unmatched` and a config knob) rather than implicit
silence; those are deferred until real usage shows the need.

## See also

- [Configuration reference]({{< relref "../reference/configuration.md" >}})
  — the exact keys and defaults.
- [Domain model]({{< relref "domain-model.md" >}}) — how collections, items,
  and the resolver fit together.
