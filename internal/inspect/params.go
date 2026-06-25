package inspect

// Params carries inspector parameters. Inspectors that do not need a parameter
// ignore it.
type Params struct {
	Selection Selection
}

// Selection describes the path-derived file subset an inspector should use.
// Empty mode means "all files".
type Selection struct {
	Label   string
	Mode    string
	Pattern string
}

// WithSelection returns a copy of p carrying selection.
func (p Params) WithSelection(selection Selection) Params {
	p.Selection = selection
	return p
}
