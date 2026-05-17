# Decisions

Resolved design decisions. Each entry records *what* was decided, *why*,
and *when* (by commit or version). Open questions live in
`decisions-to-make.md`.

## D1 — Config file format and location (v0.1)

**Decision.** The config file is `katabridge.yaml` at the repo root.
Discovery: walk up from the working directory looking for the nearest
ancestor that contains a `katabridge.yaml`. That directory becomes the
"repo root" for path resolution.

**Shape.**

```yaml
schemas:
  book:   ./schemas/book.json
  person: ./schemas/person.json

rules:
  - paths: "notes/books/**/*.md"
    schema: book
  - paths: "notes/people/**/*.md"
    schema: person
```

- `schemas` is a name → file-path map. Names are the public handle used
  by other commands (`schema show book`, inline `schema: book` keys,
  etc.). Paths are resolved relative to the config file.
- `rules` is an ordered list of `{paths, schema}` pairs. The first
  matching rule wins. Globs use Go's [doublestar][ds] syntax (so `**`
  works, unlike `path/filepath.Match`).

**Why.** YAML matches what the user already writes in frontmatter, so
there's no second format to learn. A nearest-ancestor lookup mirrors
`.git`, `.editorconfig`, and `go.mod` — familiar and predictable. We
keep `schemas` and `rules` separate so the same schema can apply to
multiple path patterns without duplication.

[ds]: https://github.com/bmatcuk/doublestar

## D2 — Schema association precedence (v0.1)

**Decision.** Highest to lowest precedence:

1. Explicit `--schema <path>` flag on the command line.
2. Inline `schema:` key inside the file's frontmatter (value is a
   schema *name* from the config).
3. First matching `rules` entry in the config.

If none of these resolve a schema for a given file, that file is treated
as an error in `validate` (exit code 1 if any such file is found). We
chose error-not-warning because silent skips hide config drift; users
who want to opt out can add a catch-all rule mapping to a permissive
schema, or pass `--allow-unmatched` (future flag, not in v0.1).

**Why.** Command-line wins so users can override config ad hoc. Inline
beats glob rules because the file's author has the most local
information about what it is. Glob rules are the bulk-association
mechanism for everything else.

## D3 — `validate --fix` is deferred (v0.2 → v0.3)

**Decision.** The original v0.2 idea of `validate --fix` adding
"sentinel values" for missing required keys is shelved. It will be
revisited in v0.3, possibly under a different name (`patch`?
`scaffold`?), with explicit user opt-in per field.

**Why.** Silently injecting placeholder values is hostile: it can mask
real problems, create merge conflicts, and produce documents that pass
schema validation while being semantically wrong. We'd rather ship
nothing than ship a bad `--fix`. A safer design (interactive, or
constrained to specific operations like "fill default from schema's
`default:` keyword") deserves its own discussion.

## D4 — `fmt` is opinionated (v0.2)

**Decision.** `katabridge fmt` normalizes frontmatter aggressively:

- Keys sorted alphabetically.
- yaml.v3 default block style (no flow-style maps/sequences in output).
- Strings unquoted where safe, double-quoted otherwise.
- Exactly one trailing newline in the file.
- Body bytes preserved verbatim.

There are no flags. Users who want a different style don't run `fmt`.

**Why.** `gofmt`/`black`/`rustfmt` taught us that the value of a
formatter comes from there being one obvious answer. Configurability
re-creates the bikeshed. The body is preserved so `fmt` is safe to run
across an entire repo without touching prose.

**Trade-off.** Comments inside the frontmatter are not preserved. That
is by design (frontmatter is structured data, not prose). If this hurts
in practice we'll revisit.
