// Package inspect profiles a directory of markdown files: it measures their
// shape — frontmatter fields, body structure, filename conventions — and
// returns evidence, the descriptive dual of internal/checks. A check asserts a
// predicate; an inspector reports the distribution that predicate would be
// tested against.
//
// # Evidence, not recommendations
//
// An inspector reports that a field appears in 94% of files; it does not say
// "make it required." The threshold that turns 94% into a required field, or a
// small recurring value set into an enum, is a judgment call kept out of the
// measurement layer. This is the load-bearing decision. If inspectors emitted
// recommendations the threshold policy would be baked in and un-tunable, and
// the evidence itself would become something to second-guess rather than
// trust. Reporting only counts, with the file count n as denominator, keeps the
// evidence trustable: the reader sees why a conclusion holds and decides.
//
// # The determinism dividing line
//
// Deterministic measurement is an inspector's job; threshold-picking and
// structure-proposing are not. Counting field presence, histogramming types,
// grouping files by frontmatter key-set are all deterministic, all inspectors.
// Deciding that 94% is "required", that two near-but-distinct key-sets are one
// collection, or what to name a schema are all judgment, none of it here.
// FrontmatterShape sits on the seam: it groups files with identical
// fingerprints (deterministic) but leaves the fuzzy "these two groups are the
// same collection" call to the reader.
//
// # Division of labor
//
// Katalyst provides the instruments; a human or an agent is the profiler. The
// intended workflow is a loop — inspect, draft a schema, check, fix the
// holdouts — but the forming, drafting, and threshold-choosing live with
// whoever drives the tool, not in this package.
//
// # Parse once
//
// Load reads the directory into a Corpus a single time; every inspector is a
// pure function of that shared, parsed set and never touches disk. That keeps
// inspectors deterministic and testable and avoids re-reading each file once
// per inspector.
//
// # Output
//
// Evidence renders as Markdown by default and JSON under --json; both are
// projections of the same values. Markdown suits agents and humans alike, so it
// is the default; JSON is for callers that parse results mechanically.
//
// # Alternatives considered
//
// A monolithic command that emits a finished schema: rejected. It bundles
// measurement and judgment into one opaque step and over-fits, encoding the
// corpus's current state — mistakes and all — as authoritative.
//
// A "candidate collections" inspector: rejected. Drawing and naming collection
// boundaries is judgment; only the deterministic fingerprinting ships, as
// FrontmatterShape.
//
// Auto-applying an inferred schema: rejected. The value is a fast, conservative
// first draft a human edits; inspect writes nothing.
//
// Counterfactual check (testing a throwaway schema with "check --try") is the
// natural companion but is deferred; inspectors stand on their own.
package inspect
