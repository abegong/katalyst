---
name: add-katalyst-check-type
description: Add a new Katalyst check type end-to-end across config parsing, check execution, CLI validation, tests, fixtures, and the generated Hugo check-types reference docs. Use when the user asks to add, extend, or document a new check type in this repository.
disable-model-invocation: true
---

# Add Katalyst Check Type

Use this skill to implement a new check type in this repo.

## Quick Start

1. Define the new check type and config payload in `internal/project/config/config.go`.
2. Implement the check in its family package under `internal/checks/<family>/`:
   one new file holding the struct, `Run`, its `Descriptor`, and an `init()`
   that calls `checks.Register` (this *is* the CLI wiring and the docs source).
3. Ensure write-path validation uses it via `cmd/write_validation.go` (usually
   automatic — it shares `engine.checksFor`).
4. Add unit + integration tests and fixtures.
5. Regenerate the check-types reference with `make docs-gen`.
6. Run validation commands and report results.

## Required Workflow

Copy this checklist and keep it updated:

```text
Check Type Task Progress:
- [ ] 1) Config model updated
- [ ] 2) Check file added (struct + Run + Descriptor + Register)
- [ ] 3) Tests added/updated
- [ ] 4) Fixtures/readmes updated
- [ ] 5) Reference regenerated
- [ ] 6) Verification commands passed
```

## 1) Config Model

Edit `internal/project/config/config.go`:

- Add a `CheckType` constant for the new check type.
- Extend `rawCheck` parsing if the check type needs new fields.
- Update `normalizeCheck(...)` validation and defaults.
- Preserve the collection `schema:` shorthand (sugar for a leading `object`
  check).

Add/extend tests in `internal/project/config/config_test.go`:

- Parses valid check payload.
- Rejects malformed payload.
- Rejects unknown check type.

## 2) Check Implementation

Add one file in the check type's family package — `internal/checks/structuredobject/`,
`markdownbodytext/`, `filesystem/`, or `plaintext/` — picking the family by the
*source data* the check reads (not its scope). Pattern it on a sibling file:

- The struct plus a `Run(ctx checks.Context) []checks.Violation` method (or
  `RunCollection(checks.CollectionContext)` for a collection-scoped check).
- Prefer a pointer-like `Path` (`/field`) and a `Line` when known. Use
  `checks.LookupLine` for the line, and shared helpers (`checks.MarkdownLines`,
  the family's `common.go`) rather than re-deriving.
- An `init()` calling `checks.Register(descriptor, build, buildColl)`: a per-item
  check passes a `build` closure and `nil` for `buildColl`; a collection-scoped
  check passes `nil` and a `buildColl` closure. The closure constructs the check
  from a `config.CheckInstance`. (The `object` check is the exception — it
  registers a descriptor with nil builders because the engine builds it specially
  to compile a schema.)
- Keep logic deterministic and side-effect free.

This is also the CLI wiring: `engine.checksFor` builds every non-object check by
registry lookup, so no `cmd/engine.go` edit is needed. Add focused unit tests in
the family package's `_test` suite, using `internal/checks/checktest` helpers.

Ensure `cmd/write_validation.go` still validates via the same pipeline (it shares
`engine.checksFor`, so this is usually automatic).

## 3) Tests and Fixtures

Integration tests:

- `cmd/check_test.go` for behavior and error output.
- `cmd/item_test.go` if write-path behavior changes.

Fixture conventions:

- Reusable fixtures go in `cmd/testdata/...`.
- Embed via `cmd/fixtures_test.go`.
- Document fixture purpose in `cmd/testdata/AGENTS.md`.

Follow `AGENTS.md` testing rules:

- external `_test` packages
- stdlib assertions only
- `t.TempDir()` isolation

## 4) Docs

The check-types reference is **generated**, not hand-written. Do not edit
`docs/content/reference/check-types/` directly.

- The `Descriptor` lives in the check type's own file (step 2). Set its family
  (`structuredObject`/`markdownBodyText`/`fileSystem`/`plainText`), slug, title,
  one-line summary, any configuration `Fields`, and a `ConfigExample` snippet.
- Run `make docs-gen` to regenerate `docs/content/reference/check-types/`, and
  commit the result.
- `registry_test.go` enforces parity between `normalizeCheck`'s switch and the
  registered descriptors, so a missing `Descriptor` fails the build.

## 5) Verify

Run:

```bash
gofmt -w .
go test ./...
make docs-gen-check   # regenerates the reference and fails on drift
```

If any command fails, fix issues and rerun before final handoff.

## Output Requirements

When done, report:

1. Files changed
2. Behavior added
3. Tests added
4. Verification command results
