# Inspect — profiling a directory into a draft schema

> **Status: planning.** Introduces **inspectors**: a descriptive (read-only)
> operation family that measures an existing directory and returns *evidence*
> about its shape — frontmatter fields, markdown structure, filename
> conventions. Inspectors are the dual of [checks](../../docs/content/explanation/general-model.md):
> a check asserts a predicate; an inspector reports the distribution that
> predicate would be tested against. The deliverable is not a magic
> "auto-schema" button but a set of **instruments an agent drives** — combining
> `inspect` (form hypotheses) with counterfactual `check` runs (test
> hypotheses) to profile a wiki and draft a `.katalyst/` schema for it. No
> plan yet; open questions below.

## Overview

Katalyst today runs in one direction. A human authors schemas and collection
definitions under `.katalyst/`, then `check`/`fix` enforce them. There is no
on-ramp for the opposite, far more common situation: *"I have a 500-page wiki
already — what schema does it follow?"* `katalyst init` only scaffolds an
**empty** `.katalyst/`, which is a blank page for someone with an existing
corpus.

This spec adds the inverse direction — `check` run backwards. Instead of
"given a schema, which files violate it," inspectors answer "given the files,
what shape do they already have, and who are the outliers." The output is a
**draft** schema a human (or agent) reviews and edits, never a schema silently
applied.

The design has two deliberate halves, reflecting a division of labor:

1. **Inspectors** (katalyst) — deterministic measurement. Read a scope, return
   structured evidence. They never recommend; they only report.
2. **An agent** (the harness, not katalyst) — judgment. Reads inspector
   evidence to form schema hypotheses, tests each hypothesis with a
   counterfactual `check` run ("what would happen if the schema were *this*?"),
   and converges on a draft.

Katalyst provides the instruments; the agent is the profiler. This matches the
[technical spec](../../docs/content/explanation/technical-spec.md)'s framing of
katalyst as "infrastructure for AI harnesses and agentic systems" supporting
"deterministic and non-deterministic rule evaluation" — inspectors are the
deterministic instruments; the agent supplies the non-deterministic judgment.

## Value

- **Onboarding.** Turns `init` from a blank page into "here is a draft schema
  for what you already have; review it." The first run after `inspect --write`
  is `check`, and it passes on the files that already conform and lights up the
  exact outliers the evidence already flagged. That round-trip — *inspect →
  write → check → see the holdouts* — is the whole onboarding story in a few
  commands.
- **A general primitive, not a one-off.** An inspector is the descriptive
  [operation](../../docs/content/explanation/general-model.md) the model has
  been missing. The same evidence powers profiling today and, later, schema
  evolution, drift detection, and migration planning — all of which need to
  "describe the shape of this data" before they can act.
- **Trust through evidence.** Because inspectors report distributions with
  denominators rather than verdicts, the human/agent sees *why* a
  recommendation holds (`status` present in 142/142, three values) and can
  judge it. A profiler that emitted opaque conclusions would not be trusted.

## Current State

- **No descriptive operation exists.** The
  [general model](../../docs/content/explanation/general-model.md) lists
  operations (read, list, search, query, **aggregate**, diff) but the engine
  only implements the evaluative ones. There is nothing that aggregates across
  a collection to describe it.
- **The parsing substrate is already here.** `internal/frontmatter` parses a
  file into `Document{HasFrontmatter, Meta map[string]any, Body, Lines}` with
  per-pointer line tracking. `internal/project` already globs a directory and
  reports unmatched files. An inspector is mostly an aggregation pass over
  `Document`s the engine already produces.
- **`check` is single-direction and human-only.** It resolves a schema, runs
  the [18-check engine](../../docs/content/explanation/technical-spec.md)
  (object / markdown / filesystem), and prints `path:line: /pointer: message`
  plus an exit code. Two relevant capabilities already exist and are reused
  below: `--schema <path>` runs an **un-installed** schema, and the resolver
  caches compiled schemas so "check 10,000 files" costs one compile.
- **`infer`/`profile` is explicitly deferred.** `cli-spec.md` lists
  "`infer`/`profile`" and "machine-readable output (`--json`)" as out of scope
  for v0. This spec is that deferred work; it renames the verb to `inspect`.

## Design

### The inspector abstraction

An **inspector** is a read-only operation that takes a **scope** (a directory,
a collection, or a single item) and returns **evidence** — a structured,
machine-readable description of what it measured. It is the dual of a check:

| | Check (evaluative) | Inspector (descriptive) |
|---|---|---|
| Input | item **+** expectation (schema) | scope (files), no expectation |
| Asks | "is `status` one of {a,b,c}?" | "what is the distribution of `status`?" |
| Returns | `[]Violation` | evidence: `{read:80, reading:12, to-read:50}` |
| Verdict? | yes (pass/fail) | **no** |

Most of the 18 checks have a natural inspector dual — "the check minus the
expectation." That symmetry is the organizing principle: inspectors get their
own **registry** mirroring `internal/checks/registry.go`, in the same three
families (object / markdown / filesystem) plus a small structural family.

The hard rule that keeps the division of labor real:

> **Inspectors return evidence, not recommendations.** An inspector reports
> "`rating` present in 94% of files, always an integer 1–5." It does **not**
> say "→ make it required." Choosing that 94% clears a `required` bar, or that
> a small recurring value set should become an `enum`, is *judgment* and lives
> in the agent, never in the inspector.

### The determinism dividing line

What belongs in an inspector versus the agent is decided by one test:

> **Deterministic measurement → inspector. Threshold-picking and
> structure-proposing → agent.**

- Walk-and-parse, field frequency, value cardinality, type observation,
  H1 presence, filename casing → **deterministic → inspectors.**
- "Is 94% enough to call it required?", "are these two directories really one
  collection?", "what should this schema be named?" → **judgment → agent.**

This resolves the open question of whether *"candidate collections"* is an
inspector. It is **not** — boundary-drawing is judgment. But its deterministic
sub-part is: a **shape-fingerprint inspector** returns, per file or per
directory, a normalized fingerprint of the frontmatter key-set (and types). The
agent clusters fingerprints into candidate collections and names them.
Katalyst measures the fingerprints; the agent draws the boundaries.

### Inspector families (initial set)

Structural (new family):

- `walk_parse` — enumerate `*.md` under the scope; per file report
  `HasFrontmatter`, parse success/failure, body presence. Parse failures are
  themselves evidence (a file that claims no metadata).
- `frontmatter_shape` — per file (or aggregated per directory), the normalized
  set of frontmatter keys and observed scalar types: the fingerprint the agent
  clusters on.

Object / frontmatter (duals of the object checks):

- `object_field_frequency` — per key: present count, presence rate over `n`.
- `object_field_types` — per key: observed type histogram.
- `object_field_values` — per key: value cardinality, and the value set when
  small (the enum signal).
- `object_field_numeric_range` — per numeric key: observed min/max.
- `object_field_string_length` — per string key: observed length range.

Markdown (duals of the markdown checks):

- `markdown_heading_shape` — fraction with exactly one H1, fraction where H1
  matches a `title` field, whether heading levels ever jump.
- `markdown_sections` — recurring section headings and their frequency
  (the `required_section` signal).
- `markdown_code_fences` — presence and language-tag rate of fenced blocks.

Filesystem (duals of the filesystem checks):

- `filesystem_naming` — filename casing histogram (kebab / snake / spaces),
  extension histogram, common filename prefixes, directory depth.

Each inspector, like each check, has a descriptor in a registry so it is
enumerable and self-documenting, and so it cannot ship undocumented.

### Evidence (the return format)

The primary consumer is an agent, so evidence is JSON. Every record is
self-describing, scoped, and **always carries the denominator `n`** so the
consumer computes confidence itself:

```json
{
  "inspector": "object_field_frequency",
  "scope": "notes/books",
  "n": 142,
  "evidence": {
    "title":  { "present": 142 },
    "rating": { "present": 133 },
    "status": { "present": 142 }
  }
}
```

```json
{
  "inspector": "object_field_values",
  "scope": "notes/books",
  "n": 142,
  "evidence": {
    "status": { "cardinality": 3, "values": {"read": 80, "reading": 12, "to-read": 50} }
  }
}
```

Properties the format must hold: machine-readable; names the inspector and
scope; reports observations not conclusions (no `→ required` arrows — those
were a mistake in early sketches); composable so multiple inspector records
combine into one profile. A run over a scope emits one record per inspector.

### The `inspect` command

```
katalyst inspect <path> [--inspector <name> ...] [--json]
```

- Reads the scope, runs the selected inspectors (default: all), returns
  evidence. **Writes nothing** — `inspect` is a diagnosis, never a mutation.
- `--json` emits the evidence records above (the agent path). Without it, the
  command renders the same records as a human-readable profile (the agent-less
  path) — frequencies, detected conventions, outliers. **One source of truth:**
  the human report is a thin renderer over the same evidence the agent gets.
- Note what is **absent**: no `--strictness`/`--threshold` flag. Thresholds are
  recommendation policy, and recommendations are the agent's job, not the
  command's. `inspect` only measures.

Illustrative human rendering (a *projection* of evidence, not new data):

```
notes/books/  (142 files, 142 parsed, 0 errors)
  frontmatter
    title    142/142  string
    author   141/142  string
    rating   133/142  integer (1–5)
    status   142/142  string  {read, reading, to-read}
    isbn      17/142  string
  markdown
    single H1            142/142
    H1 == title          138/142
    "## Review" section   129/142
  filesystem
    kebab-case names     142/142
    spaces in path         3      ⚠
  outliers
    notes/books/Dune Messiah.md   spaces in filename; missing 'author'
```

### Counterfactual `check` (the agent's other instrument)

Profiling is a loop: *inspect* to form a hypothesis, then test the hypothesis
by running a **throwaway** schema and seeing what breaks. "What would happen if
the schema were this?" Three extensions to `check` make that possible; the
first capability half-exists already (`--schema <path>` runs an un-installed
schema):

1. **Structured output** (`check --json`). Today `check` emits human text and
   an exit code. The agent needs machine-readable results: per-item
   pass/fail and the violations, plus an aggregate.
2. **Ephemeral full-config dry-run.** `--schema` only swaps the *object*
   schema. The counterfactual needs to test a whole candidate spec (object +
   markdown + filesystem checks) against a `<path>` that is **not yet a
   registered collection**, writing nothing. (Flag spelling is an open
   question — e.g. `check --try <collection-def-file> <path>`.)
3. **Holdouts, not just counts.** "139/142 pass" is far weaker signal than
   *which 3 fail and why*. The refinement loop lives in the holdouts: they tell
   the agent whether to tighten the schema or flag genuinely bad files. Per-item
   results (extension 1) already carry this; it is called out because it is the
   point, not an incidental.

### The agent workflow this enables

1. `inspect <path> --json` → evidence for every candidate collection
   (clustered from `frontmatter_shape` fingerprints).
2. Agent drafts a candidate `.katalyst/schemas/*` + `collections/*` from the
   evidence, applying its own thresholds.
3. `check --json --try <draft> <path>` → per-item holdouts.
4. Agent inspects holdouts: tighten the draft, loosen a field to optional, or
   leave outliers to be fixed. Repeat 2–4 until satisfied.
5. `inspect --write` (or the agent writes the files) materializes the draft;
   the user reviews the diff and runs the real `check`.

Two cautions shape the loop:

- **Inspect-heavy, check-light.** If the agent brute-forces dozens of schema
  variants over thousands of files, the loop is slow and expensive. Evidence
  must be rich enough that the agent forms *good* hypotheses up front and needs
  few counterfactual rounds. Inspectors exist to *reduce* the search, not just
  enable it.
- **Determinism + caching.** `walk_parse` should parse the corpus once and
  cache `Document`s; repeated counterfactual checks re-evaluate predicates
  against parsed evidence rather than re-reading disk. This also keeps
  inspectors pure and testable.

### Domain-model impact

- **New concept: inspector.** A descriptive operation; the dual of a check.
  Added to the [general model](../../docs/content/explanation/general-model.md)
  (it realizes the long-listed `aggregate` operation) and the
  [glossary](../../docs/content/reference/glossary.md).
- **`check` gains a machine-readable, collection-less mode.** Reconciles with
  the [domain model](../../docs/content/explanation/domain-model.md) lifecycle
  of `check`, which currently assumes a loaded config and human output.
- **Out-of-scope list updates.** `cli-spec.md` moves `infer`/`profile` and
  `--json` from "out of scope" into this spec (under the name `inspect`).

## Open Questions

- **Verb name.** `inspect` (this spec) vs the `cli-spec.md` placeholder
  `infer`/`profile`. `inspect` reads as descriptive ("look and report") rather
  than generative ("produce a schema"), which matches "evidence, not verdicts."
  Leaning `inspect`; not locked.
- **Counterfactual flag surface.** How to pass an ephemeral full-collection
  definition to `check` (`--try <file>`? `--config`? stdin?) and how it
  composes with `--schema`. Needs a concrete grammar.
- **Fingerprint granularity.** Does `frontmatter_shape` fingerprint on key-set
  only, or key-set + types? Types catch "same keys, different meaning"; key-set
  alone is cheaper and clusters more aggressively. Probably key-set first,
  types as a second signal.
- **Where clustering lives.** Pure agent reasoning over fingerprints, or a thin
  deterministic `--cluster` helper that proposes groupings the agent accepts or
  overrides? The determinism line says agent, but a deterministic *suggestion*
  might cut agent rounds.
- **`--write` target.** Does `inspect --write` exist at all, or is writing
  purely the agent's job (it has the judgment)? If it exists, what does it emit
  when the corpus is multi-collection?
- **Evidence schema versioning.** The evidence format is a public contract for
  agents; it likely needs a version field from day one.

## Rejected alternatives

- **A monolithic `profile` command that emits a finished schema.** Bundles
  measurement and judgment into one opaque step. It would encode the corpus's
  current state — mistakes and all (the one typo'd `tag`, the three
  fat-fingered files) — as authoritative schema, and over-fit. Splitting
  measurement (inspector) from judgment (agent) is the whole point.
- **Inspectors that emit recommendations (`→ required`).** Tempting and in an
  early sketch, but it smuggles threshold policy into the measurement layer,
  making thresholds un-tunable and the evidence un-trustable. Inspectors report;
  the agent decides.
- **A "candidate collections" inspector.** Boundary-drawing is judgment, not
  measurement; baking a clustering heuristic into katalyst freezes a policy that
  belongs in the agent. Ship the deterministic `frontmatter_shape` fingerprint
  instead and let the agent cluster.
- **Reusing `check` alone, no inspector layer.** `check` can only answer
  "does this specific schema fit?" Profiling needs "what is *here* before I have
  any schema?" — a question `check` structurally cannot ask. The descriptive
  operation has to exist.
- **Auto-applying the inferred schema.** The value is a fast, conservative
  *first draft a human edits*, not an authoritative inference. Auto-apply
  re-introduces the over-fitting risk and destroys trust. `inspect` writes
  nothing by default.

## Test checklist (what the pending tests will assert)

Inspectors:
- [ ] `walk_parse` counts files, reports per-file parse success/failure
- [ ] a file with no frontmatter is reported as such, not skipped
- [ ] `object_field_frequency` present-counts match `n` and the corpus
- [ ] `object_field_values` reports cardinality and value set below a size cap
- [ ] `object_field_types` reports a mixed-type key as mixed, not first-wins
- [ ] `markdown_heading_shape` rates single-H1 and H1==title correctly
- [ ] `markdown_sections` surfaces a section present in most files
- [ ] `filesystem_naming` histograms casing and flags spaces/extensions
- [ ] `frontmatter_shape` yields identical fingerprints for same-key files

Evidence format:
- [ ] every record carries `inspector`, `scope`, `n`, `evidence`
- [ ] no record contains a recommendation/verdict field
- [ ] records from multiple inspectors compose into one profile

`inspect` command:
- [ ] writes nothing (read-only); exit 0 on a readable scope
- [ ] `--json` emits evidence records; default emits the human projection
- [ ] human projection is derived from the same evidence (one source of truth)
- [ ] `--inspector` narrows to the named inspectors

Counterfactual `check`:
- [ ] `check --json` emits per-item pass/fail + violations + an aggregate
- [ ] ephemeral candidate config runs against an unregistered `<path>`,
      writes nothing, mutates no `.katalyst/`
- [ ] per-item holdouts identify exactly the failing files and reasons
- [ ] parsed documents are reused across repeated counterfactual runs (cache)
