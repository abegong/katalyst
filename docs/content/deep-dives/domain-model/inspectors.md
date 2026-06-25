+++
title = "Inspectors"
weight = 46
+++

# Inspectors

An **inspector** profiles content and returns *evidence*: counts and
distributions, never recommendations. Inspectors are the descriptive dual of
[checks]({{< relref "checks.md" >}}) - a check asserts a predicate and reports
violations; an inspector reports the distribution that predicate would be tested
against. They drive the [`inspect`]({{< relref "../../reference/cli.md" >}})
command. For the per-inspector catalog see the [inspectors
reference]({{< relref "../../reference/inspectors/_index.md" >}}); this page is the
model and the rationale behind it.

## Two layers

Inspectors come in two layers, distinguished by *how they reference the data*:

- **The raw base layer** (`SourceInspector` over a `SourceView`) measures a
  base directly, before any collection configuration, addressed by
  base-native reference (a relative path today). It answers "what is in this
  base?" - the onboarding case. `file_tree`, `file_tree_content`, and
  `document_shape` live here.
- **The collection layer** (`CollectionInspector` over a `CollectionView`)
  measures a configured collection's items, addressed by domain identity
  (collection + item id) and reached through the project's
  `CollectionDefinition`, never a raw path. `object_fields` and `markdown_body`
  live here.

The two are **distinct interfaces, not one type at two scopes**, precisely
because they reference the data through different machinery. This mirrors the
seam in the [base]({{< relref "base.md" >}}).

## Built from primitives

Most measurement lives in three reusable, layer-agnostic primitives, so the
inspectors themselves are thin wrappers that point a primitive at an input:

- **`objectFields`** - a data dictionary over a set of object maps: per field,
  presence, a type histogram, scalar cardinality, and an enum-candidate value
  set. String and numeric scalars are kept distinct; arrays and nested objects
  are typed but not yet characterized.
- **`markdownBody`** - heading-shape and recurring-section facets over a set of
  bodies.
- **`fileMetadata` and `fileTree`** - path-level conventions and filesystem
  shape (types, naming, depth, regions, directory density) over references,
  opening no files.

The same small primitives are reused where the layer makes sense, but raw base
inspectors avoid proposing collections. They report store and content facts; a
human or agent decides what collection boundaries those facts imply.

## Evidence, not recommendations

An inspector reports that a field appears in 94% of items; it does **not** say
"make it required." The threshold that turns 94% into a required field, or a
small recurring value set into an enum, is a judgment call kept out of the
measurement layer.

This is the load-bearing decision. If inspectors emitted recommendations the
threshold policy would be baked in and un-tunable, and the evidence itself would
become something to second-guess rather than trust. Reporting only counts, with
the unit count `n` as denominator, keeps the evidence trustable: the reader sees
why a conclusion holds and decides.

## The determinism dividing line

Deterministic measurement is an inspector's job; threshold-picking and
structure-proposing are not. Counting field presence, histogramming types,
mapping filesystem regions, and summarizing selected-file content structure are
all deterministic, all inspectors. Deciding that 94% is "required", that a
directory should be a collection, or what to name a schema are all judgment,
none of it here.

## Keeping output small

`file_tree` and `file_content_shape` keep Markdown output small with
deterministic caps: small trees get an actual tree; content-shape reports show
the selected file set, dominant structures, and compact text/tabular/tree
facets, with `-v` for expanded evidence.

## Output

Evidence renders as Markdown by default and JSON under `--json`; both are
projections of the same values. A single `inspect` run is one layer's
inspectors, rendered together. `inspect` writes no schema and mutates nothing.

## Division of labor

Katalyst provides the instruments; a human or an agent is the profiler. The
intended workflow is a loop - inspect, draft a schema, check, fix the holdouts -
but the forming, drafting, and threshold-choosing live with whoever drives the
tool, not in the engine.

## See also

- The [inspectors reference]({{< relref "../../reference/inspectors/_index.md" >}})
  for the per-inspector surface, generated from the registry.
- [Checks]({{< relref "checks.md" >}}) - the prescriptive dual; an
  inspector measures the distribution a check would assert against.
- [Domain model]({{< relref "_index.md" >}}) for where profiling sits in
  the catalog-define-enforce loop.
- `go doc ./internal/inspect` for the code-level engine contract.
