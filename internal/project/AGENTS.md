# internal/project

The project domain layer: finds `.katalyst/`, loads schemas and storage
instances, exposes collections, resolves selectors, and enumerates concrete
items for the CLI.

Architecture and rationale live in the
[domain model](../../docs/content/deep-dives/domain-model/_index.md),
[configuration](../../docs/content/reference/configuration.md), and
[storage](../../docs/content/deep-dives/domain-model/storage.md) docs. This file keeps only
local code conventions.

## Conventions

- The loader owns the `.katalyst/` vocabulary: discovery mode, config format,
  schema names, storage instance names, collection uniqueness, and selector
  parsing. Do not duplicate that parsing in `cmd/`.
- Storage and collection details stay below the storage boundary. This package
  assembles `storage/collection.Collection` values and calls a
  `CollectionDefinition`; it should not inline globbing, path joins, or
  filename-as-id assumptions.
- Keep import direction one-way: `project` imports `internal/storage` and
  `internal/storage/collection`, never the reverse.
- Public lookup/listing results are sorted for deterministic CLI output and
  tests. Preserve that property when adding new loaded concepts.
- Loader tests should focus on project configuration behavior: discovery,
  defaults, validation, collection shaping, and check spec parsing.
- It is fine for loader tests to verify end-to-end registry wiring at the check
  spec boundary. Concrete check implementation behavior belongs in the owning
  check-family tests.
- Use `internal/project/projecttest` for temporary `.katalyst` fixture setup.
  Do not duplicate project scaffolding helpers in individual packages when a
  test only needs fixture construction.
