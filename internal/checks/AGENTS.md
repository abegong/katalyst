# internal/checks

The check engine: the check types Katalyst ships, the libraries that run them,
and the violations they produce.

**Architecture and design rationale** - the model (check type vs. instance,
family vs. library, granularity), check libraries, how a check runs, and the
trade-offs - live in the
[How checks work](../../docs/content/deep-dives/checks.md) deep-dive, which is
the source of truth. The per-type catalog is the generated
[check-types reference](../../docs/content/reference/check-types/), and the
code-level contract is `go doc ./internal/checks`. This file keeps only the
local code conventions.

## Conventions

- One check type per file, with its `Descriptor` and an `init()` that registers
  it through the package's `register` helper (in `library.go`). To add one, see
  the [add-katalyst-check-type](../../.cursor/skills/add-katalyst-check-type/SKILL.md)
  skill.
- Family packages (`structuredobject/`, `markdownbodytext/`, `filesystem/`,
  `plaintext/`) import the core `checks` package, never the reverse. Callers
  wire every family in by blank-importing `internal/checks/all`.
- `kind` ids are the wire contract: never change an existing id, even when a
  check's family changes.
