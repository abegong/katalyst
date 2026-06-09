---
name: add-katalyst-rule
description: Add a new Katalyst rule/check kind end-to-end across config parsing, check execution, CLI validation, tests, fixtures, and Hugo rules reference docs. Use when the user asks to add, extend, or document a new rule kind in this repository.
disable-model-invocation: true
---

# Add Katalyst Rule

Use this skill to implement a new rule/check kind in this repo.

## Quick Start

1. Define the new check kind and config payload in `internal/config/config.go`.
2. Implement check behavior in `internal/checks/`.
3. Wire check instantiation in `cmd/validate.go` (`checksFor`).
4. Ensure write-path validation uses it via `cmd/write_validation.go`.
5. Add unit + integration tests and fixtures.
6. Update rules reference docs under `docs/rules/`.
7. Run validation commands and report results.

## Required Workflow

Copy this checklist and keep it updated:

```text
Rule Task Progress:
- [ ] 1) Config model updated
- [ ] 2) Check implementation added
- [ ] 3) CLI wiring updated
- [ ] 4) Tests added/updated
- [ ] 5) Fixtures/readmes updated
- [ ] 6) Docs updated
- [ ] 7) Verification commands passed
```

## 1) Config Model

Edit `internal/config/config.go`:

- Add a `CheckKind` constant for the new rule.
- Extend `rawCheck` parsing if the rule needs new fields.
- Update `normalizeCheck(...)` validation and defaults.
- Keep backward compatibility with legacy `rules[].schema`.

Add/extend tests in `internal/config/config_test.go`:

- Parses valid check payload.
- Rejects malformed payload.
- Rejects unknown kind.

## 2) Check Implementation

Add a new check type in `internal/checks/`:

- Follow the existing `Run(ctx Context) []Violation` pattern.
- Prefer returning a pointer-like `Path` (`/field`) and `Line` when known.
- Keep logic deterministic and side-effect free.

Update `internal/checks/checks_test.go` with focused unit tests.

## 3) CLI Wiring

Edit `cmd/validate.go` in `checksFor(...)`:

- Map new `config.CheckKind` to the new `checks.*` implementation.
- Preserve precedence behavior:
  - `--schema` overrides object schema checks only.
  - non-object checks come from the first matched rule.

Ensure `cmd/write_validation.go` still uses the same check pipeline for
`create`/`update` strict validation.

## 4) Tests and Fixtures

Integration tests:

- `cmd/validate_config_test.go` for behavior and error output.
- `cmd/crud_test.go` if write-path behavior changes.

Fixture conventions:

- Reusable fixtures go in `cmd/testdata/...`.
- Embed via `cmd/fixtures_test.go`.
- Document fixture purpose in `cmd/testdata/README.md`.

Follow `AGENTS.md` testing rules:

- external `_test` packages
- stdlib assertions only
- `t.TempDir()` isolation

## 5) Docs

Update docs as reference-first content:

- Add/extend pages in `docs/rules/`.
- Keep clean human page titles.
- Put exact machine identifier at top as:
  - `## Rule ID`
  - ``kind: ...``
- Update `docs/configuration.md` links if needed.

## 6) Verify

Run:

```bash
gofmt -w .
go test ./...
make docs-build
```

If any command fails, fix issues and rerun before final handoff.

## Output Requirements

When done, report:

1. Files changed
2. Behavior added
3. Tests added
4. Verification command results
