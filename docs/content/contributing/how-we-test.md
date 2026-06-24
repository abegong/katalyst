+++
title = "How we test"
weight = 20
+++

# How we test

Katalyst follows TDD: new behavior arrives with a failing test first. Two traits
shape the suite beyond that. Tests are the source of truth, and the docs are
generated from them: the reference pages come from the check and inspector
registries, and the worked examples come from the example registry, so a green
suite keeps behavior and its documentation honest at once. And we test at
**seams**, the named interfaces a `katalyst` run flows through, with a small set
of **styles** that interlock into end-to-end coverage.

## Testing seams

A `katalyst` run flows through a pipeline of narrow, independently testable
boundaries: config loading, project resolution, storage, frontmatter parsing,
the check engine, the checks themselves, inspectors, and the query layer,
composed behind the CLI. We **unit-test each major module under `internal/`**
against its own boundary, with inline literals or a small scaffolded input, so
each assertion stays fast and close to the code that can break it.

A behavior is covered when its owning seam has a focused unit test **and** the
CLI seam exercises the composition (below). Push each assertion to the lowest
seam that can make it; reserve the CLI for what genuinely needs the whole
pipeline. The
[testing coverage report](https://github.com/abegong/katalyst/blob/main/product/testing-coverage-report.md)
maps every seam to its boundary and its current coverage.

## How the seams interlock

- **Unit at each seam.** Fast, precise tests against one module's exported API.
- **Integration at the CLI seam.** The `cmd` tests drive the real Cobra root
  over a temp project, exercising config, storage, the engine, and the checks
  together. Snapshot the user-facing text; property-test the behavior (exit
  codes, side effects, query semantics).
- **Parity guards at the registries.** `registry_test` asserts every check type
  and inspector has a descriptor and a library, so a seam cannot ship orphaned
  or drift out of sync with the docs.
- **Generated docs and dogfood close the loop.** `docs-gen-check` fails on doc
  drift, and CI runs `katalyst check` over the project's own `.katalyst/`
  corpus, validating real content.

## Test styles

Any style can apply at any seam. Three carry most of the suite.

- **Behavior (property) tests.** The default: assert what the code does.
- **Text-contract snapshots.** Pin user-facing output (help, list and show,
  diagnostics) with golden files under `cmd/testdata/snapshots/`, and keep the
  behavior behind that text as property tests. Snapshot the text, property-test
  the behavior.
- **Executable examples that double as docs.** The `internal/examples` registry
  runs a real command over a tiny corpus; a golden test gates the output and
  `cmd/gendocs` renders the same run into the published docs, so an example
  cannot drift. Embed one into a prose page with
  `{{</* katalyst-example "id" */>}}` (output only) or
  `{{</* katalyst-example-full "id" */>}}` (full corpus and command). See
  `internal/examples/AGENTS.md` for how to add one.

## The golden-file workflow

Golden fixtures, snapshots, the generated reference, and the worked examples are
generated, never hand-written. Regenerate, then review the diff as the contract
before committing:

```bash
go test ./cmd -run TestThing -update   # snapshots
make docs-gen                          # generated reference and examples
```

`make docs-gen-check`, run in CI, fails if the committed output drifts from its
generators.

## Where the code-level conventions live

This page is the strategy. The mechanical conventions for writing a Go test in
this repo, external test packages, standard-library-only, naming, `t.TempDir`,
fixtures versus inline literals, and `//go:embed`, live in the root
[`AGENTS.md`](https://github.com/abegong/katalyst/blob/main/AGENTS.md) `Testing`
section and the per-package `testdata/AGENTS.md` files. Do not repeat them here.
