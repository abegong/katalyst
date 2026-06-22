# Semantic checks — Tier 4

> **Status: planning.** Checks that judge *meaning*, delegated to an LLM —
> "this section summarizes that file," "the description matches the body,"
> "these tags fit the content." A deliberate, separate product surface: it
> breaks the determinism, offline-execution, and zero-config assumptions every
> existing check relies on.

## Overview

A semantic check states an assertion about meaning and asks a model to judge it.
The motivating example: a `## Summary` section in `index.md` should be a
faithful summary of `report.md`. The check sends both, asks the model whether
the assertion holds, and turns the structured verdict into a violation.

This is a different kind of check from everything in Tiers 1–3. Those are pure
functions: fast, offline, deterministic, reproducible in CI. A semantic check is
slow, networked, billable, and non-deterministic. Treating it like a normal
check would poison those properties for the whole engine, so it ships as an
**opt-in pass** behind a flag, with its own config block, caching, and
failure semantics.

## Value

No other Katalyst check can assert that two pieces of prose *mean* the same
thing, that metadata honestly describes a body, or that a document is internally
consistent. It turns Katalyst from a structural linter into a hybrid
static/semantic one — genuinely novel territory for a frontmatter linter. The
bet is real because the cost is real; this spec exists to scope it so the
default `check` stays free and deterministic.

## Current state

Every check is a pure `Run(Context) []Violation` (`internal/checks/checks.go`):
synchronous, offline, no credentials, no per-check config beyond static fields.
`check` is exit-code-driven and reproducible — invariant for CI use
([validate in CI](../../docs/content/how-to/validate-in-ci.md)).
There is no network access, no secret handling, no model configuration anywhere
in the codebase, and the repo commits to no LLM provider today. The domain model
is explicit that every run is stateless with no cache file.

Semantic checks violate all of these. The design's job is to contain the blast
radius.

## Design

### A separate pass, not a normal check

`Run` is synchronous and returns `[]Violation`; LLM calls are slow, fallible,
and async. Rather than contort the `Check` interface, run semantic checks as a
distinct phase:

- Default `check` runs Tiers 1–3 only — unchanged: deterministic, offline, free.
- `katalyst check --semantic` (or `check semantic`) additionally runs the
  semantic pass. Without the flag, semantic checks are parsed and validated but
  not executed.

This keeps CI green-by-default and makes the cost explicit at the call site.

### Provider and model config

Add a `semantic:` block to `.katalyst/config.yaml` (settable per the existing
config-discovery model):

```yaml
semantic:
  provider: anthropic
  model: claude-opus-4-8   # default; latest, most capable Claude model
  api_key_env: ANTHROPIC_API_KEY
  max_calls: 200           # per-run budget guard
```

Default to Claude via the official Go SDK (`github.com/anthropics/anthropic-sdk-go`),
calling `claude-opus-4-8`. Credentials come from the environment, never config.
The provider is an interface so a second backend can be added later, but v1 is
Claude-only (see Rejected alternatives).

### The semantic checks (catalog)

| Kind | Judges |
|---|---|
| `semantic_section_summarizes` | A named section (`heading`) is a faithful summary of another file (`target`). The motivating case. |
| `semantic_field_matches_body` | `field` (e.g. `description`) accurately reflects the body. |
| `semantic_tags_appropriate` | `field` (a list, e.g. `tags`) matches the topics the body covers. |
| `semantic_no_contradiction` | Body doesn't contradict a frontmatter claim (e.g. `status: published` with draft/TODO language). |
| `semantic_reading_level` | Body matches a target tone/reading level (`level`). |

`semantic_section_summarizes` and `semantic_field_matches_body` both need
cross-document or cross-region content — they compose with the Tier 3
`ProjectView` to resolve `target`.

### Determinism, caching, and CI stability

A linter must be stable enough to gate a build. Three levers:

- **Structured verdict.** The model returns a structured `{pass, reason}` (via
  the SDK's structured-output support), not free text — `reason` becomes the
  violation message. Use adaptive thinking; no sampling knobs to tune.
- **Content-hash cache.** Key each result on a hash of (check kind, prompt
  template, input content). Re-running over unchanged files is free and yields
  identical verdicts. This reintroduces *persisted derived state*, which the
  domain model lists as out of scope — call it out explicitly: the semantic
  cache is the one sanctioned cache file, lives under `.katalyst/`, and is
  opt-in with the semantic pass. It does not affect deterministic checks.
- **Budget guard.** `max_calls` caps spend per run; exceeding it is a usage
  error, not a silent partial pass.

### Failure semantics

Network errors, timeouts, and rate limits are *infrastructure* failures, not
content violations. They must not be reported as `path:line:` violations.
Default: an infra failure fails the run with exit 2 (usage/IO), distinct from
exit 1 (check failure). A `--semantic-soft-fail` flag downgrades infra failures
to warnings for environments that want best-effort semantic linting.

### Trust and privacy

Running a semantic check sends document content to an external provider.
Document this prominently: content leaves the machine, may be retained per the
provider's policy, and the feature should be used deliberately on private repos.
This is a publish-style side effect and belongs in the feature's how-to page,
not buried in config reference.

## Open questions

1. **Sync vs. async engine phase.** The semantic pass can run checks
   sequentially (simple, slow) or fan out concurrently (fast, needs rate-limit
   handling and bounded concurrency). _Leaning: bounded concurrency from the
   start — semantic latency makes sequential impractical past a handful of
   items — but confirm the complexity is worth it for v1._
2. **CI verdict: block or warn by default?** Should a semantic failure under
   `--semantic` exit 1 (block the build) or warn? Structured output + caching
   make verdicts stable, but they're still model judgments. _Leaning: block, so
   the flag is meaningful in CI; offer `--semantic-soft-fail` for the cautious._
3. **Cache location and the stateless invariant.** Confirm the
   `.katalyst/`-resident semantic cache is acceptable as the documented
   exception to "every run is stateless," and define its invalidation (hash
   covers content + prompt; does a model/version change bust it? — yes, fold
   `model` into the key).
4. **Provider abstraction depth for v1.** Ship a thin Claude-only client, or a
   provider interface with one implementation? _Leaning: interface with one impl
   — the seam is cheap now and expensive to retrofit — but keep the interface
   minimal (one `Judge(ctx, prompt) (Verdict, error)` method)._

## Rejected alternatives

- **Embedding LLM calls in the normal check pass.** Couples cost, latency, and
  non-determinism to every `check` run and breaks CI reproducibility. The
  opt-in pass keeps the default engine pure.
- **Reporting infra failures as violations.** Conflates "the network is down"
  with "the summary is wrong." Distinct exit codes keep the two legible to CI.
- **Free-text model output parsed heuristically.** Brittle and unstable across
  runs. Structured output is the only path to CI-gradeable verdicts.
- **A provider-neutral abstraction with multiple backends in v1.** Premature —
  it multiplies prompt-tuning and testing surface before there's a second
  provider in demand. One backend behind a one-method interface.
