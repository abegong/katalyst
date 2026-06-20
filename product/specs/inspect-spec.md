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
  for what you already have; review it." Once the agent has written the draft,
  the next run is `check`: it passes on the files that already conform and
  lights up the exact outliers the evidence already flagged. That round-trip —
  *inspect → agent drafts a schema → check → see the holdouts* — is the whole
  onboarding story in a few commands.
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

This resolves whether *"candidate collections"* is an inspector. It is
**not** — drawing and naming collection boundaries is judgment. But the
deterministic parts are fair game for an inspector: a **`frontmatter_shape`
inspector** returns per-file fingerprints of the frontmatter key-set (and
optionally types), and it may go further and **group files that share an
identical fingerprint** — that grouping is deterministic, so it belongs to the
inspector. What stays with the agent is the *judgment*: deciding that two
near-but-not-identical groups are really one collection, where the boundary
falls, and what to name it.

So aggregation and clustering can be a capability an **individual inspector**
owns; they are **not** a behavior the `inspect` command imposes on every
inspector, and the fuzzy boundary calls remain the agent's.

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

Evidence is an internal structure each inspector produces; `inspect` then
*renders* it. **Markdown is the default rendering** — agents handle Markdown
well and humans read it for free, so one format serves both. `--json` emits the
same evidence as a structured object for callers that want to parse it
mechanically. Neither is more "real" than the other: both are projections of
one underlying evidence value — one source of truth, two serializations.

Every record names its inspector and scope and **always carries the denominator
`n`**, so the consumer computes confidence itself rather than trusting a
baked-in threshold. The `--json` form of two records:

```json
{ "inspector": "object_field_frequency", "scope": "notes/books", "n": 142,
  "evidence": { "title": {"present": 142}, "status": {"present": 142} } }
{ "inspector": "object_field_values", "scope": "notes/books", "n": 142,
  "evidence": { "status": {"cardinality": 3,
                "values": {"read": 80, "reading": 12, "to-read": 50}} } }
```

The default Markdown rendering of the same run is shown in
[The `inspect` command](#the-inspect-command) below. Properties either
rendering must hold: names the inspector and scope; carries `n`; reports
observations not conclusions (no `→ required` arrows — those were a mistake in
early sketches); composable, so records combine into one profile. A run over a
scope emits one record per inspector.

### The `inspect` command

```
katalyst inspect <path> [--inspector <name> ...] [--json] [-o <file>]
```

- Reads the scope, runs the selected inspectors (default: all), and renders
  their evidence. **Writes no schema** — `inspect` is a diagnosis, never a
  mutation of the project. Drafting `.katalyst/` files from the evidence is the
  agent's job (see workflow below).
- **Default output is Markdown**; `--json` emits the same evidence as a
  structured object. One source of truth, two serializations.
- `-o <file>` saves the rendered report to a file. Pure convenience — the bytes
  are identical to stdout (equivalent to a shell redirect), not a separate
  artifact.
- Note what is **absent**: no `--strictness`/`--threshold` flag and no
  `--write`. Thresholds are recommendation policy and writing a schema is a
  judgment call; both belong to the agent, not the command. `inspect` only
  measures and reports.

The default Markdown rendering groups evidence by family (a *projection* of the
records, not new data):

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
   registered collection**, writing nothing. The candidate must be
   **self-contained** — schema inline or by path, never by name, since nothing
   is installed to resolve a name against. Grammar: `check --try <def> <path>`;
   `--try -` reads the candidate from stdin so the agent needn't write temp
   files; `--try` and `--schema` are mutually exclusive (a candidate already
   supplies its own object check).
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
5. The agent writes the draft `.katalyst/` files; the user reviews the diff and
   runs the real `check`. (`inspect` itself never writes a schema.)

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

Resolved (folded into the design above):

- **Verb name → `inspect`.** Reads as descriptive ("look and report") rather
  than generative, matching "evidence, not verdicts." Supersedes the
  `cli-spec.md` `infer`/`profile` placeholder.
- **Default output → Markdown, `--json` optional.** Agents handle Markdown well
  and humans read it for free; there is no reason to make JSON the default. Both
  render the same evidence.
- **No `--write`.** `inspect` never writes a schema — that is the agent's
  judgment call. A `-o <file>` convenience may save a copy of the report, but
  that is just the rendered output, not a new artifact.
- **Evidence versioning is out of scope.** Versioning the evidence contract
  drags in versioning the schema format, the config, and the check registry
  with it — a separate, larger decision. Defer the whole question; do not add a
  version field now.
- **Clustering can live inside an inspector.** Deterministic grouping (files
  with identical `frontmatter_shape` fingerprints) is a capability the
  individual inspector may own; only fuzzy boundary-drawing is reserved for the
  agent. It is not a property of the `inspect` command.

- **Counterfactual flag grammar → `check --try <def> <path>`.** `--try -` reads
  the candidate from stdin; `--try` and `--schema` are mutually exclusive (the
  candidate already carries its object check). `--try` over `--as` because it
  names the throwaway-hypothesis intent. A rename is mechanical if it flips.
- **Fingerprint identity → key-set.** `frontmatter_shape` fingerprints on the
  sorted set of frontmatter keys; observed per-key types ship as *adjacent*
  evidence but are not part of the grouping identity. Key-set is cheaper and
  clusters more aggressively, and the agent has the types alongside if it wants
  to split a group. (Internal to that one inspector, not a command property.)
- **Initial inspectors are parameterless.** No descriptor options in v1; the
  only aggregation is `frontmatter_shape`'s identical-fingerprint grouping,
  hardcoded in that inspector. A parameter mechanism (like checks' `field:`) is
  deferred until an inspector needs one.

Still open: _None._

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
- [ ] writes no schema/project files (read-only); exit 0 on a readable scope
- [ ] default output is Markdown; `--json` emits the same evidence as JSON
- [ ] both renderings derive from the same evidence (one source of truth)
- [ ] `-o <file>` writes bytes identical to stdout
- [ ] `--inspector` narrows to the named inspectors

Counterfactual `check`:
- [ ] `check --json` emits per-item pass/fail + violations + an aggregate
- [ ] ephemeral candidate (`--try <def>`) runs against an unregistered `<path>`,
      writes nothing, mutates no `.katalyst/`
- [ ] `--try -` reads the candidate definition from stdin
- [ ] a candidate referencing a schema by name (not inline/path) is rejected
- [ ] per-item holdouts identify exactly the failing files and reasons
- [ ] parsed documents are reused across repeated counterfactual runs (cache)
