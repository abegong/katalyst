// Package listing runs the `item list` filter/grep/sort/skip/limit pipeline
// over in-memory records. It is deliberately decoupled from the project,
// config, and frontmatter packages: callers assemble a []Record and the
// engine answers with the surviving, ordered subset.
//
// The pipeline mirrors MongoDB's find(filter).sort().skip().limit():
// filter and grep (both ANDed predicates) narrow the set, then sort, skip,
// and limit shape it, in that order.
//
// Predicate parsing lives in the sibling predicate package because collection
// variants reuse the same metadata grammar without depending on item listing.
package listing

import (
	"regexp"
	"sort"

	"github.com/abegong/katalyst/internal/storage/collection/predicate"
)

// Region selects which bytes of a record --grep searches.
type Region int

const (
	RegionAll         Region = iota // the whole raw file (default)
	RegionBody                      // the markdown body only
	RegionFrontmatter               // the raw frontmatter block only
)

// Record is one item as the engine sees it. Meta is the parsed
// frontmatter (nil/empty when the item has none or failed to parse); Raw,
// Body, and Frontmatter are the byte regions --grep can target.
type Record struct {
	ID          string
	Status      int
	Meta        map[string]any
	Raw         []byte
	Body        []byte
	Frontmatter []byte
}

// Options is the assembled listing operation. The zero value is a valid no-op
// (every record passes, default id-ascending order, no cap).
type Options struct {
	Filters      []predicate.Predicate
	Greps        []*regexp.Regexp
	GrepIn       Region
	Sorts        []SortKey
	Skip         int
	Limit        int
	TypeMismatch string // "skip" (default) | "error"
	SortMissing  string // "last" (default) | "lowest"
}

// Apply runs the pipeline and returns the surviving records. A filter type
// mismatch under TypeMismatch == "error" aborts with a *TypeMismatchError;
// every other outcome (including an empty result) is a nil error.
func Apply(recs []Record, opts Options) ([]Record, error) {
	out := make([]Record, 0, len(recs))
	for _, r := range recs {
		ok, err := matchAll(r, opts)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, r)
		}
	}

	if len(opts.Sorts) > 0 {
		missing := opts.SortMissing
		if missing == "" {
			missing = "last"
		}
		sort.SliceStable(out, func(i, j int) bool {
			return less(out[i], out[j], opts.Sorts, missing)
		})
	} else {
		sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	}

	if opts.Skip > 0 {
		if opts.Skip >= len(out) {
			return out[:0], nil
		}
		out = out[opts.Skip:]
	}
	if opts.Limit > 0 && opts.Limit < len(out) {
		out = out[:opts.Limit]
	}
	return out, nil
}

// matchAll reports whether a record satisfies every filter and grep
// (logical AND across both).
func matchAll(r Record, opts Options) (bool, error) {
	for _, p := range opts.Filters {
		ok, err := p.Matches(r.Meta, opts.TypeMismatch)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	for _, re := range opts.Greps {
		if !re.Match(region(r, opts.GrepIn)) {
			return false, nil
		}
	}
	return true, nil
}

func region(r Record, in Region) []byte {
	switch in {
	case RegionBody:
		return r.Body
	case RegionFrontmatter:
		return r.Frontmatter
	default:
		return r.Raw
	}
}
