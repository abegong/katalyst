# internal/storage/collection

The collection read stack: how katalyst reads a collection's items from a
storage backend. `collection.go` is the backend-neutral contract
(`CollectionDefinition` + the thin `Item`); per-backend implementations live in
subpackages (`filesystem` today, `sql` later); `document` is the markdown codec
the readers decode and encode with; `query` is the filter/sort grammar.

Architecture and rationale — why a collection owns the read, why items are thin,
and how a backend attaches — live in the
[storage layer](../../../docs/content/deep-dives/storage.md) and
[collections](../../../docs/content/deep-dives/collections.md) deep-dives.

## Conventions

- The contract (`CollectionDefinition`, `Item`) stays backend-neutral: no
  filesystem assumptions (globbing, stem-as-id, path joins) leak into it. A new
  backend is a new subpackage implementing the interface.
- `document` is a leaf codec: it imports no other internal package. Keep it that
  way so the source-layer inspector and the checks can decode without pulling in
  a backend or `config`.
- Read and write are duals: the backend reader locates and `document.Parse`
  decodes; `fix` computes the new bytes and the backend persists them
  (`filesystem.Write`). Backend-specific IO stays in the backend subpackage.
