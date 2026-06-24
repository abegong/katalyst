# cmd conventions

Conventions specific to the CLI layer. Root standards live in the repo
[`AGENTS.md`](../AGENTS.md); don't repeat them here.

## Command placement

The command surface is two grammars, kept apart (the full rationale is in
[docs/deep-dives/command-organization.md](../docs/content/deep-dives/command-organization.md)).
When adding a top-level command, decide which family it joins:

- **Blessed verb:** `katalyst <verb> [selector ...]`, a cross-cutting
  operation over content, accepting a selector at any depth and multiple
  selectors (`check`, `fix`). `init` and `inspect` are verbs too, they take
  flags or a path rather than a selector. `inspect` infers its inspector
  **layer** from the single argument: a configured collection name runs the
  collection layer; anything else is a filesystem path for the raw-source layer
  (with no project, always raw). Layer selection is by argument, deliberately
  not a flag, to keep the onboarding case (`inspect ./wiki`) flag-free.
- **Resource noun:** `katalyst <noun> <verb> <selector>`, a group whose
  CRUD-shaped sub-verbs act on one resource at a fixed depth (`collection`,
  `item`, `schema`, `rules`).

The rule, concretely:

- No top-level CRUD verbs: CRUD lives under its noun (`item add`, not `add`).
- No cross-cutting verb under a noun: `check` stays a blessed verb, never
  `item check`.
- A resource noun is built with no `RunE`, so invoking it bare prints help
  rather than running a default action. See `rules.go` / `collection.go` for
  the pattern: a parent command that only `AddCommand`s its sub-verbs.

When you change the no-args help surface (a new command, a renamed group, a
changed `Short`), update the golden string in `root_test.go`. Register a new
command in its help group in `root.go` (`addGrouped`), not a bare
`AddCommand`, an ungrouped command falls to "Additional Commands".

## Help text copy

Help text is product copy for people in a terminal. Keep it literal and
task-oriented.

- **Describe behavior, not implementation.** Say what the command does for the
  user (`inspect`, `initialize`, `run checks`), not how it is built.
- **Use imperative verbs for action commands.** Verb entries read as direct
  actions: `Inspect …`, `Initialize …`, `Run …`, `Apply …`.
- **Use `Commands to …` for resource nouns.** Noun parents describe their
  subcommand family, not an action they run themselves.
- **Use project vocabulary.** Prefer `collection`, `item`, `schema`, `check`,
  `inspector`, `selector`; avoid introducing synonyms unless they add clarity.
- **Keep one line scannable.** Favor short, concrete phrases over long
  qualifiers; remove filler and hedging words.
- **Keep tone neutral and mechanical.** No marketing adjectives and no
  anthropomorphic wording.
- **Order verbs by lifecycle.** In root help, list `Verbs` in the order a user
  runs them in a new project, not alphabetically.
- **Order resources by setup priority.** In root help, list `Resources` by what
  a project must configure first, then what follows.
- **No trailing periods in help descriptions.** `Short` strings and root help
  lines end without `.`.

Patterns to reuse:

- **Verb command (`Short`):** `<Imperative verb> <target> [and <outcome>]`
- **Resource noun (`Short`):** `Commands to <verb> and <verb> <resource> …`
- **Root `Long`:** one-sentence "what Katalyst is" + one-sentence "what you do
  with it", then stable project links.

Examples:

- `inspect     Analyze a directory and report its structure and conventions`
- `init        Initialize a directory as a katalyst project`
- `collection  Commands to inspect and modify collections in this project`

## Error messages

One grammar for every user-facing error:

```
[<subject>: ]<problem>[ (<hint>)]
```

- **Lowercase, no trailing period.** Go error convention; these strings get
  wrapped with `%w`.
- **Subject is the offending thing, never the command name:** a flag
  (`--limit`), or a leading path in diagnostic position (`notes/dune.md`). A
  problem that already names its subject (`unknown collection %q`) takes no
  prefix. The command name appears only in a usage hint (arity errors).
- **Quote user tokens with `%q`:** collection/item/schema/inspector/kind
  names, selectors, flag *values*. Never quote a flag name (`--limit`, literal).
- **Paths are bare** in leading/diagnostic position and IO errors
  (`write report.md: …`); use `%w` so the cause unwraps.
- **Flag errors:** `--flag: <problem>`. Enumerations: `--flag: must be a, b, or
  c (got %q)`. Bounds: `--flag: must not be negative`.
- **Not-found:** `unknown <thing> %q` with a discovery hint when one exists:
  `unknown collection %q (try \`katalyst collection list\`)`.
- **Hints** are parenthesized: a runnable `(try \`katalyst ...\`)` or
  `(usage: katalyst <cmd> <args>)`.

## Exit codes and streams

- `usageErr(msg)` → exit **2** (bad args, unknown identifier, refused
  overwrite, unreadable input, flag-parse failure). Diagnostic failures (check
  violations, `fix --check`) → exit **1**. See `exitError` in `check.go`.
- Errors and diagnostics go to **stderr**; normal output to **stdout**.
- The `path:line: /pointer: message` diagnostic format is a machine contract,
  exempt from the prose rules; don't reword it.

## Helpers (use these, don't hand-roll)

- **Arity:** `exactArgs(n, usage)`, `minArgs(n, usage)`, `maxArgs(n, usage)`
  (`usage.go`) instead of `cobra.ExactArgs` etc., they emit the standard usage
  error (exit 2), not Cobra's `accepts N arg(s)`.
- **Unknown collection:** `unknownCollectionErr(name)`.
- **Flag-parse errors** are routed through `usageErr` by the root
  `FlagErrorFunc` (`root.go`); nothing to do per command.

New commands inherit the standard by using these helpers and this grammar.

## Testing the CLI

Two test styles, kept apart:

- **Snapshot the text.** User-facing output contracts — help, `list`/`show`
  output, the `inspect` report, canonical stderr diagnostics — are pinned as
  golden fixtures via the `snapshot` harness (`snapshot_test.go`), reviewed as
  plain text under `testdata/snapshots/`. Generate with `-update`, then review
  the diff. Output that embeds a temp path is normalized with `normTmp(dir)`.
- **Property-test the behavior.** Exit codes (`errors.As` on `Code()`),
  precedence (`--schema`, inline `schema:`, variants), side effects
  (`add`/`update`/`delete`), `--json` shape, and query semantics
  (`item list` filters) stay assertions in code.
- **Hybrid tests keep both halves.** A test that asserts an exit code *and* a
  message keeps the `Code()` check and moves only the wording to a snapshot. A
  snapshot existing for a surface never justifies dropping a behavior assertion.
