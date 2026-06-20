# `item list` query spec — DRAFT

> **Status: draft / in flight.** Extends `katalyst item list <collection>`
> (see [`cli-spec.md`](./cli-spec.md)) with MongoDB-`find`-inspired
> **filtering**, **grep**, **sort**, **skip**, and **limit**. No other
> `list` command (`collection list`, `schema list`) is touched.
>
> Design decisions are settled; this is ready to drive implementation.
> Delete this file when the work lands (per `product/specs/` convention).

## Motivation

`item list` today prints every item in a collection as `id <tab> status`,
in id order, with no way to narrow, search, reorder, or cap the output. As
collections grow this becomes unusable for "show me the draft books from
after 1965, newest first". MongoDB's
`find(filter, projection).sort().skip().limit()` is the reference model; we
adopt its *pipeline* (filter → sort → skip → limit) with a CLI-ergonomic
surface instead of JSON query documents.

## Command surface

```
katalyst item list <collection>
  [--filter EXPR ]...        # repeatable; ANDed together
  [--grep PATTERN ]...       # repeatable; ANDed together
  [--grep-in REGION]         # all (default) | body | frontmatter
  [-i | --ignore-case]       # case-insensitive --grep
  [--sort KEY ]...           # repeatable; KEY or -KEY (desc); comma-joinable
  [--skip N]
  [--limit N]
  [--on-type-mismatch MODE]  # skip (default) | error   -- overrides config
  [--sort-missing MODE]      # last (default) | lowest  -- overrides config
```

Selector depth stays **1** (`<collection>`); wrong depth is a usage error
(exit 2), unchanged. Output columns are unchanged: `id <tab> status`.
(Projection — choosing which frontmatter fields to show — is **out of
scope**; see below.)

## Evaluation pipeline

The flags compose as a fixed pipeline, mirroring Mongo:

1. **Enumerate** the collection's items (current `Items`).
2. **Filter** — keep items matching *every* `--filter` expression (AND).
3. **Grep** — keep items matching *every* `--grep` pattern (AND), within the
   `--grep-in` region.
4. **Sort** — order by the `--sort` keys (default: `id` ascending).
5. **Skip** — drop the first `--skip N` items.
6. **Limit** — keep at most `--limit N` items.

Steps 2–3 are independent predicates; their order relative to each other
doesn't affect the result. Steps 4–6 are strictly ordered.

Every item's frontmatter is parsed once up front (today's `item list` does
not parse frontmatter — this is the main new per-item cost). The raw file
bytes are also kept for `--grep`.

## `--filter EXPR` — field predicates

Repeatable; each occurrence is one predicate; all are ANDed (Mongo's
implicit top-level AND). There is **no `OR`** in this iteration (Mongo's
`$or` is a future addition).

### Grammar

```
EXPR := FIELD OP VALUE          # comparison / regex / membership
      | FIELD                   # existence  ("key is present")
      | ! FIELD                 # absence    ("key is absent")
```

`FIELD` addresses a frontmatter key, with **dot notation** for nested maps
(`author.name`). Deeper structure access (array indexing like `tags.0`) is
out of scope; an array-valued field is addressed by its bare name and
handled by `in` (membership). `FIELD` is matched against the parsed
frontmatter `Meta` only — not the body.

### Operators

| Op    | Meaning                  | Mongo analogue        |
|-------|--------------------------|-----------------------|
| `=`   | equals                   | implicit / `$eq`      |
| `!=`  | not equals               | `$ne`                 |
| `>`   | greater than             | `$gt`                 |
| `>=`  | greater than or equal    | `$gte`                |
| `<`   | less than                | `$lt`                 |
| `<=`  | less than or equal       | `$lte`                |
| `=~`  | matches regex            | `$regex`              |
| `=`   with comma RHS      | equals any of         | `$in`                 |
| `!=`  with comma RHS      | equals none of        | `$nin`                |
| (bare `FIELD`)            | key exists            | `$exists: true`       |
| (`!FIELD`)                | key absent            | `$exists: false`      |

Operators are recognized by scanning the expression for the **longest**
operator substring first (`>=`/`<=`/`!=`/`=~` before `>`/`<`/`=`), so field
names never collide with operators.

### Value typing

`VALUE` is parsed as a **YAML scalar**, identical to `item add`/`update`
assignments (`parseAssignment`, README "key=value parsing"):
`year>=1965` → integer compare, `draft=true` → boolean, `title=Dune` →
string. This keeps one typing rule across the CLI.

Comparison semantics:

- **Numbers** compare numerically; **strings** lexicographically; booleans
  by `false < true`.
- A **type mismatch** between the field's value and `VALUE` (e.g. `>` on a
  boolean, or comparing a number field to a string literal) is governed by
  the **`filterTypeMismatch`** setting (see [Configuration](#configuration)).
  Default `skip`: the item simply does not match, so a stray non-conforming
  item never aborts the listing. Set to `error` to make a mismatch a usage
  error (exit 2). `--on-type-mismatch` overrides it per invocation.
- `=~` coerces the field value to its canonical string form and tests it
  against a **Go `regexp`** pattern. `-i` / `--ignore-case` applies to
  `--grep` **only**, not to `=~`; for a case-insensitive filter regex use an
  inline `(?i)` flag (`title=~(?i)^the`).
- `in` / `nin` (comma RHS): for a scalar field, true iff the value is (not)
  among the listed values; for an **array** field, true iff the array
  shares (shares no) element with the list. Each comma-separated token is
  YAML-typed individually.

### Examples

```bash
katalyst item list books --filter 'year>=1965'
katalyst item list books --filter 'status=draft' --filter 'year>=1965'
katalyst item list books --filter 'tags=sci-fi,classic'   # has either tag
katalyst item list books --filter 'title=~^The'
katalyst item list books --filter 'isbn'                   # has an isbn key
katalyst item list books --filter '!isbn'                  # missing isbn
```

### Parse failures

An item whose frontmatter fails to parse is treated as having **empty
`Meta`**: it matches `!FIELD` (absence) and fails all positive field
predicates. It can still match `--grep` against its raw bytes.

## `--grep PATTERN` — text search

Repeatable; each pattern is a **Go `regexp`**; all must match (AND). The
search region is set by `--grep-in`:

| `--grep-in`   | Region searched                                  |
|---------------|--------------------------------------------------|
| `all` (default) | the entire raw file (frontmatter + body)       |
| `body`        | the markdown body only (after the closing fence) |
| `frontmatter` | the raw frontmatter block only                   |

`-i` / `--ignore-case` makes all `--grep` patterns case-insensitive
(equivalent to prefixing each with `(?i)`). `--grep` and `--filter` are
ANDed together; an item must satisfy both.

```bash
katalyst item list notes --grep 'TODO'
katalyst item list notes --grep 'TODO' --grep-in body -i
```

## `--sort KEY` — ordering

Repeatable, and a single occurrence may list multiple comma-separated keys;
keys apply in order (first is primary). A leading `-` means descending:

```bash
katalyst item list books --sort -year            # newest first
katalyst item list books --sort -year,title      # then title A→Z
katalyst item list books --sort -year --sort title
```

- `KEY` is `id`, `status`, or any frontmatter field (dot notation).
  `status` sorts by error count (`ok` = 0).
- **Default** (no `--sort`): `id` ascending — today's order.
- Comparison uses the same number/string/bool rules as `--filter`. Across
  mixed types, a stable type ordering applies (numbers < strings, etc.).
- **Missing** fields are placed per the **`sortMissing`** setting (see
  [Configuration](#configuration)). Default `last`: missing-field items go to
  the end in **both** directions. `lowest`: a missing field is treated as
  less than any present value (so first under ascending, last under
  descending). `--sort-missing` overrides it per invocation.
- The sort is **stable**; ties (including equal/missing keys) break by `id`
  ascending.

## `--skip N` and `--limit N` — pagination

Applied after sorting, in that order (Mongo semantics).

- `--skip N` (N ≥ 0) drops the first `N` results. `0`/absent = drop none.
- `--limit N` (N ≥ 1) keeps at most `N` results. `0`/absent = no cap
  (matching Mongo's `limit 0`). Negative `--limit` is a usage error.

```bash
katalyst item list books --sort -year --limit 10        # 10 newest
katalyst item list books --sort -year --skip 10 --limit 10   # next page
```

## Configuration

Two behaviors have configurable defaults. They live in a `query:` block,
which may appear in **both** the project config and an individual
collection's config:

```yaml
# .katalyst/config.yaml — project-wide default
query:
  filterTypeMismatch: skip   # skip (default) | error
  sortMissing: last          # last (default) | lowest
```

```yaml
# .katalyst/collections/books.yaml — overrides the project default for books
path: notes/books
schema: book
query:
  filterTypeMismatch: error
```

Resolution precedence, highest first:

1. CLI flag (`--on-type-mismatch`, `--sort-missing`) — this invocation only.
2. The selected collection's `query:` block.
3. The project `query:` block in `.katalyst/config.yaml`.
4. Built-in default (`skip` / `last`).

A `query:` block may set either key independently; an unset key falls
through to the next level. Schema files (`.katalyst/schemas/*.yaml`) stay
pure JSON Schema — they hold **no** query settings.

## Exit codes

Unchanged from `item list` today:

| Code | Meaning                                             |
|-----:|-----------------------------------------------------|
| `0`  | Listed successfully — **including an empty result** |
| `2`  | Usage error: bad selector/depth, unknown collection, malformed `--filter`/`--sort` expression, invalid regex, negative `--limit`/`--skip`, or IO error |

A filter/grep matching nothing is a successful empty list (exit 0), **not**
grep's "exit 1 on no match".

## Out of scope (named so the boundary is explicit)

- **Projection** (`--fields a,b` / Mongo projection) — choosing which
  frontmatter fields to display as columns. Output stays `id <tab> status`.
  Strong candidate for a follow-up.
- **`OR` / nested boolean logic** (`$or`, `$and`, `$not`), array index
  paths (`tags.0`), `$size`, `$type`, and other Mongo operators beyond the
  table above.
- Machine-readable output (`--json`), watch mode.
- Extending filter/grep/sort to `collection list` or `schema list`.

## Implementation notes (non-normative)

- New code lives in `cmd/` next to `newItemListCmd` (`cmd/item.go:41`);
  the predicate/sort engine is a good candidate for a small, table-tested
  helper (possibly `internal/query/`) so it can be unit-tested without the
  cobra layer. TDD per `AGENTS.md`.
- Reuse `frontmatter.Parse` for `Meta`, body, and the raw frontmatter
  region; reuse `parseAssignment`'s YAML-scalar typing for filter/sort
  values; reuse `itemStatus` for the `status` sort key.
- Filtering forces a frontmatter parse per item — acceptable for v0
  (collections are local files).

## Test checklist (to drive the pending tests)

`--filter`:
- [ ] `=`, `!=`, `>`, `>=`, `<`, `<=` with numeric and string values
- [ ] `=~` regex match; invalid regex → exit 2
- [ ] comma RHS → `in` (scalar and array fields); `!=` comma → `nin`
- [ ] bare `FIELD` (exists) and `!FIELD` (absent)
- [ ] dot-notation nested field
- [ ] multiple `--filter` are ANDed
- [ ] type mismatch, default `skip` → no match (no error)
- [ ] type mismatch, `filterTypeMismatch: error` / `--on-type-mismatch error` → exit 2
- [ ] unparseable frontmatter → matches `!FIELD`, fails positive predicates

`--grep`:
- [ ] matches across whole file by default
- [ ] `--grep-in body` / `frontmatter` narrows the region
- [ ] `-i` makes patterns case-insensitive
- [ ] multiple `--grep` are ANDed; combine with `--filter`
- [ ] invalid regex → exit 2

`--sort` / `--skip` / `--limit`:
- [ ] ascending default by id; `-KEY` descending
- [ ] multi-key precedence (comma and repeated forms)
- [ ] sort by `status` and by frontmatter field
- [ ] missing field, default `last`; `sortMissing: lowest` / `--sort-missing lowest` flips placement
- [ ] ties break by id; stable
- [ ] `--skip`/`--limit` applied after sort, in order
- [ ] empty result → exit 0
- [ ] negative `--limit`/`--skip` → exit 2

Configuration:
- [ ] project `query:` default applies
- [ ] collection `query:` overrides project default
- [ ] CLI flag overrides both; unset keys fall through each level
- [ ] schema files carry no query settings
