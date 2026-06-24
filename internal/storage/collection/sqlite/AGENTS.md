# internal/storage/collection/sqlite

SQLite storage backend. One configured table is one Katalyst collection; each
row is one item; the configured `id` column is item identity; scalar columns are
metadata; an optional `body` column supplies body text.

## Conventions

- Keep all SQL driver usage in this package. Other packages should reach
  SQLite through `project.Project` or the collection storage interfaces.
- Validate table and column identifiers before interpolating them into SQL.
  Values always use query parameters.
- Filesystem checks are rejected at load time for SQLite collections. Do not
  make check families backend-aware to compensate.
- `fix` is not part of the first SQLite cut. `item add`, `item update`, and
  `item delete` own the write-path coverage for now.
