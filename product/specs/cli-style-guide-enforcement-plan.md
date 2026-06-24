# Plan - CLI style guide enforcement
> Issue: #37

## Current State

`cmd/AGENTS.md` already carries most of the style guide content:

- the two CLI grammars, blessed verbs and resource nouns;
- root help ordering and copy rules;
- terminal readout layout;
- user-facing error message grammar;
- exit-code and stream contracts;
- the split between snapshot tests and behavior tests.

Recent CLI work also added enforcement pieces:

- help and readout snapshots under `cmd/testdata/snapshots/`;
- shared readout helpers in `cmd/list_format.go`;
- dogfooded `.katalyst` checks for snapshot hygiene and readout layout.

The remaining work is to make this an explicit, complete style guide and to
close the enforcement gaps that are still only conventions.

## Goal

Finish #37 by making the CLI standard easy to find, complete enough to guide new
commands, and automatically checked where the rules are mechanical.

## Implementation Steps

### 1. Consolidate the Style Guide

Review `cmd/AGENTS.md`, `cmd/organization.md`, `cmd/testdata/AGENTS.md`, and
`docs/content/reference/cli.md`.

Decide whether the guide should remain entirely in `cmd/AGENTS.md` or whether
`cmd/style-guide.md` should become the canonical long-form guide with
`cmd/AGENTS.md` summarizing local rules and linking to it.

The guide must cover:

- command grammar and top-level placement;
- command, subcommand, flag, and resource naming;
- human vs. machine output;
- stdout vs. stderr;
- errors and exit codes;
- help text and examples;
- selector syntax;
- snapshot vs. property test expectations.

### 2. Audit the Current CLI Surface

Walk the Cobra tree and compare every top-level command and direct subcommand
against the guide:

- verbs vs. nouns;
- parent commands with no default action;
- `Short` text shape and punctuation;
- `Long` help structure where present;
- list/show/get output layout;
- `--json` behavior;
- selector wording and usage strings;
- standard arity and usage errors.

Record any intentionally deferred cleanup in #27 rather than hiding it in the
style-guide PR.

### 3. Add Mechanical Enforcement

Prefer dogfooding where the rule naturally fits Katalyst checks:

- keep snapshot hygiene checks in `.katalyst`;
- add checks for new snapshot directories when new CLI surfaces are covered;
- enforce stable text layout patterns only for outputs that are meant to use the
  readout contract.

Use Go tests where the rule depends on the Cobra command tree:

- no ungrouped top-level commands;
- resource noun parents do not run a default action;
- top-level help order stays intentional;
- every resource noun has `list` or a documented reason not to;
- every human-facing text surface has a snapshot unless it is explicitly a raw
  or machine-oriented surface.

### 4. Wire Verification into CI

Confirm CI already runs the relevant checks. If not, extend the existing CI path
so style-guide violations fail in the normal test/check suite.

The target verification set is:

```bash
./bin/katalyst check
go test ./cmd
```

Broaden only if the implementation touches shared packages outside `cmd`.

## Acceptance Checklist

- [ ] The repo has one clear CLI style-guide source of truth.
- [ ] The guide covers grammar, naming, output, errors, exit codes, help text,
      flags, and selectors.
- [ ] Mechanical rules are enforced by dogfooded Katalyst checks or focused Go
      tests.
- [ ] Any remaining non-mechanical cleanup is tracked in #27.
- [ ] CI runs the enforcement path.
