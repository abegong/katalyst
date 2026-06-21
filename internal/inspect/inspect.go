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
type Inspector interface {
	Name() string
	Inspect(Corpus) Evidence
}
