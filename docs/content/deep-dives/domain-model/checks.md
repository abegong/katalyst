+++
title = "Checks"
weight = 45
+++

# Checks

A **check** asserts one condition on an item and reports a violation when the
item fails it. The check engine turns a collection's configuration into a
verdict on each item: it resolves the checks that apply, runs them, and
collects their violations. This page explains the model the engine is built on,
how check libraries supply and run checks, and why the pieces are shaped the way
they are. For the per-type catalog see the [check types
reference]({{< relref "../../reference/check-types/_index.md" >}}); for the
end-to-end data flow of one `check` invocation see the [domain
model]({{< relref "_index.md" >}}).

## Terms

| Term | Meaning |
|---|---|
| **Check** | Shorthand for a check instance when context is unambiguous. A check asserts one condition and reports a violation when the condition fails. |
| **Check type** | The reusable definition of a constraint: `object_required_field`, `markdown_single_h1`, and so on. A check type is selected by its `kind:` id and appears in the generated check types reference. |
| **Check instance** | One configured check attached to a collection or filesystem scope: a check type plus its arguments, written as one YAML object under `checks:`. The type is the rule; the instance is the rule applied here. |
| **Family** | The kind of source data a check type reads: `structuredObject` (frontmatter), `markdownBodyText` (the body), `fileSystem` (names and paths), or `plainText` (raw body text). |
| **Attachment target** | The config site where a check instance is valid: `collection` for collection `checks:`, or `filesystem` for base-level `filesystemChecks`. |
| **Check library** | The provider that supplies and runs a check type. Native libraries wrap hand-written checks; schema-backed libraries delegate to an external validation engine. |
| **Runtime granularity** | The level where a check runs. Most checks are file-scoped; a few are file-set-scoped and reason across every selected file. |
| **Severity** | The consequence of a violation. `error` fails the run; `warning` is advisory and does not change the exit code. |
| **Violation** | One failed check result, with a message, source location, JSON pointer when applicable, severity, and sometimes a sibling file for collection-scoped findings. |

Family, library, attachment target, and runtime granularity are separate axes.
Family answers *what data does this check read?* Library answers *who runs it?*
Attachment target answers *where can a user configure it?* Runtime granularity
answers *does it run once per file or once per selected file set?* A single
family can span libraries:
`structuredObject` includes both `object` from the JSON Schema library and
`object_required_field` from the native structured-object library.

The registry is the single source of truth for check types. Each check type
self-registers a `Descriptor` (its id, family, targets, docs metadata) and a
constructor. `cmd/engine` builds the runnable list by registry lookup; the docs
generator and `katalyst check-types list` read the same descriptors. A parity
test fails if a configured kind has no descriptor, so a check type cannot ship
undocumented.

## Attachment targets

Collection-attached checks live under a collection's `checks:` list. They run
after selector resolution and can use schema precedence, variants, and the
collection's full sibling set. This is the historical model and remains the
right place for rules that depend on collection identity.

Filesystem-attached checks live under a filesystem base's `filesystemChecks`
list. Each scope selects raw files with `include` and `exclude` globs. A
no-selector `katalyst check` runs filesystem scopes before collection checks,
so a project can enforce path policy before any collection exists. Explicit
collection selectors stay collection-only.

The descriptor's `Targets` list controls where a check type may be configured.
During migration, an empty list means `collection`. File-system-family checks
that only read paths can usually support both targets. Checks that read
frontmatter or body text declare that they need a parsed document so filesystem
scopes parse lazily and path-only scopes avoid unnecessary document work.

## Check libraries

A **CheckLibrary** is the provider behind a check type. It is the seam that lets
Katalyst delegate work to engines it does not implement itself.

Libraries come in two kinds:

- **Native libraries** wrap hand-written checks: `filesystem`, `plaintext`,
  `markdownbodytext`, `structuredobject`. Their logic is Go code in the
  repository.
- **Schema-backed libraries** delegate to an external engine that compiles a
  named **schema** and runs items against it. JSON Schema
  (`internal/checks/jsonschema`) is the first and currently only one; a prose
  linter such as Vale is the next candidate.

A schema-backed library does three things: it compiles a schema from source
bytes, runs an item's data against the compiled schema, and maps the engine's
findings back into Katalyst violations. The JSON Schema library wraps
`santhosh-tekuri/jsonschema/v6` behind this contract, which keeps the rest of
the codebase off the underlying library, lets the input normalize from
YAML-native types into the JSON shape the validator expects, and flattens the
library's nested error tree into a flat violation list with source lines.

**Schema is the Katalyst concept, not the JSON Schema document.** A schema
defines a collection's shape; JSON Schema is one way to express it, a Vale style
config is another. Schemas are stored flat under `.katalyst/schemas/` and named
by filename stem. The library that compiles a given schema is resolved at the
binding site, from the referencing check type's `kind`: `kind: object` resolves
its schema through the JSON Schema library. There is no per-library directory
and no migration when a second library arrives.

**Availability is a hard error.** A library reports whether it can run. An
in-process library is always available; an out-of-process tool probes for its
binary and an acceptable version. The engine checks availability before running
a library's checks, and a missing or too-old tool fails the run rather than
silently skipping enforcement, so a misconfigured CI fails loudly.

**Libraries are in-process or out-of-process.** JSON Schema runs in the same
process. An out-of-process library shells out to a separate binary, which adds
the availability concern above and a performance one: launching a process per
item is wasteful, so a batched run over a whole collection (mapping findings
back to files by name) is the planned optimization, tracked in
[#68](https://github.com/abegong/katalyst/issues/68). The first cut runs per
item, the simplest correct path.

## Running a check

Per item, the engine resolves which checks apply, then runs them.

Resolution starts from the collection's configured checks and adds the checks of
the first [variant]({{< relref "../../reference/configs/variants.md" >}}) whose `when`
predicates the item's metadata satisfies. The object schema is selected by a
precedence the JSON Schema library owns (a forced `--schema`, then an inline
`schema:` directive, then the collection's object checks); see the [domain
model]({{< relref "_index.md" >}}) for the precedence table and the
full per-item lifecycle. Before any schema compiles, the engine confirms the
owning libraries are available.

Running is uniform: every file check returns a list of violations, which the
engine concatenates; file-set checks run in a second pass over the selected
files. For collections, that file set is the whole collection. For filesystem
checks, it is the scope's selected file set plus its unmatched files. A
violation carries a JSON-pointer `Path`, a `Message`, a source `Line`, an
optional `File` (for file-set findings that name a sibling), and a `Severity`.
An item with no violations prints `path: OK`.

## Design rationale

**Self-registration over a central dispatch.** A check type owns its descriptor
and constructor in one file. The registry is assembled from those `init()`
calls, wired in by a single blank-import aggregator. The cost is that the import
must be present for the catalog to populate; the benefit is that adding a check
type never edits a shared switch, and the parity test keeps the registry and the
config grammar in lockstep.

**Family and library are separate axes on purpose.** Earlier the `object` check
was filed only by its family and its provider was implicit. Splitting provenance
(library) from source-data kind (family) lets a second engine supply a check
type into an existing family without disturbing the native checks there: a
prose linter joins `markdownBodyText` beside the native markdown checks. Family
stays the documentation and routing axis; library stays the execution axis.

**A thin wrapper over each external engine.** The codebase depends on the
`Schema` interface, not on `santhosh-tekuri/jsonschema`. That keeps the engine
swappable, isolates input normalization and error flattening to one place, and
gives every library the same shape, so the engine compiles, caches, and gates
them identically regardless of whether the work happens in-process or in a
subprocess.

**File-set checks re-scan the whole collection when attached to a collection.** A uniqueness or
required-index verdict is only correct against every item, so these checks run a
second pass over the full collection even under a single-item selector. The
trade is that `katalyst check notes/one.md` does more work than its name
suggests; the alternative (a per-item approximation) would report wrong answers.

## Trade-offs and alternatives

**One registry, not two.** Check types and their libraries share a single
registry; every check type names its owning library through its descriptor. A
separate library registry was rejected: it would split "which check types exist"
from "who owns them" and force the engine to consult both.

**Flat schemas, not per-library directories.** Namespacing schemas under
`.katalyst/schemas/<library>/` looked tidier but would migrate every existing
schema path for no functional gain, since the binding's `kind` already names the
library. Schemas stay flat.

**Per-item invocation first, batching later.** Running an out-of-process tool
once per item is slow at scale but simple and correct. Batching a collection
into one invocation is the optimization, deferred to
[#68](https://github.com/abegong/katalyst/issues/68) rather than built before a
real out-of-process library exists.

## Invariants

1. **The registry is authoritative.** Every runnable check type has a
   descriptor, and generated docs read the same registry as the engine.
2. **Family and library stay separate.** Family describes the data a check
   reads; library describes the provider that runs it.
3. **Collection-attached file-set checks see the whole collection.** A selector
   may narrow output, but a collection-level verdict still needs the full
   sibling set.

## See also

- The [check types reference]({{< relref "../../reference/check-types/_index.md" >}})
  for the precise per-type surface, generated from the registry.
- The [domain model]({{< relref "_index.md" >}}) for the per-`check`
  lifecycle, the schema resolver, and the validation result.
- The [glossary]({{< relref "../../reference/glossary.md" >}}) for the canonical
  terms (check type, check instance, CheckLibrary, schema, violation).
- The [base]({{< relref "base.md" >}}) for the collection and item
  identities checks run against, and the inspector that is a check's descriptive
  dual.
- `go doc ./internal/checks` for the code-level engine contract.
