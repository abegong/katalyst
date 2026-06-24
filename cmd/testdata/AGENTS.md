# cmd/testdata

Reusable fixtures for `cmd/` CLI tests. Loaded via `//go:embed` in
[`../fixtures_test.go`](../fixtures_test.go) so tests can reference them
even after `chdir`-ing into a temp directory.

See [`AGENTS.md`](../../AGENTS.md) for the project-wide policy on when
to use a fixture vs an inline literal.

## Fixtures

### `schemas/`

| File              | Used by                                                  | Purpose                                                          |
|-------------------|----------------------------------------------------------|------------------------------------------------------------------|
| `book.json`       | `check_test.go`, `item_test.go`, `schema_test.go`        | Minimal `{title, year}` schema; carries JSON-Schema `title: book`. |
| `person.json`     | `schema_test.go`, `collection_test.go`                   | Minimal object schema; carries JSON-Schema `title: person`.       |
| `strict-book.json`| `check_test.go`, `item_test.go`                          | Adds required `isbn` with a regex `pattern`. Used to test that an inline `schema:` key in a doc's frontmatter overrides the collection's configured schema. |

These are deliberately simpler than `internal/checks/jsonschema/testdata/schemas/book.json`,
which exists to exercise the validator itself.

### `snapshots/`

Golden fixtures for the CLI's user-facing text contracts, compared by the
`snapshot` harness in [`../snapshot_test.go`](../snapshot_test.go). The whole
tree is embedded in one `//go:embed testdata/snapshots`
([`../fixtures_test.go`](../fixtures_test.go)) so reads survive the per-test
`chdir`. The split — snapshot the *text*, property-test the *behavior* — is
specified in [`../AGENTS.md`](../AGENTS.md) ("Testing the CLI").

Fixtures are grouped one directory per command surface; the name passed to
`snapshot(t, "<group>/<name>.txt", …)` is the path under `snapshots/`:

| Group | Surface |
|---|---|
| `help/` | Root help and every `<cmd> --help` (`root_test.go`, `help_snapshot_test.go`) |
| `collection/` | `collection list`, `collection get` |
| `schema/` | `schema list`, `schema get` |
| `check-types/` | `check-types list`, `list --family`, `show` |
| `inspectors/` | `inspectors list`, `list --layer`, `show` |
| `item/` | `item list`, `item get --frontmatter`/`--body` |
| `inspect/` | The `inspect` source-layer Markdown report (path normalized) |
| `check/` | Canonical stderr diagnostics — pointer, unmatched, writing-tell (paths normalized) |
| `selftest/` | Tiny fixture the harness self-tests read; not a command surface |

### Readout formatting contract

For human-facing read commands, snapshots also enforce the terminal layout
contract:

- section header followed by an underline divider,
- `list` sections include counts,
- entries render as bullets with indented detail lines.

Do not apply this layout to machine-contract surfaces (`check/` diagnostics,
`fix --check` path lists, JSON output, raw content output).

**Path normalization.** Output that embeds the test's temp dir (the `check`
diagnostics, the `inspect` report header) is passed through `normTmp(dir)`,
which rewrites the temp path to `<project>` so the fixture is deterministic.
Pure-text surfaces need no normalizer.

**Updating.** Fixtures are generated, never hand-written. Regenerate with the
`-update` flag, then **review the diff as the contract** before committing:

```
go test ./cmd -run TestThing -update
```

Project layout (the `.katalyst/` directory, schema files, and collection
bodies) is scaffolded inline by `writeProject` / `storageLocal` in
[`../helpers_test.go`](../helpers_test.go) rather than as fixtures. Because the
shared schema fixtures are JSON, test projects set `schemas: { format: json }`
in their `.katalyst/config.yaml` (the `schemaFormatJSON` helper).

## Adding a fixture

**A snapshot fixture** (under `snapshots/`): write the test calling
`snapshot(t, "<group>/<name>.txt", got [, normTmp(dir)])`, run it once with
`-update` to generate the file, review the generated text, then re-run without
`-update` to confirm it asserts. No `//go:embed` line is needed — the whole
`snapshots/` tree is already embedded; just add a row to the group table above
if you introduce a new surface.

**A reusable input** (e.g. under `schemas/`):

1. Drop the file under the relevant subfolder (or a new one when needed).
2. Embed it in `../fixtures_test.go` with `//go:embed`.
3. Add a row to the relevant table above.
