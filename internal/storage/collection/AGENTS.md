# internal/storage/collection

The collection read stack: how katalyst reads a collection's items from a
storage backend. `collection.go` is the backend-neutral contract
(`CollectionDefinition` + the thin `Item`); `parse.go` owns the `Collection`
type (plus `CollectionVariant`/`ListingDefaults`) and parses a collection's
config block (`Build`), since a collection is a storage concept; per-backend
implementations live in subpackages (`filesystem` today, `sql` later);
`internal/codec/markdownbodytext` is the markdown body text codec the readers
decode and encode with; `predicate` is the metadata predicate grammar; `listing`
is the in-memory `item list` pipeline.

Architecture and rationale — why a collection owns the read, why items are thin,
and how a backend attaches — live in the
[storage layer](../../../docs/content/deep-dives/domain-model/storage.md) and
[collections](../../../docs/content/deep-dives/domain-model/collections.md) deep-dives.

## Conventions

- The contract (`CollectionDefinition`, `Item`) stays backend-neutral: no
  filesystem assumptions (globbing, stem-as-id, path joins) leak into it. A new
  backend is a new subpackage implementing the interface.
- `internal/codec/markdownbodytext` is a leaf codec: it imports no other
  internal package. Keep it that way so the source-layer inspector and the
  checks can decode without pulling in a backend.
- `collection` owns the `Collection` type, so the project loader imports
  `collection`, not the reverse. Keep the edge pointing that way: `collection`
  imports `checks` (to parse a check's args) and the sibling `predicate`
  grammar, but never the loader. `Build` takes schema validation as an injected
  `SchemaKnown func(string) bool` rather than reaching back into the loader.
- Read and write are duals: the backend reader locates and
  `markdownbodytext.Parse` decodes; `fix` computes the new bytes and the backend
  persists them (`filesystem.Write`). Backend-specific IO stays in the backend
  subpackage.
