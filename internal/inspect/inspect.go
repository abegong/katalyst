package inspect

import "github.com/abegong/katalyst/internal/storage"

// Evidence is the result of one inspector. It is descriptive only: counts and
// distributions, never a recommendation or verdict. N is the denominator (the
// number of units measured) so a consumer computes its own confidence rather
// than trusting a baked-in threshold. Description is a one-line statement of
// what the results mean, filled from the registry at render time. Data holds the
// inspector-specific payload; one map keeps a single renderer pair serving every
// inspector.
type Evidence struct {
	Inspector   string         `json:"inspector"`
	Description string         `json:"description,omitempty"`
	Scope       string         `json:"scope"`
	N           int            `json:"n"`
	Data        map[string]any `json:"evidence"`
}

// CollectionInspector measures a configured collection, addressed by domain
// identity (Collection + Item.ID) through a CollectionView rather than by raw
// path. It is the collection-layer half of the two-layer inspector model; the
// raw base half is SourceInspector. Params carries the collapse tolerance for
// summarizing inspectors and is ignored by those that don't summarize.
type CollectionInspector interface {
	Name() string
	Inspect(CollectionView, Params) Evidence
}

// SourceInspector measures a raw base before any collection
// configuration, addressed by backend-native reference (a path today) through a
// SourceView. AppliesTo gates backend-specific inspectors: one returns false for
// a BaseType it cannot describe, so it is simply absent there. It is the
// raw base half of the two-layer model; the collection half is
// CollectionInspector.
type SourceInspector interface {
	Name() string
	AppliesTo(storage.BaseType) bool
	Inspect(SourceView, Params) Evidence
}
