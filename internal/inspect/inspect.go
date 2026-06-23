package inspect

// Evidence is the result of one inspector over a Corpus. It is descriptive
// only: counts and distributions, never a recommendation or verdict. N is the
// denominator (the number of files measured) so a consumer computes its own
// confidence rather than trusting a baked-in threshold. Description is a
// one-line statement of what the results mean, filled from the registry at
// render time. Data holds the inspector-specific payload; one map keeps a
// single renderer pair serving every inspector.
type Evidence struct {
	Inspector   string         `json:"inspector"`
	Description string         `json:"description,omitempty"`
	Scope       string         `json:"scope"`
	N           int            `json:"n"`
	Data        map[string]any `json:"evidence"`
}

// Inspector measures one aspect of a Corpus and returns Evidence. An inspector
// is a pure function of the Corpus — it reads parsed documents and never
// touches disk — so repeated runs are deterministic.
//
// Deprecated: the single-Corpus model is being replaced by the two-layer model
// (SourceInspector / CollectionInspector) from inspector-layers-spec.md. The
// Corpus-based inspectors are removed in the registry cutover.
type Inspector interface {
	Name() string
	Inspect(Corpus) Evidence
}

// CollectionInspector measures a configured collection, addressed by domain
// identity (Collection + Item.ID) through a CollectionView rather than by raw
// path. It is the collection-layer half of the two-layer inspector model; the
// raw-source half is SourceInspector. Params carries the collapse tolerance for
// summarizing inspectors and is ignored by those that don't summarize.
type CollectionInspector interface {
	Name() string
	Inspect(CollectionView, Params) Evidence
}
