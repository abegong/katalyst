# Error message standards

> **Status: implementing.** Style guide defined and applied; the durable
> convention now lives in [`cmd/AGENTS.md`](../../cmd/AGENTS.md), with shared
> helpers (`cmd/usage.go`) and a root `FlagErrorFunc`. `make all` green.
> Graduation (deleting this spec) waits on merge. Defines one grammar for the
> CLI's user-facing error and diagnostic messages and normalizes every site in
> `cmd/`.

## Overview

Katalyst's errors are written ad hoc: some lead with a command name
(`inspect:`), some with a flag (`--grep:`), some with the offending
identifier (`unknown collection %q`), and some are Cobra's own
(`accepts 1 arg(s), received 0`). Quoting, capitalization, and whether a
message suggests a next step all vary. None of this is wrong in isolation, but
the surface reads as if written by several people, and Cobra's defaults clash
with the project's voice.

This spec defines a single grammar for messages, the exit code each maps to,
and the stream each goes to — then rewrites every message to match.

## Value

- **Predictable for users.** One shape — subject, problem, optional hint —
  so an error is scannable and the fix is obvious.
- **Predictable for tooling.** Diagnostics keep their machine-parseable
  `path:line: /pointer: message` form; usage errors are consistent enough to
  grep.
- **A standard for new code.** Once the grammar lives in a doc and a helper,
  new commands inherit it instead of reinventing a phrasing.

## Current State

The full inventory is in the [appendix](#appendix-message-inventory). The
inconsistencies that matter:

- **Prefixes diverge.** `cmd/inspect.go` leads with `inspect:`; `cmd/item.go`
  leads with a flag (`--grep:`) or nothing (`unknown collection %q`);
  `cmd/item.go:464` says `delete: %v`. There is no rule for when a prefix
  appears or what it names.
- **Quoting diverges.** Identifiers are usually `%q`
  (`unknown collection %q`) but sometimes bare (`%s/%s already exists`,
  `cmd/item.go:320`).
- **Cobra defaults leak.** Fixed-arity commands use `cobra.ExactArgs(1)` /
  `MinimumNArgs`, so a missing arg prints `accepts 1 arg(s), received 0`
  (the trigger for this spec). `inspect` is the only command with a custom,
  on-voice arity message. Flag-parse errors (`unknown flag`) print Cobra's
  text and exit `1`, not `2`.
- **Hints are rare.** Only two messages suggest a next step
  (`try \`katalyst schema list\``, `run \`katalyst init\``); the other
  "unknown X" errors leave the user to guess.
- **Capitalization/punctuation** are mostly lowercase and period-free already,
  but not enforced anywhere.

Terminology is already good — messages use the glossary's *collection*,
*item*, *selector*, *schema*, *check*. The diagnostic format
(`path:line: /pointer: message`, `cmd/check.go:128`) is deliberate and stays.

## Design

### Message anatomy

Every error message is one line:

```
[<subject>: ]<problem>[ (<hint>)]
```

- **`<subject>`** — the offending thing, when naming it adds clarity: a flag
  (`--limit`), or a leading path in diagnostic position (`notes/dune.md`). A
  message whose problem already names its subject (`unknown collection "wiki"`)
  takes no prefix. **The command name is never the subject** — the user knows
  which command they ran. (Arity/usage errors are the one exception; see below.)
- **`<problem>`** — what is wrong, lowercase, no trailing period, phrased as a
  state or a constraint (`must not be negative`, `not a readable directory`,
  `already exists`).
- **`<hint>`** — optional, parenthesized, a runnable next step in backticks
  (`(try \`katalyst collection list\`)`) or the usage line
  (`(usage: katalyst inspect <path>)`).

### Rules

1. **Lowercase, no trailing period.** Go error convention; errors compose into
   larger ones. Hints may contain a backticked command verbatim.
2. **Quote user-supplied tokens with `%q`** — identifiers (collection, item,
   schema, inspector, check kind), selectors, flag *values*. Never quote the
   flag name itself (it's literal: `--limit`, not `"--limit"`).
3. **Paths are bare** in the leading "diagnostic" position
   (`notes/dune.md: not a directory`) and in IO errors (`write report.md: …`),
   matching the existing `path:line:` format. Quote a path only when it sits
   mid-sentence and bareness would be ambiguous.
4. **Identifier-not-found** errors read `unknown <thing> %q`, with a discovery
   hint when one exists: `unknown collection %q (try \`katalyst collection
   list\`)`.
5. **Flag errors** read `--flag: <problem>`. Enumerations read
   `--flag: must be a, b, or c (got %q)`. Numeric bounds read
   `--flag: must not be negative`.
6. **IO errors** read `<verb> <path>: <cause>` with the wrapped cause last
   (`write report.md: permission denied`). Use `%w`, not `%v`, so callers can
   unwrap.
7. **Arity / usage errors** are the one place the command appears, inside the
   usage hint: `<problem> (usage: katalyst <cmd> <args>)`, e.g.
   `missing directory (usage: katalyst inspect <path>)`. Produced by a shared
   helper, not by Cobra's `ExactArgs`.
8. **Terminology is the glossary's.** Describe selector shape as
   `<collection>/<item>` (angle brackets).

### Streams and exit codes

Unchanged, but now stated as part of the standard:

- **stdout** — normal output (`path: OK`, listings, rendered reports).
- **stderr** — every error and every diagnostic (violations, unmatched files).
- A **usage error** (bad arguments, unknown identifier, refused overwrite,
  unreadable input, flag-parse failure) carries **exit 2** via `usageErr` /
  the `exitError` machinery in `cmd/check.go`.
- A **diagnostic failure** (check violations, `fix --check` pending) is
  **exit 1**.
- The `path:line: /pointer: message` diagnostic format is exempt from the
  prose rules: it is a machine contract and stays byte-for-byte.

### Cobra defaults

Two seams leak Cobra's voice; both get closed:

- **Arity.** Replace per-command `cobra.ExactArgs(n)` / `MinimumNArgs(n)` on
  fixed-shape commands with a shared `cmd` helper that returns a `usageErr`
  (exit 2) in the standard arity grammar. `inspect`'s existing custom validator
  becomes the template.
- **Flag parsing.** Set a `FlagErrorFunc` on the root so `unknown flag` and
  bad-value errors are lowercased, stripped of the `Error:` decoration, and
  routed through `usageErr` (exit 2) like every other usage error.

Unknown *subcommands* keep Cobra's handling for now (see Open Questions).

### Where the standard lives

This spec is staging. At graduation the style guide graduates into
**`cmd/AGENTS.md`** (a co-located package convention — messages are written in
`cmd/`), and the arity/flag helpers carry doc comments pointing at it. There is
no user-facing docs page for error wording; the diagnostic *format* is already
documented in the command reference and glossary.

## Open Questions

- **Unknown-subcommand handling.** Cobra prints `unknown command "foo" for
  "katalyst"` and exits 1. Routing it through `usageErr` (exit 2) for symmetry
  is possible via `RunE` on the root, but interacts with help/completion.
  Deferred unless trivial; not blocking.
- **A `--quiet`/verbosity knob for diagnostics** is out of scope — this spec is
  about wording and consistency, not new flags.

## Rejected alternatives

- **Per-command prefixes everywhere (`inspect:`, `item:`).** Uniform, but the
  command name is noise — the user knows what they ran, and the prefix
  crowds out the subject that actually helps (`--limit`, the bad path). Reserve
  the command name for usage hints, where it earns its place.
- **A message catalog / error-code registry** (à la `TS1005`). Overkill for a
  CLI this size; it adds indirection and a lookup step without helping the user
  more than a clear sentence does.
- **Capitalized, punctuated sentences.** Fights Go's error convention and the
  fact that these strings get wrapped (`%w`) into larger errors.

## Test checklist (what the pending tests assert)

Grammar (representative sites; the rule applies to all):
- [ ] missing required arg → standard arity message with usage hint, exit 2,
      and **not** Cobra's `accepts N arg(s)` text
- [ ] too many args → standard arity message, exit 2
- [ ] unknown flag → lowercased usage error, exit 2 (not Cobra's exit 1)
- [ ] `unknown collection %q` carries a `try \`katalyst collection list\`` hint
- [ ] `unknown schema` / `unknown inspector` / `unknown check kind` carry their
      discovery hints
- [ ] refuse-overwrite quotes the selector (`%q`), exit 2
- [ ] flag enumeration errors read `must be a, b, or c (got %q)`
- [ ] negative `--limit`/`--skip` read `--flag: must not be negative`
- [ ] IO errors read `<verb> <path>: <cause>` and wrap with `%w`

Invariants preserved:
- [ ] `path:line: /pointer: message` diagnostic format unchanged
- [ ] `path: OK` still on stdout; violations/errors still on stderr
- [ ] exit codes unchanged (0 ok, 1 diagnostic failure, 2 usage)

## Appendix: message inventory

The normalization target for each current site. Grouped by file; `→` shows the
standardized form. Sites already conformant are omitted.

**`cmd/inspect.go`** — drop the `inspect:` prefix; arity via the shared helper:
- `inspect: provide a directory to inspect, e.g. …` → `missing argument(s) (usage: katalyst inspect <path>)`
- `inspect: expected one directory, got %d` → `too many arguments (usage: katalyst inspect <path>)`
- `inspect: %q is not a readable directory` → `%s: not a readable directory`
- `inspect: unknown inspector %q` → `unknown inspector %q (try \`katalyst inspect --help\`)`
- `inspect: write %s: %v` → `write %s: %w`
- `inspect: %v` (load) → `%s: %w` (path + cause)

**`cmd/item.go` / `cmd/collection.go`**:
- `%s/%s already exists; refusing to overwrite` → `%q already exists; refusing to overwrite` (selector quoted)
- `--grep-in: unknown region %q (want all, body, or frontmatter)` → `--grep-in: must be all, body, or frontmatter (got %q)`
- `--on-type-mismatch: want skip or error, got %q` → `--on-type-mismatch: must be skip or error (got %q)`
- `--sort-missing: want last or lowest, got %q` → `--sort-missing: must be last or lowest (got %q)`
- `delete: %v` → `delete %s: %w`
- `unknown collection %q` (item/collection) → add hint `(try \`katalyst collection list\`)`

**`cmd/schema.go`**: already close; ensure `unknown schema %q (try …)` hint kept; `read %s: %w` conforms.

**`cmd/rules.go`**: enumeration/`unknown` messages aligned to the grammar
(`unknown check kind %q (try \`katalyst rules\`)`).

**Arity validators** (`collection get`, `item list/get/add/update/delete`,
`schema show`): replace `cobra.ExactArgs`/`MinimumNArgs` with the shared helper.

**`internal/project/`** selector + lookup errors: already lowercase and
quoted; add discovery hints at the `cmd/` boundary where they surface (the
package stays UI-agnostic — hints are appended by `asUsageErr` in `cmd/`, not
hard-coded in `internal/`).
