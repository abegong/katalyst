# Spec - grouped filesystem unmatched-file diagnostics

> **Status: implementing.** Issue #126 asks `filesystem_unmatched_files` to
> collapse noisy per-file diagnostics for whole disallowed subtrees while keeping
> detailed output available on demand.

## Overview

`filesystem_unmatched_files` reports regular files under a filesystem scope that
match neither `include` nor `exclude`. Today it emits one violation per
unmatched file. That is precise, but it is noisy when the real problem is that
an entire sibling directory sits outside the configured allowlist.

Change the default output for this check so repeated unmatched files under one
disallowed subtree collapse into one directory-level diagnostic with a file
count. Add a verbose path that preserves the old per-file output.

This is a targeted reporting change for one check type, not a generic
diagnostic deduplication layer.

## Value

Users read `katalyst check` output to decide what to fix next. When a single
configuration gap produces dozens or hundreds of identical unmatched-file
diagnostics, the output hides the useful signal.

Given a filesystem scope like:

```yaml
filesystemChecks:
  - name: docs
    path: .
    include:
      - README.md
      - ongoing/**
      - episodic/**
    checks:
      - kind: filesystem_unmatched_files
```

and a disallowed `one-time/` subtree with many files, the default diagnostic
should point at the directory problem:

```text
filesystem docs: one-time/: /: unmatched files (28 files; matches no include pattern [README.md, ongoing/**, episodic/**] and no exclude pattern [])
```

An unmatched file inside an otherwise allowed directory should still report as
a file:

```text
filesystem docs: ongoing/stray.tmp: /: unmatched file (matches no include pattern [README.md, ongoing/**, episodic/**] and no exclude pattern [])
```

Users who need every file can run:

```sh
katalyst check --verbose
```

## Current State

Filesystem checks run only for no-selector `katalyst check`. The command expands
each configured filesystem scope through `internal/storage/filesystemcheck`.
That expansion produces two sorted sets:

- `Selected`: regular files matching at least one `include` pattern and no
  `exclude` pattern.
- `Unmatched`: regular files matching neither `include` nor `exclude`.

`cmd/filesystem_check.go` turns the expansion into `checks.FileSetContext`.
`filesystem_unmatched_files` reads `FileSetContext.Unmatched` and emits one
`checks.Violation` per relative path. The check's current message is identical
for every file in a scope because the reason is the scope's include and exclude
set, not a per-file property.

Collection unmatched-file reporting in `cmd/check.go` is a separate legacy path
for collection selectors. It reports files inside a collection directory that do
not match the collection pattern. This spec does not change that path.

This work extends the existing FileSetCheck model from
`docs/content/deep-dives/domain-model/checks.md`: file-set checks run once over
the selected file set and report violations with an optional sibling `File`.
`filesystem_unmatched_files` already fits that model because it reasons over the
scope's unmatched file set, not one selected file at a time.

## Design

### Grouping rule

For default output, `filesystem_unmatched_files` derives report targets from the
selected and unmatched file sets:

1. Sort `ctx.Unmatched` for deterministic output.
2. Build the set of directories that contain selected files, using paths
   relative to `ctx.Root`.
3. Count how many unmatched files sit below each ancestor directory.
4. For each unmatched file, find the shallowest ancestor directory that:
   - has no selected files below it, and
   - contains more than one unmatched file.
5. Group files by that directory. Report the directory with a trailing slash and
   the number of files it represents.
6. Leave files with no such directory as individual file diagnostics.
7. Sort final report targets by displayed path.

This rule preserves detail in mixed directories. If `ongoing/page.md` is
selected and `ongoing/stray.tmp` is unmatched, `ongoing/` is not collapsed
because it contains selected content. The stray file remains visible.

The rule also avoids unhelpful one-file directory summaries. A directory with
only one unmatched file reports that file, because `foo/ (1 file)` is not less
noisy or clearer than `foo/bar.tmp`.

### Message shape

Single-file reports keep the existing wording:

```text
unmatched file (matches no include pattern [..] and no exclude pattern [..])
```

Grouped reports use plural wording and include the count:

```text
unmatched files (N files; matches no include pattern [..] and no exclude pattern [..])
```

The diagnostic printer remains unchanged, so grouped filesystem diagnostics use
the existing contract:

```text
filesystem <scope>: <path>: /: <message>
```

### Verbose output

Add `katalyst check --verbose` / `-v`. For now, the flag affects only
filesystem set checks that opt into `FileSetContext.Verbose`.

`filesystem_unmatched_files` uses `Verbose` to skip grouping and return the old
one-violation-per-file output. Other checks ignore the flag.

The flag is intentionally phrased around unmatched filesystem files rather than
all diagnostics:

```text
Show every unmatched filesystem file instead of grouped directory summaries.
```

This keeps room for future verbose behavior without promising it yet.

## Open Questions

_None._

The cross-check question is resolved in the design: do not add a generic
diagnostic grouping layer for this issue. Keep grouping local to
`filesystem_unmatched_files` until another check has the same shape and the same
need for selected-set context.

## Documentation updates

- `cmd/testdata/snapshots/help/check.txt`: pin the new `--verbose` / `-v` flag.
- `docs/content/reference/cli.md`: no change. The page is deliberately thin and
  points readers to command help for per-command flags.
- `docs/content/reference/check-types/file-system/unmatched-files.md`: no
  hand edit. The page is generated from the descriptor, and this change does not
  alter descriptor metadata. Do not run `make docs-gen` for this change alone.
- `docs/content/deep-dives/domain-model/checks.md`: no change. The file-set
  check architecture stays the same. Update this page only if
  `FileSetContext.Verbose` becomes a broader runtime contract.
- `AGENTS.md`, package `AGENTS.md` files, Go package docs, and `.cursor/skills`:
  no change. This is behavior inside one check type, not a new contributor
  convention or skill workflow.

## Rejected alternatives

### Generic diagnostic deduplication

The grouping belongs in `filesystem_unmatched_files`, not in
`printFilesystemViolation` or a shared violation reducer.

- The check has the domain data needed to distinguish a disallowed subtree from
  a stray file in an allowed subtree: selected files, unmatched files, and the
  scope root.
- Generic printers see only flattened violations. By then, the relationship
  between selected files and unmatched files has been lost.
- Other high-volume checks need different semantics. `unique_filename` and
  `filesystem_unique_field` already group by collision value. Text checks and
  path-segment checks often need file or line specificity. A generic grouping
  layer would either under-group the issue case or over-group unrelated checks.

This spec therefore adds one reusable context bit, `FileSetContext.Verbose`,
but keeps grouping behavior local to the check that owns the concept.

### Collection unmatched-file grouping

Collection unmatched-file diagnostics in `cmd/check.go` remain unchanged. They
come from collection selector resolution and enforce a different invariant:
files inside a configured collection directory must match the collection
pattern. The issue targets filesystem scopes, where the check has both
`Selected` and `Unmatched` sets and can decide whether a subtree is wholly
outside the allowlist.

## Scope

In scope:

- Default grouping for `filesystem_unmatched_files`.
- `--verbose` / `-v` on `katalyst check`.
- Test coverage at the check level and CLI level.
- Help snapshot update for the new flag.

Out of scope:

- Changing collection unmatched-file diagnostics in `cmd/check.go`.
- Adding selector support for filesystem scopes.
- Adding a general diagnostic grouping or suppression framework.
- Changing JSON output, since `check` has no JSON diagnostics today.
- Regenerating check-type reference pages, since the descriptor surface does not
  change.

## Edge Cases

### Mixed allowed and disallowed content

If any selected file sits under a directory, that directory is not reported as a
collapsed unmatched subtree. This prevents `ongoing/` from hiding a stray
`ongoing/notes.tmp` when other `ongoing/` files are valid.

### Nested disallowed subtree

If `one-time/a.md` and `one-time/deep/b.md` are both unmatched, the report target
is `one-time/`, not `one-time/deep/`, because `one-time/` is the shallowest
directory whose subtree is entirely unmatched.

### Single unmatched file

One unmatched file remains a file-level violation, even if it is nested under a
directory with no selected files. The goal is to reduce noise, not to replace
specific file paths with less specific directory paths.

### Empty selected set

If a scope has no selected files and multiple unmatched files under a shared
directory, the shared directory can be collapsed. If unmatched files are spread
across top-level directories, each top-level directory collapses independently
when it has more than one unmatched file.

Root-level files have no ancestor directory and therefore report individually.

## Test Checklist

- Unit: `filesystem_unmatched_files` groups two or more unmatched files under a
  disallowed subtree.
- Unit: a stray unmatched file in a directory with selected files remains a
  file-level violation.
- Unit: verbose mode returns individual unmatched files.
- CLI: default `katalyst check` prints a grouped directory diagnostic for a
  disallowed subtree.
- CLI: default `katalyst check` does not also print the files hidden by the
  grouped directory diagnostic.
- CLI: default `katalyst check` still prints a stray file in an otherwise
  selected directory.
- CLI: `katalyst check --verbose` prints each unmatched file and does not print
  the grouped directory summary.
- Snapshot: `katalyst check --help` lists `--verbose` / `-v`.
- Full suite: `make test` passes.
- Build: `make build` passes.

## Graduation Notes

When this lands:

- Keep the durable behavior in tests; the tests are the executable examples for
  the grouping rule.
- Update user-facing docs only if `check --verbose` grows beyond this narrow
  unmatched-file use. The command help and generated help snapshot cover the
  shipped surface for now.
- Do not add an `AGENTS.md` convention unless another filesystem set check needs
  similar behavior. At that point, revisit whether `FileSetContext.Verbose`
  needs a documented package-level contract.
