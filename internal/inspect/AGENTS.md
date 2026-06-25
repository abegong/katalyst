# internal/inspect

Profiles content and returns evidence (counts and distributions), the
descriptive dual of `internal/checks`.

**Architecture and design rationale** - the two layers, the measurement
primitives, evidence-not-recommendations, the determinism dividing line - live
in the [How inspectors work](../../docs/content/deep-dives/domain-model/inspectors.md)
deep-dive (also summarized in `go doc ./internal/inspect`), which is the source
of truth. This file keeps only the local code conventions.

## Conventions

- Inspectors self-register into the registry with a per-layer parity test
  (mirroring `internal/checks`), so none ships undocumented.
- Inspectors **measure only**: counts and distributions, with the unit count
  `n` as denominator. Threshold-picking and structure-proposing belong to the
  caller, never here.
- Collection-view tests are project-backed because `NewCollectionView` resolves
  collection items through `internal/project`. Use `internal/project/projecttest`
  for `.katalyst` fixture setup so that import stays explicit integration setup
  rather than ad hoc YAML construction.
