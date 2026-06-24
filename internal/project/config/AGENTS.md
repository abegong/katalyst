# internal/project/config

Loads a project's `.katalyst/` directory, resolves named schemas and
collections, and decides which schema applies to a given item - the
orchestration hub the `check` lifecycle runs from. It is the `config` package
nested under `project`, alongside `collection`.

**Architecture and design rationale** - the `.katalyst/` layout and discovery,
the collection / item / selector / schema model, three-tier schema resolution,
variants, and the `check` lifecycle - live in the
[How collections work](../../../docs/content/deep-dives/collections.md) deep-dive,
which is the source of truth. The key-by-key surface is the
[configuration reference](../../../docs/content/reference/configuration.md), and the
code-level contract is `go doc ./internal/project/config`. This file keeps only the
local code conventions.

## Conventions

- `config` never imports `internal/storage`: it validates a declared storage
  `type` against a parse-time allowlist and otherwise treats it as opaque. The
  dependency runs the other way (`internal/storage` depends on `config`).
- `config` imports `internal/storage/collection/query` for the variant `when`
  predicate grammar; `query` imports no `config`, so there is no cycle. (This is
  the one `project/ → storage/` cross-tree edge; the config-distribution spec
  retires it.)
- Schema discovery resolves symlinks on both the root and the input path
  (`EvalSymlinks`), or relative-path resolution breaks under macOS `$TMPDIR`.
