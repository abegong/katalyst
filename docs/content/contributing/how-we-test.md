+++
title = "How we test"
weight = 20
+++

# How we test

Katalyst follows TDD: new behavior arrives with a failing test first. Two traits
shape the suite beyond that. Tests are the source of truth, and many of our
docs are generated from them: the reference pages come from the check and
inspector registries, and the worked examples come from the example registry,
so a green suite keeps behavior and its documentation honest at once. And we
test at **seams**, the named interfaces a `katalyst` run flows through, with a
small set of **styles** that interlock into end-to-end coverage.

## Testing seams

A `katalyst` run flows through a pipeline of narrow, independently testable
boundaries, such as config loading, frontmatter parsing, and the check engine,
composed behind the CLI. We **unit-test each major module under `internal/`**
against its own boundary, with inline literals or a small scaffolded input, so
each assertion stays fast and close to the code that can break it.

A behavior is covered when its owning seam has a focused unit test **and** the
CLI or docs tests exercise the composition (below). Push each assertion to the
lowest level that can make it; reserve the CLI for what genuinely needs the
whole pipeline. Known coverage gaps are tracked in
[issue #86](https://github.com/abegong/katalyst/issues/86).

## How they interlock

- **Unit at each module.** Fast, precise tests against one module's exported API.
- **Integration at the CLI.** The `cmd` tests drive the real Cobra root over a
  temp project, exercising config, storage, the engine, and the checks together.
  Snapshot the user-facing text; property-test the behavior (exit codes, side
  effects, query semantics).
- **Parity guards at the registries.** `registry_test` asserts every check type
  and inspector has a descriptor and a library, so a check type or inspector
  cannot ship orphaned or drift out of sync with the docs.
- **Generated docs and dogfood close the loop.** `docs-gen-check` fails on doc
  drift, and CI runs `katalyst check` over the project's own `.katalyst/`
  corpus, validating real content.

## Test styles

Any style can apply anywhere, but each has a usual home. Three carry most of the
suite.

- **Behavior (property) tests.** The default, across the `internal/` modules and
  for CLI behavior: assert what the code does (outputs, exit codes, semantics).
- **Text-contract snapshots.** At the CLI: pin user-facing output (help, list
  and show, diagnostics) with golden files under `cmd/testdata/snapshots/`, and
  keep the behavior behind that text as property tests. Snapshot the text,
  property-test the behavior.
- **Executable examples that double as docs.** In the docs: the
  `internal/examples` registry runs a real command over a tiny corpus; a golden
  test gates the output and `cmd/gendocs` renders the same run into the
  published docs, so an example cannot drift. Embed one into a prose page with
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

## Testing conventions

Mechanical conventions for writing tests live in the root
[`AGENTS.md`](https://github.com/abegong/katalyst/blob/main/AGENTS.md); see its
Testing section. Per-package `testdata/AGENTS.md` files contain additional
conventions as needed.
