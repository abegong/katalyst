+++
title = "Inspectors"
weight = 45
+++

# Inspectors

Why Katalyst has a read-only operation that *describes* a corpus instead of
validating it, and why that operation deliberately stops short of recommending
a schema.

## Background

A [check]({{< relref "domain-model.md" >}}) is evaluative: given an item and a
schema, it asks "does this satisfy the predicate?" and returns violations. An
**inspector** is its descriptive dual: given a directory, it asks "what is the
distribution of this aspect across the files?" and returns **evidence** —
counts and distributions, with the file count `n` as the denominator. Where a
check asks *is `status` one of {read, reading, to-read}?*, the matching
inspector reports *`status` is present in 142 of 142 files with those three
values*.

This is the `aggregate` operation from the [general
model]({{< relref "general-model.md" >}}), which the engine had listed but
never implemented. It is the missing on-ramp: Katalyst could enforce a schema
you wrote, but had nothing to help you understand a corpus you inherited.
`inspect` fills that gap — point it at a wiki and it tells you the shape the
files already have.

## Design rationale

**Evidence, not recommendations.** An inspector reports that a field appears in
94% of files; it does *not* say "make it required." The threshold that turns
94% into `required`, or a small recurring value set into an `enum`, is a
judgment call, and judgment is kept out of the measurement layer. This is the
load-bearing decision. If inspectors emitted recommendations, the threshold
policy would be baked in and un-tunable, and the evidence itself would become
something to second-guess rather than trust. By reporting only counts with a
denominator, an inspector stays trustable: the reader sees *why* a conclusion
holds and decides for themselves.

**A determinism dividing line.** What belongs in an inspector versus the reader
follows one test: deterministic measurement is an inspector's job; threshold
picking and structure proposing are not. Counting how often a field appears,
histogramming its types, grouping files by their frontmatter key-set — all
deterministic, all inspectors. Deciding that 94% is "required", that two
near-but-distinct key-sets are really one collection, or what to name a
schema — all judgment, none of it in Katalyst. The `frontmatter_shape`
inspector illustrates the seam: it groups files with *identical* fingerprints
(deterministic, so it does it) but leaves the fuzzy "these two groups are the
same collection" call to the reader.

**A division of labor.** Katalyst provides the instruments; a human or an agent
is the profiler. The intended workflow is a loop — inspect to form a hypothesis
about the schema, draft it, check it, fix the holdouts — but the forming,
drafting, and threshold-choosing live with whoever drives the tool. This keeps
Katalyst's surface small and honest: it measures and it enforces, and it does
not pretend to infer intent.

**Parse once.** `inspect` reads the directory into a `Corpus` a single time and
runs every inspector over that shared, parsed set. Inspectors are pure
functions of the corpus — they never touch disk — which keeps them
deterministic and testable, and keeps a multi-inspector run from re-reading
every file once per inspector.

**Markdown by default.** The report renders as Markdown unless `--json` is
passed. Agents read Markdown well and humans read it for free, so one format
serves both; JSON is there for callers that parse results mechanically. Both
are projections of the same evidence — one source of truth, two
serializations.

## Trade-offs and alternatives

- **A monolithic `profile` command that emits a finished schema** was rejected.
  It would bundle measurement and judgment into one opaque step and over-fit:
  the inferred schema would encode the corpus's current state, mistakes and all
  — the one typo'd field, the three fat-fingered files — as authoritative.
  Splitting measurement (inspector) from judgment (reader) is the whole point.

- **A "candidate collections" inspector** was rejected for the same reason.
  Drawing and naming collection boundaries is judgment; baking a clustering
  heuristic into Katalyst would freeze a policy that belongs to the reader. The
  deterministic part — fingerprinting files by key-set — ships as
  `frontmatter_shape`; the clustering does not.

- **Auto-applying an inferred schema** was rejected. The value is a fast,
  conservative first draft a human edits, not an authoritative inference.
  `inspect` writes nothing; authoring the `.katalyst/` files is a separate,
  deliberate act.

- **Counterfactual `check` is deferred.** The tighter loop — testing a throwaway
  candidate schema against the corpus without installing it (`check --try`) — is
  the natural companion to inspect, but it is out of scope for the initial work
  and tracked separately. Inspectors stand on their own: you draft from their
  evidence and validate with the normal `check`.

## See also

- [Inspectors reference]({{< relref "../reference/inspectors/_index.md" >}}) — the inspector set and what each reports.
- [Profile an existing wiki]({{< relref "../how-to/profile-an-existing-wiki.md" >}}) — the task recipe.
- [Domain model]({{< relref "domain-model.md" >}}) — where inspectors sit among checks, items, and collections.
