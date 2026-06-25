# internal/storage

The backend boundary. This package names base backend kinds and keeps the
small registry of implemented backends; `collection/` holds the mapping from a
backend store to Katalyst collections and items.

Architecture and rationale live in the
[Bases deep-dive](../../docs/content/deep-dives/domain-model/storage.md). The collection
read stack has its own local guide in
[`collection/AGENTS.md`](collection/AGENTS.md).

## Conventions

- Add a backend kind here only when its `CollectionDefinition` implementation
  exists. `Known` is the source of truth the project loader uses to validate
  configured base types.
- `Reference` is opaque. Treat it as a backend-native locator, not always a
  filesystem path; filesystem interpretation belongs in `collection/filesystem`.
- Scope is a property of the base type, not user configuration. Keep that
  decision in code so collection/item roles stay portable across backends.
- Keep this package small and dependency-light. Backend-specific parsing,
  discovery, IO, and persistence belong under `collection/<backend>/`.
