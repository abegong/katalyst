# internal/fix

The transform engine for `katalyst fix`: given an item's content and its
collection, compute the canonical, fixed form. Backend-agnostic and IO-free —
deciding *what* to write lives here; *persisting* it is the storage backend's
job (`storage/collection/filesystem.Write`).

Why the canonical form is deliberately inflexible, and why `fix` never injects
missing values, lives in the
[Frontmatter and fix](../../docs/content/deep-dives/domain-model/formatting.md) deep-dive.

## Conventions

- No file IO. `fix` returns bytes; the caller persists through the backend.
- Serialization belongs to the `internal/codec/markdownbodytext` codec
  (`Parse`/`Encode`); `fix` is policy (canonical ordering, text fixes), not
  byte plumbing.
- A text fix re-checks its own work and fails rather than emit a still-broken
  result.
