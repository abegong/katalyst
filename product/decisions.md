# Decisions

Resolved design decisions. Each entry records *what* was decided, *why*,
and *when* (by commit or version). Open questions live in
`decisions-to-make.md`.

## D1 — Project layout and config location (v0.1; revised)

**Decision.** A project is marked by a `.katalyst/` directory. Discovery:
walk up from the working directory looking for the nearest ancestor that
contains a `.katalyst/` directory; that ancestor is the **project root**
for path resolution. Schemas and collections are each defined one named
file per definition:

```
<root>/.katalyst/
  config.yaml            # optional project-level settings
  schemas/<name>.yaml    # JSON Schema, authored in YAML
  collections/<name>.yaml
```

A schema's or collection's **name** is its filename stem
(`.katalyst/schemas/book.yaml` → `book`). A collection file holds the
same fields a `collections:` map entry held (`path`, `pattern`, `schema`,
`checks`), minus the name.

**Config options.** `.katalyst/config.yaml` is optional; every key
defaults. It configures, **per kind**, how definitions are discovered and
in what format:

```yaml
schemas:
  discovery: convention   # convention (scan the dir) | explicit (use defs)
  format: yaml            # yaml | json | both
  # defs:                 # name → path, only under discovery: explicit
collections:
  discovery: convention
  format: yaml
  # defs:                 # name → definition, only under discovery: explicit
```

**Why.** A single hideable `.katalyst/` directory groups everything
katalyst owns (mirroring `.git`, `.github`), instead of scattering a root
`katalyst.yaml` and a sibling `schemas/` among the user's documents. One
file per schema/collection diffs and reorganizes better than one growing
map, and makes the name-is-the-filename convention obvious. YAML matches
what users already write in frontmatter; JSON Schema is just a data shape,
so a YAML schema compiles through the same path a `.json` file does. The
nearest-ancestor lookup is unchanged in spirit — only the marker moved
from a file to a directory.

**Supersedes.** The original D1 put a single `katalyst.yaml` at the repo
root with top-level `schemas:` and `rules:`/`collections:` maps. The
explicit `defs` map preserves that style for users who want one declared
list. See `product/specs/project-layout-spec.md` for the full rationale
and rejected alternatives.

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

**Decision.** `katalyst fmt` normalizes frontmatter aggressively:

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
