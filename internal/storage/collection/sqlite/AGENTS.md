# internal/storage/collection/sqlite

SQLite storage backend. One configured table is one Katalyst collection; each
row is one item; the configured `id` column is item identity; configured column
captures become item attributes; optional `content` maps one column into text or
markdown content.

## Conventions

- Keep all SQL driver usage in this package. Other packages should reach
  SQLite through `project.Project` or the collection storage interfaces.
- Validate table and column identifiers before interpolating them into SQL.
  Values always use query parameters.
- Filesystem checks are rejected at load time for SQLite collections. Do not
  make check families backend-aware to compensate.
- Prefer `attributes` and `content` terminology in new SQLite work. `body:` is a
  compatibility alias, not the model.
- `fix` reaches a collection that maps a content column, gated on
  `Collection.HasTextContent()` rather than on the backend name. It writes the
  content column alone: the frontmatter it canonicalizes was synthesized from the
  row by `rawDocument`, so it is canonical already and the attribute columns have
  no reason to move.
