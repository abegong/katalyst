// Package inspect profiles content and returns evidence, the descriptive dual
// of internal/checks: a check asserts a predicate; an inspector reports the
// distribution that predicate would be tested against. Inspectors come in two
// layers (raw-source and collection) and are built from a few reusable
// measurement primitives. They report counts and distributions only, never
// recommendations.
//
// The full architecture and design rationale (the two layers, the primitives,
// evidence-not-recommendations, and the determinism dividing line) live in the
// "How inspectors work" deep-dive at docs/content/deep-dives/inspectors.md.
package inspect
