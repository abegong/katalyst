# internal/examples

The worked-example registry. Each `Example` pairs a tiny input corpus with a
`katalyst` command; it is **a test that doubles as documentation**. The same
`Run` a golden test gates is what `cmd/gendocs` renders into the published docs,
so an example cannot drift from the tool. See
[How we test](https://github.com/abegong/katalyst/blob/main/docs/content/contributing/how-we-test.md)
for where this fits the wider testing strategy.

## Conventions

- **One entry per example** in `All()` (`examples.go`). `ID` is the slug used
  for the snippet filenames, the shortcode arguments, and the golden fixture
  (`testdata/<ID>.md`); it must be unique (guarded by `TestExamples_uniqueIDs`).
- **`Run` executes the real CLI** (`cmd.NewRootCmd`) in a temp dir and
  normalizes the temp path to `<project>`, so output is deterministic.
- **Gated by a golden test.** `TestExamples` snapshots `RenderPage`. Regenerate
  with `go test ./internal/examples -run TestExamples -update` and review the
  diff as the contract; a behavior change that alters an example fails here.
- **Two snippets per example.** `cmd/gendocs` writes
  `docs/generated/examples/<id>.txt` (raw output) and `<id>.full.md` (full
  corpus, command, and output, with headings at H3 via `RenderPageAt(_, _, 3)`).
- **Embedding.** Output only: `{{< katalyst-example "id" >}}`. Full corpus:
  `{{< katalyst-example-full "id" >}}`. Feature examples are routed onto their
  reference page by `examplesByPage` in `cmd/gendocs`; command- and
  workflow-level examples are embedded by hand on how-to and deep-dive pages.

## Corpus house style

- Data files first, then the base config, then a schema file if one is kept.
- Name the base config `.katalyst/bases/my_directory.yaml`.
- Prefer inline `checks:` over a schema file; keep a schema only when the example
  is specifically about schema binding.

Presentation polish for the rendered examples is tracked in #84; full coverage
across check types and inspectors in #85.
