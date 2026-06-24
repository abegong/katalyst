---
name: add-katalyst-check-type
description: Add a new Katalyst check type end-to-end across check execution, config parsing, CLI validation, tests, fixtures, and the generated Hugo check-types reference docs. Use when the user asks to add, extend, or document a new check type in this repository.
disable-model-invocation: true
---

# Add Katalyst Check Type

Use this skill to implement a new check type in this repo.

A check type now owns its own config. There is **no central config switch to
edit**: you add a kind constant, then write one file in the check's family
package that holds the struct, `Run`, the `Descriptor`, and an `init()` that
registers the check together with the parser for its YAML args. The loader
(`internal/project`) validates a configured check by calling that registered
parser at load time, so wiring is automatic.

## Quick Start

1. Add a `CheckType` constant in `internal/checks/kinds.go`.
2. Add one file in the family package under `internal/checks/<family>/`: the
   struct, `Run`, an args struct, the `Descriptor`, and an `init()` that calls
   the family's `registerParsed(...)` with a parse closure and a build closure.
   This *is* the config parsing, the CLI wiring, and the docs source.
3. Add unit + integration tests and fixtures.
4. Regenerate the check-types reference with `make docs-gen`.
5. Run validation commands and report results.

## Required Workflow

Copy this checklist and keep it updated:

```text
Check Type Task Progress:
- [ ] 1) Kind constant added (internal/checks/kinds.go)
- [ ] 2) Check file added (struct + Run + args + Descriptor + registerParsed)
- [ ] 3) Tests added/updated
- [ ] 4) Fixtures/readmes updated
- [ ] 5) Reference regenerated
- [ ] 6) Verification commands passed
```

## 1) Kind Constant

Add a `Check…` constant to the `CheckType` block in
`internal/checks/kinds.go`. Its string value is the `kind:` selector in YAML.
That is the only edit outside the family package — the loader has no per-kind
switch to update.

## 2) Check Implementation

Add one file in the check type's family package — `internal/checks/structuredobject/`,
`markdownbodytext/`, `filesystem/`, or `plaintext/` — picking the family by the
*source data* the check reads (not its scope). Pattern it on a sibling file
(e.g. `plaintext/requires.go`):

- The struct plus a `Run(ctx checks.Context) []checks.Violation` method (or
  `RunCollection(checks.CollectionContext)` for a collection-scoped check).
- An args struct with `yaml:"…"` tags for the check's own config keys.
- Prefer a pointer-like `Path` (`/field`) and a `Line` when known. Use
  `checks.LookupLine` for the line, and shared helpers (`checks.MarkdownLines`,
  the family's `common.go`) rather than re-deriving.
- An `init()` calling the family's `registerParsed(descriptor, parse, build,
  buildColl)`:
  - `parse func(*yaml.Node) (any, error)` decodes the node into the args struct
    and validates it. Use `internal/checks/argcheck` helpers (`RequireString`,
    `OneOf`, …) for uniform, test-stable error phrasing, plus any family-local
    validators in the family's `args.go`. A nil node means no args block.
  - `build func(any) checks.Check` type-asserts the args and constructs the
    check. A per-item check passes `build` and `nil` for `buildColl`; a
    collection-scoped check passes `nil` and a `buildColl func(any)
    checks.CollectionCheck`.
  - The `object` check is the exception — it registers a descriptor only
    (`checks.RegisterDescriptor`), because the engine builds it specially to
    compile a schema.
- Keep logic deterministic and side-effect free.

This is also the config parsing and the CLI wiring: the loader builds the
configured-check list by calling the registered parser (`checks.Parse`), and
`engine.checksFor` builds each check by registry lookup, so no `cmd/engine.go`
or loader edit is needed. `cmd/write_validation.go` shares `engine.checksFor`,
so write-path validation picks the new check up automatically.

Add focused unit tests in the family package's `_test` suite, using
`internal/checks/checktest` helpers.

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
- `internal/checks/registry_test.go` enforces that every descriptor is
  well-formed (family, slug, title, summary, config example, owning library), so
  a missing or malformed `Descriptor` fails the build.

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
