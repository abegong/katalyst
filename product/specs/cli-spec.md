# CLI spec — v0

> **Status: the contract for the v0 rebuild.** This specifies the command
> surface we are building toward; the failing/pending tests derive from it
> line-for-line. It supersedes the current ad-hoc set (`validate`, `fmt`,
> `create`/`read`/`update`/`delete`).
>
> **One open naming call:** the conformance verb is written here as `check`
> (rationale: the engine's primitive is already a `Check` — see
> `internal/checks/`). `validate` remains a candidate; flipping it is a
> mechanical rename and changes nothing structural in this spec.

## Scope of v0

- Backend: the filesystem only — specifically **one project directory**.
- Items: markdown files with YAML frontmatter, **unique names**.
- Collections: a project has one or more **named** collections; each maps to
  a directory of `*.md` files. v0 ships with the single-collection case but
  the collection is always explicit and addressable.
- Reuses the existing engine: config discovery, frontmatter parse with line
  tracking, JSON Schema validation, and the multi-check engine
  (`internal/checks/`).
- Everything in [`connectors.md`](../connectors.md), `diff`, `query`, `infer`,
  `migrate`, and machine-readable output is **out of scope** (see end).

## Concepts (recap)

- **Project** — the directory containing `katalyst.yaml`. The implicit
  top-level scope.
- **Collection** — a named group of items, declared in config, backed by a
  directory. Visible and addressable even when there is only one.
- **Item** — one markdown file. Its **id** is the filename stem, defined as a
  *collection-relative identifier* (today equal to the stem; see
  [`connectors.md`](../connectors.md) for how this grows).

## Selector grammar

A **selector** identifies a target; depth determines scope.

```
                 (omitted)   → the whole project (all collections)
                 <collection>          → one collection (all its items)
                 <collection>/<item>   → one item
```

Rules:

- The **first segment is always a collection**. A bare token with no slash
  (`notes`) is a *collection* selector, never an item.
- An item is addressed only as `<collection>/<item>` (`notes/dune`).
- Blessed verbs (`check`, `fix`) accept a selector at **any depth**, and
  accept **multiple** selectors.
- Noun commands expect a selector at a **fixed depth** (stated per command);
  wrong depth is a usage error (exit 2).
- An unknown collection or item is a usage error (exit 2).

## Config (v0)

Config lives in a `.katalyst/` directory at the project root (see
[`decisions.md`](../decisions.md) D1 and
[`project-layout-spec.md`](./project-layout-spec.md)). Schemas and
collections are each **one named file** — the filename stem is the name.

```
.katalyst/
  config.yaml                 # optional project-level settings
  schemas/note.yaml           # JSON Schema, authored in YAML
  collections/notes.yaml
```

```yaml
# .katalyst/collections/notes.yaml — the name "notes" is the filename stem.
path: notes            # directory, relative to root
pattern: "*.md"        # optional; default "*.md"
schema: note           # a schema name from .katalyst/schemas/; OR use `checks:`
# checks:
#   - kind: object_required_field
#     field: title
```

- Item `notes/dune` resolves to `notes/dune.md` (path + id + extension).
- A file inside a collection's directory that does not match `pattern` is an
  **unmatched reference** → an error under `check` (cf. [`decisions.md`](../decisions.md) D2).
- Discovery (`convention` | `explicit`) and format (`yaml` | `json` | `both`)
  are settable per kind in `.katalyst/config.yaml`, defaulting to convention +
  YAML.

## Command reference

### `katalyst init [--dir <path>]`

Prepare the current directory as a katalyst project: create `.katalyst/`
with empty `schemas/` and `collections/` directories and a commented
`config.yaml`. Writes **no example content**, and refuses to run if a
`.katalyst/` directory already exists. See
[`project-layout-spec.md`](./project-layout-spec.md).

### `katalyst check [selector ...]`

Run the configured checks against the selected items.

- No selector → the whole project (every collection).
- Per item: print `<path>: OK`, or one line per failure as
  `<path>:<line>: /<pointer>: <message>` (line falls back to nearest known
  ancestor when the leaf has no source position).
- Unmatched files in a collection directory are reported as errors.
- **Exit:** `0` all valid · `1` any failure or unmatched · `2` usage/IO.
- Flags: `--schema <path>` (override resolution for this run, all selected
  items).

### `katalyst fix [selector ...]`

Apply the **deterministic, safe** subset of fixes to the selected items.

- v0 scope: frontmatter normalization (sorted top-level keys, block style,
  single trailing newline; body bytes preserved verbatim) plus any fix that
  is unambiguous and lossless.
- **Never invents semantic values** (no placeholder injection) — this is the
  D3 guardrail; auto-remediation that needs a human decision stays out.
- `--check`: CI mode — write nothing, print files that *would* change, exit
  `1` if any would.
- **Exit:** `0` nothing to do / fixed · `1` (`--check`) changes needed · `2`
  usage/IO.

### `katalyst collection list`

List collections: name, directory, item count, schema. Exit `0`/`2`.

### `katalyst collection get <collection>`

Show one collection: path, pattern, schema/checks, item count. Selector
depth = 1. Exit `0`/`2`.

### `katalyst item list <collection>`

List items in a collection: id and check status (`ok` / `n errors`).
Selector depth = 1. Exit `0`/`2`. Supports `--filter`/`--grep`/`--sort`/
`--skip`/`--limit` (MongoDB-`find`-inspired); see the
[commands reference](../../docs/content/reference/commands.md#item-list-query-flags).

### `katalyst item get <collection>/<item> [--frontmatter | --body]`

Print one item. **Default: frontmatter and body.** `--frontmatter` prints
only the parsed frontmatter; `--body` prints only the body. Selector depth =
2. Exit `0` · `2` if not found.

### `katalyst item add <collection>/<item> [key=value ...]`

Create a new item file with the given frontmatter and an empty body.

- Refuses to overwrite an existing item (exit `2`).
- Validates the result before writing (validate-on-write); `--no-validate`
  bypasses.
- `--schema <path>` overrides resolution.
- **Exit:** `0` created · `1` validation failed (nothing written) · `2`
  usage/exists/IO.

### `katalyst item update <collection>/<item> key=value ...`

Set/merge the given keys into an existing item's frontmatter; body
untouched.

- Validates the resulting document before writing; `--no-validate` bypasses.
- `--schema <path>` overrides resolution.
- **Exit:** `0` updated · `1` validation failed (nothing written) · `2`
  usage/not-found/IO.
- Key removal (`--unset`) is **out of scope** for v0.

### `katalyst item delete <collection>/<item> [<collection>/<item> ...]`

Delete one or more items. Exit `0` · `2` if any not found / IO error.

### `katalyst schema list` · `katalyst schema show <name>`

**Carried over unchanged.** Not expanded in v0 (`check`, `diff`, `infer`,
`migrate` under `schema` are out of scope).

### Global

`--version`, `completion <shell>`, and `--help` on every command (Cobra
defaults).

## Exit codes (global)

| Code | Meaning |
|-----:|---------|
| `0` | Success / all valid |
| `1` | Check failures, or `fix --check` found pending changes |
| `2` | Usage error, unknown/missing selector, refuse-overwrite, or IO error |

## `key=value` parsing

Values are parsed as **YAML scalars**: `year=2026` → integer, `draft=true` →
boolean, `title="New title"` → string. This matches the existing
`create`/`update` behavior. Used by `item add` and `item update`.

## Validation-on-write

`add` and `update` validate before writing (default on). On failure: nothing
is written and the command exits `1`. `--no-validate` skips the check.
`--schema <path>` overrides config-based resolution. (Same precedence rules
as `check`.)

## Changes from the current CLI

| Today | v0 |
|---|---|
| `validate` | `check` (open: may stay `validate`) |
| `fmt` | `fix` (broadened scope, D3-guarded) |
| `create` | `item add` (hard rename, no alias) |
| `read` | `item get` (now defaults to frontmatter **and** body) |
| `update` | `item update` |
| `delete` | `item delete` |
| `rules:` list in config | named `collections:` map |
| (none) | `collection list` / `collection get`, `item list` |

## Out of scope for v0 (named so the boundary is explicit)

- `item diff` / `schema diff`, `query`, `aggregate`, `infer`/`profile`,
  `migrate`.
- `schema check`/`validate`, `schema infer`, etc. (the `schema` noun stays at
  `list`/`show`).
- Non-filesystem backends / the connector layer ([`connectors.md`](../connectors.md)).
- Machine-readable output (`--json`), `--unset`, bulk-add, watch mode.
- Multiple-collection ergonomics beyond addressing (it works, but v0 ships
  the single-collection case).

## Test checklist (what the pending tests assert)

Selector resolution:
- [ ] empty selector → all collections
- [ ] `<collection>` → that collection's items only
- [ ] `<collection>/<item>` → one item
- [ ] bare token resolves as collection, never item
- [ ] unknown collection / item → exit 2
- [ ] wrong-depth selector to a noun command → exit 2

`check`:
- [ ] valid item → `OK`, exit 0
- [ ] invalid item → `path:line: /pointer: message`, exit 1
- [ ] missing-required error falls back to ancestor line
- [ ] unmatched file in collection dir → error, exit 1
- [ ] `--schema` override applies to all selected items

`fix`:
- [ ] normalizes frontmatter, preserves body bytes
- [ ] idempotent (second run is a no-op)
- [ ] `--check` writes nothing, exit 1 when changes pending, exit 0 when clean
- [ ] never injects values for missing required keys

`item` CRUD:
- [ ] `add` writes frontmatter + empty body; YAML-scalar typing of values
- [ ] `add` refuses existing item (exit 2)
- [ ] `add` validation failure writes nothing (exit 1)
- [ ] `get` default prints frontmatter + body; `--frontmatter` / `--body` narrow
- [ ] `update` merges keys, leaves body untouched, validates result
- [ ] `delete` removes one and many; missing → exit 2

`collection`:
- [ ] `collection list` shows name, path, count, schema
- [ ] `collection get` shows one collection's detail
- [ ] `item list <collection>` shows ids + status

Config:
- [ ] named `collections:` loads; item id → file path resolution is correct
- [ ] reverse resolution (`add notes/dune` → `notes/dune.md`) is correct
- [ ] every collection's `schema` references a known `schemas:` entry
