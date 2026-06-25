package inspect

import "fmt"

// collapseMode selects how the summarizer decides class membership.
type collapseMode int

const (
	// thresholdMode merges profiles whose similarity meets a fixed threshold.
	thresholdMode collapseMode = iota
	// budgetMode lowers the threshold until the class count fits maxClasses.
	budgetMode
)

// Params carries inspector parameters. Today it holds only the summarizer's
// collapse tolerance, the first inspector parameter. Inspectors that don't
// summarize ignore it.
type Params struct {
	mode       collapseMode
	threshold  float64
	maxClasses int
	Selection  Selection
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

// detailThresholds maps the named --detail levels to similarity thresholds.
// exact keeps only identical profiles together; coarse merges aggressively.
var detailThresholds = map[string]float64{
	"exact":   1.0,
	"grouped": 0.6,
	"coarse":  0.3,
}

// ParseParams resolves the three collapse-tolerance forms into Params. The
// forms are mutually exclusive: a caller passes at most one of a named detail
// level, a 0–1 similarity proportion, or a max-classes budget. With none set,
// the default is the `grouped` named level. Unset sentinels: detail "",
// similarity < 0, maxClasses <= 0.
func ParseParams(detail string, similarity float64, maxClasses int) (Params, error) {
	set := 0
	if detail != "" {
		set++
	}
	if similarity >= 0 {
		set++
	}
	if maxClasses > 0 {
		set++
	}
	if set > 1 {
		return Params{}, fmt.Errorf("--detail, --similarity, and --max-classes are mutually exclusive")
	}

	switch {
	case maxClasses > 0:
		return Params{mode: budgetMode, maxClasses: maxClasses}, nil
	case similarity >= 0:
		if similarity > 1 {
			return Params{}, fmt.Errorf("--similarity: must be between 0 and 1 (got %v)", similarity)
		}
		return Params{mode: thresholdMode, threshold: similarity}, nil
	default:
		level := detail
		if level == "" {
			level = "grouped"
		}
		thr, ok := detailThresholds[level]
		if !ok {
			return Params{}, fmt.Errorf("--detail: must be exact, grouped, or coarse (got %q)", level)
		}
		return Params{mode: thresholdMode, threshold: thr}, nil
	}
}
