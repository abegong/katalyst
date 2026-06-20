# katalyst

Define and enforce schemas for structured metadata (frontmatter) on markdown
files. Inspired by [JSON Schema][js] and the [MongoDB validation API][mv].

[js]: https://json-schema.org/
[mv]: https://www.mongodb.com/docs/manual/core/schema-validation/

> **Status:** v0. The command surface is `init`, `check`, `fix`,
> `collection list/get`, `item list/get/add/update/delete`, and
> `schema list/show` (see [`docs/reference/commands.md`](docs/reference/commands.md)).
> See the deep-dive pages under
> [`docs/deep-dives/`](docs/deep-dives/) for the design rationale.

## Install

```
go install github.com/katabase-ai/katalyst@latest
```

Or from source:

```
git clone https://github.com/katabase-ai/katalyst
cd katalyst
make build  # produces ./bin/katalyst
```

## Quickstart

```bash
mkdir my-notes && cd my-notes
katalyst init                  # prepares a .katalyst/ project directory
katalyst check                 # check every item in the project
```

The config is picked up automatically: every command discovers the nearest
`.katalyst/` directory walking up from the working directory, then resolves
**selectors** against the collections it declares.

## Selectors

Commands address targets by **selector**, where depth determines scope:

```
(omitted)              the whole project (all collections)
<collection>           one collection (all its items)
<collection>/<item>    one item
```

The first segment is always a collection; a bare token (`notes`) is a
collection, never an item. `check` and `fix` accept selectors at any depth
and accept several at once; the noun commands (`collection`, `item`)
expect a fixed depth.

## Configuring

A `.katalyst/` directory at your project root holds the config. Schemas and
collections are each one named file, discovered by filename:

```
.katalyst/
  config.yaml                  # optional project-level settings
  schemas/
    book.yaml                  # JSON Schema, authored in YAML
    person.yaml
  collections/
    books.yaml
    people.yaml
```

```yaml
# .katalyst/collections/books.yaml — the name "books" is the filename stem.
path: notes/books     # directory, relative to the project root
pattern: "*.md"        # optional; default "*.md"
schema: book           # a schema name from .katalyst/schemas/, OR use checks:
```

The item `books/dune` resolves to `notes/books/dune.md` (path + id +
extension). A file inside a collection's directory that does not match its
`pattern` is reported as an unmatched reference (an error under `check`).

Discovery and file format are settable per kind in `.katalyst/config.yaml`
(defaults shown):

```yaml
schemas:
  discovery: convention   # convention (scan the dir) | explicit (list under defs)
  format: yaml            # yaml | json | both
collections:
  discovery: convention
  format: yaml
```

Object-schema resolution precedence, highest first:

1. `--schema <path>` on the command line (overrides everything).
2. An inline `schema: <name>` key inside an item's frontmatter (the name
   refers to an entry in `schemas:` above; the directive itself is
   stripped before validation, so you can have strict
   `additionalProperties: false` schemas without listing it).
3. The collection's configured object checks.

## Commands

### `katalyst check [selector ...]`

Run the configured checks against the selected items (the whole project
when no selector is given).

```
$ katalyst check books/dune
notes/books/dune.md: OK

$ katalyst check books/bad
notes/books/bad.md:3: /year: got string, want integer
notes/books/bad.md: /: missing property 'isbn'
```

Errors include `:line` when the source position is known; missing-required
errors fall back to the nearest known ancestor line. Files in a collection
directory that don't match its `pattern` are reported as unmatched.

Exit codes:

| Code | Meaning                                    |
|-----:|--------------------------------------------|
| `0`  | All items valid                            |
| `1`  | One or more failures, or unmatched files   |
| `2`  | Usage error, unknown selector, or IO error |

`--schema <path>` overrides object-schema resolution for every selected
item.

### `katalyst fix [selector ...]`

Apply the deterministic, safe subset of fixes: normalize frontmatter
(top-level keys sorted alphabetically, default block style, exactly one
trailing newline). Body preserved verbatim. `fix` never invents semantic
values for missing keys — see
[`internal/frontmatter/README.md`](internal/frontmatter/README.md).

```
katalyst fix                       # rewrites the whole project in place
katalyst fix --check               # CI mode: no writes, exit 1 if any change
katalyst fix books books/dune      # selected scopes only
```

### `katalyst collection list` / `katalyst collection get <collection>`

```
$ katalyst collection list
NAME   DIRECTORY    ITEMS  SCHEMA
books  notes/books  1      book
people notes/people 0      person

$ katalyst collection get books
name:    books
path:    notes/books
pattern: *.md
schema:  book
items:   1
checks:  object
```

### `katalyst item ...`

Item-level commands, all addressed by `<collection>/<item>` selector
(except `item list`, which takes a `<collection>`):

```bash
katalyst item list books                              # ids + check status
katalyst item get books/dune                          # frontmatter + body
katalyst item get books/dune --frontmatter            # or --body
katalyst item add books/dune title="Dune" year=1965   # create
katalyst item update books/dune year=1965             # merge keys
katalyst item delete books/dune [books/other ...]     # remove
```

`item list` filters, searches, sorts, and paginates with a MongoDB
`find`-inspired pipeline (filter → grep → sort → skip → limit):

```bash
katalyst item list books --filter 'year>=1965' --filter 'status=draft'
katalyst item list books --filter 'tags=sci-fi' --filter '!isbn'  # in / absent
katalyst item list books --grep TODO --grep-in body -i
katalyst item list books --sort -year --limit 10                  # 10 newest
```

`--filter` is `field OP value` (`= != > >= < <= =~`; comma RHS is `in`; a
bare `field` tests existence, `!field` absence; dot paths reach nested
keys). Repeated `--filter`/`--grep` are ANDed. See the
[commands reference](docs/content/reference/commands.md) for the full flag
list and the `query:` config defaults.

`key=value` values are parsed as YAML scalars (`year=2026` → integer,
`draft=true` → boolean, `title="New title"` → string).

Validation behavior for write commands (`add`, `update`):

- Default is strict validation before write; nothing is written on failure.
- `--no-validate` bypasses the check.
- `--schema` overrides config-based schema resolution (same precedence
  rules as `check`).
- `add` refuses to overwrite an existing item.

### `katalyst schema list` / `katalyst schema show <name>`

```
$ katalyst schema list
book    .katalyst/schemas/book.yaml
person  .katalyst/schemas/person.yaml

$ katalyst schema show book
type: object
required: [title, year]
...
```

### `katalyst init [--dir <path>]`

Prepare the current directory as a katalyst project: create `.katalyst/`
with empty `schemas/` and `collections/` directories and a commented
`config.yaml`. Writes no example content, and refuses to run if a
`.katalyst/` directory already exists.

### Shell completion

Cobra provides scripts for bash, zsh, fish, and powershell:

```bash
source <(katalyst completion zsh)
katalyst completion zsh > "${fpath[1]}/_katalyst"   # persistent
```

## Documentation site (Hugo)

User-facing docs live under `docs/` — its own Hugo site with a separate
`docs/go.mod`, so the application module's `go mod tidy` never touches the
theme. Content is `docs/content/`; the site config is `docs/hugo.yaml`.
The site uses the [Hugo Book theme](https://github.com/alex-shpak/hugo-book)
as a Hugo Module (no npm toolchain required).

Hugo Book requires the Hugo **extended** build for SCSS support.

If an extended `hugo` is on your `PATH`, `make` will use it directly.
Otherwise the docs targets automatically fall back to:

```bash
go run -tags extended github.com/gohugoio/hugo@latest
```

Optional local install:

```bash
go install github.com/gohugoio/hugo@latest
```

Run docs locally:

```bash
make docs-serve
```

On first run, Hugo may download/update the theme module dependency.

Build static docs:

```bash
make docs-build
```

## Development

This project follows TDD. Tests live next to the code they exercise, in
`*_test.go` files using only the standard library `testing` package.

```
make test    # go test ./...
make vet     # go vet ./...
make all     # vet, test, build
```

Layout:

```
cmd/                  cobra commands (root, init, check, fix, collection, item, schema)
internal/config       .katalyst/ loader + named collection/schema resolution
internal/project      collection/item domain layer: selectors, item enumeration
internal/frontmatter  YAML frontmatter parser + formatter, with line tracking
internal/validator    JSON Schema validation (wraps santhosh-tekuri/jsonschema)
cmd/gendocs           generates docs/reference/rules/ from the checks registry
docs/                 published Hugo site (users + contributors)
product/specs/        in-flight specs only (deleted when their work lands)
```

See [`AGENTS.md`](AGENTS.md) for conventions on tests, fixtures, and code style.
