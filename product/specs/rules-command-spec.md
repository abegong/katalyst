# `rules` command spec

> **Status: planning.** Adds a read-only `katalyst rules` command that exposes
> the check registry — the catalog `cmd/gendocs` already renders — through the
> CLI and as JSON. Additive to [`cli-spec.md`](./cli-spec.md); no engine change.

## Overview

The catalog of check kinds and their config keys lives in
`internal/checks/registry.go` and is surfaced only by `cmd/gendocs`, which
renders the docs site. `katalyst rules` gives the same catalog a second reader:
a human at the terminal, an editor integration, or a skill that needs the list
of kinds and keys without parsing the docs site or hardcoding a copy.

## Value

A skill or tool that needs to know what `kind:` values exist today calls
`katalyst rules --json` instead of embedding a catalog that drifts the moment a
check is added. The single-source-of-truth policy already enforced for docs
(`registry_test.go` fails if a dispatched kind has no descriptor; CI fails if
the generated tree drifts) extends to every downstream consumer.

## Current State

`checks.Descriptors()` and `checks.Families()` are the canonical record. Each
`Descriptor` already carries everything a reader needs — `Kind`, `Family`,
`Title`, `Summary`, `Fields` (with `Required`, `Default`, `Desc`), and
`ConfigExample` (`internal/checks/registry.go:20`). `cmd/gendocs` consumes them
to write `docs/content/reference/rules/` (`cmd/gendocs/main.go:36`).

The CLI has no way to read this. Anything outside `gendocs` that needs the
catalog — the skill that authors `katalyst.yaml`, an editor completion — must
either scrape the generated Markdown or keep its own list. Both drift.

The proposal floated a contingency: *if `Descriptor` lacks the per-key metadata
gendocs renders, lift it there.* It does not lack it. `Fields []Field` already
holds the full per-key table. No registry change is required; both consumers
read the same struct.

## Design

A new `rules` noun, registered in `NewRootCmd` alongside the others
(`cmd/root.go:25`). It reads `checks.Descriptors()` / `checks.Families()`
directly — the same calls `gendocs` makes — and formats them. No new source of
truth.

### Not a selector command

`rules` sits outside the selector grammar that governs `check`, `item`, and
`collection` ([`cli-spec.md`](./cli-spec.md), Selector grammar). Its argument is
a **check kind** (`object_required_field`), the literal `kind:` token from
config — not a `<collection>/<item>` selector. It is an introspection command
over static engine metadata, so it **loads no project**: it runs in any
directory, with or without a `.katalyst/`. This is the sharp line — `collection`
and `item` answer "what is in *this* project"; `rules` answers "what can the
engine enforce, anywhere".

### `katalyst rules`

The catalog, grouped by family in `Families()` order (objects → markdown →
filesystem). Under each family heading, a `tabwriter` table mirroring
`collection list` (`cmd/collection.go:34`):

```
KIND                      PURPOSE                                   REQUIRED       OPTIONAL
object_required_field     Require that a frontmatter field exists.  field          —
object_number_range       Constrain a numeric field to a min/max.   field          min, max
```

- **REQUIRED** / **OPTIONAL** split `Fields` by `Field.Required`. A check with
  no fields prints `—` in both.
- **PURPOSE** is `Summary` with inline-code backticks stripped, reusing the
  `plain` transform `gendocs` applies for the same reason (`cmd/gendocs/main.go:121`).
- Exit `0`.

### `katalyst rules <kind>`

Detail for one kind, mirroring the generated rule page
(`cmd/gendocs/main.go:91`) in plain terminal text: the `kind:` id, purpose, the
full key table (Field / Required / Default / Meaning), and the `ConfigExample`.

- Unknown kind → usage error, exit `2`, consistent with `collection get` on an
  unknown collection (`cmd/collection.go:69`).
- Exit `0` on a known kind.

### `--json`

On `katalyst rules`, a JSON array of every descriptor in `Descriptors()` order.
On `katalyst rules <kind>`, the single descriptor object. Each descriptor
carries its `family`, so a consumer can group without a second call; family
*ordering* and intro copy stay in `Families()` and are not part of this payload.

This **diverges** from [`cli-spec.md`](./cli-spec.md), which lists `--json`
machine-readable output as out of scope for v0. That decision concerns
machine-readable output of *project data* — check results, item lists, which
need a stable result schema still being designed. The `rules` payload is static
engine metadata, the same structs `gendocs` already serializes to Markdown, and
the downstream consumer (the skill) is the entire reason the command exists.
Withholding JSON here would defeat the purpose; the v0 `--json` deferral does
not apply to introspection metadata.

### Exit codes

| Code | Meaning |
|-----:|---------|
| `0` | Catalog or kind printed |
| `2` | Unknown kind, or usage error |

No `1`: `rules` runs no checks and reads no project, so there is no failure
state, only "printed it" or "you asked for a kind that doesn't exist".

## Open Questions

1. **JSON wire contract — struct tags.** `Descriptor` and `Field` have no
   `json:` tags, so today they would marshal with Go field names
   (`Kind`, `Required`, …). Add explicit snake_case tags
   (`kind`, `required`, `default`, `desc`) so the wire contract is stable
   against Go field renames and reads like the YAML keys it describes?
   Recommend yes — the payload is a published contract for the skill, and
   coupling it to Go identifiers invites a silent break on the next refactor.
   `config.CheckKind` is a string type and needs no tag to marshal as its
   value.

2. **`ConfigExample` in `--json`.** The example is a raw YAML string with
   newlines. Emit it verbatim as a JSON string (a consumer re-parses or
   displays it), or omit it from JSON and keep it human-output only? Recommend
   emit verbatim — it is the one field that lets a consumer show a working
   snippet, and dropping it forces the skill back to scraping docs for examples.

## Test checklist (what the pending tests assert)

Driving `NewRootCmd` per the repo convention (`cmd/helpers_test.go:14`), with no
project on disk — proving `rules` needs none:

`katalyst rules`:
- [ ] every kind in `checks.Descriptors()` appears in output
- [ ] output is grouped under each family title in `Families()` order
- [ ] a check's required keys land in REQUIRED, optional keys in OPTIONAL
- [ ] a no-field check prints `—` for both
- [ ] exit `0`; runs in a directory with no `.katalyst/`

`katalyst rules <kind>`:
- [ ] known kind prints purpose, every field with its required/default/meaning,
      and the config example
- [ ] unknown kind → exit `2`
- [ ] exit `0` on a known kind

`--json`:
- [ ] `rules --json` is a valid JSON array with one entry per descriptor
- [ ] every kind and every field name appears in the JSON
- [ ] `rules <kind> --json` is the single matching descriptor object
- [ ] field keys are stable (per Open Question 1's resolution)

Parity guard:
- [ ] a test asserts the CLI catalog covers exactly `checks.Descriptors()`, so a
      new check surfaces in `rules` without a manual edit — the same parity
      `registry_test.go` enforces for docs
