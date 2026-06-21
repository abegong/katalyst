# cmd conventions

Conventions specific to the CLI layer. Root standards live in the repo
[`AGENTS.md`](../AGENTS.md); don't repeat them here.

## Error messages

One grammar for every user-facing error:

```
[<subject>: ]<problem>[ (<hint>)]
```

- **Lowercase, no trailing period.** Go error convention; these strings get
  wrapped with `%w`.
- **Subject is the offending thing, never the command name** — a flag
  (`--limit`), or a leading path in diagnostic position (`notes/dune.md`). A
  problem that already names its subject (`unknown collection %q`) takes no
  prefix. The command name appears only in a usage hint (arity errors).
- **Quote user tokens with `%q`** — collection/item/schema/inspector/kind
  names, selectors, flag *values*. Never quote a flag name (`--limit`, literal).
- **Paths are bare** in leading/diagnostic position and IO errors
  (`write report.md: …`); use `%w` so the cause unwraps.
- **Flag errors:** `--flag: <problem>`. Enumerations: `--flag: must be a, b, or
  c (got %q)`. Bounds: `--flag: must not be negative`.
- **Not-found:** `unknown <thing> %q` with a discovery hint when one exists —
  `unknown collection %q (try \`katalyst collection list\`)`.
- **Hints** are parenthesized: a runnable `(try \`katalyst …\`)` or
  `(usage: katalyst <cmd> <args>)`.

## Exit codes and streams

- `usageErr(msg)` → exit **2** (bad args, unknown identifier, refused
  overwrite, unreadable input, flag-parse failure). Diagnostic failures (check
  violations, `fix --check`) → exit **1**. See `exitError` in `check.go`.
- Errors and diagnostics go to **stderr**; normal output to **stdout**.
- The `path:line: /pointer: message` diagnostic format is a machine contract —
  exempt from the prose rules; don't reword it.

## Helpers (use these, don't hand-roll)

- **Arity:** `exactArgs(n, usage)`, `minArgs(n, usage)`, `maxArgs(n, usage)`
  (`usage.go`) instead of `cobra.ExactArgs` etc. — they emit the standard usage
  error (exit 2), not Cobra's `accepts N arg(s)`.
- **Unknown collection:** `unknownCollectionErr(name)`.
- **Flag-parse errors** are routed through `usageErr` by the root
  `FlagErrorFunc` (`root.go`); nothing to do per command.

New commands inherit the standard by using these helpers and this grammar.
