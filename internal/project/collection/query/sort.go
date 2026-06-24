package query

import (
	"fmt"
	"strings"
)

// SortKey is one parsed --sort key. Build it with ParseSort.
type SortKey struct {
	field string
	desc  bool
}

// ParseSort parses a comma-joined sort spec ("year,-title"). A leading "-"
// marks a key descending. Keys are "id", "status", or a frontmatter dot
// path.
func ParseSort(s string) ([]SortKey, error) {
	var keys []SortKey
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := false
		if strings.HasPrefix(part, "-") {
			desc = true
			part = strings.TrimSpace(part[1:])
		}
		if part == "" {
			return nil, fmt.Errorf("invalid sort key in %q", s)
		}
		keys = append(keys, SortKey{field: part, desc: desc})
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("empty sort spec")
	}
	return keys, nil
}

// less is the comparator for the sort stage: walk the keys in order, and on
// the first decisive one return whether a precedes b. Ties (all keys equal)
// break by id ascending. missing is "last" or "lowest".
func less(a, b Record, keys []SortKey, missing string) bool {
	for _, k := range keys {
		va, oka := keyValue(a, k.field)
		vb, okb := keyValue(b, k.field)

		if !oka && !okb {
			continue
		}
		if !oka || !okb {
			if missing == "last" {
				// Present value always precedes a missing one,
				// regardless of sort direction.
				return oka
			}
			// "lowest": a missing field is below any present value,
			// then direction applies.
			c := 1
			if !oka {
				c = -1
			}
			if k.desc {
				c = -c
			}
			return c < 0
		}

		c, ok := compareVals(va, vb)
		if !ok || c == 0 {
			continue
		}
		if k.desc {
			c = -c
		}
		return c < 0
	}
	return a.ID < b.ID
}

// keyValue extracts the sortable value for a key. "id" and "status" are the
// record's own fields; anything else is a frontmatter dot path.
func keyValue(r Record, field string) (any, bool) {
	switch field {
	case "id":
		return r.ID, true
	case "status":
		return r.Status, true
	default:
		return lookup(r.Meta, field)
	}
}

// compareVals orders two values for sorting. Directly comparable pairs
// (numbers, strings) use compare; otherwise a stable type rank applies
// (numbers < strings < bools < other) so mixed-type columns stay
// deterministic. ok is false only for same-rank incomparable values (e.g.
// two arrays), treated as equal.
func compareVals(a, b any) (int, bool) {
	if c, ok := compare(a, b); ok {
		return c, true
	}
	ra, rb := typeRank(a), typeRank(b)
	if ra != rb {
		if ra < rb {
			return -1, true
		}
		return 1, true
	}
	return 0, false
}

func typeRank(v any) int {
	if _, ok := toFloat(v); ok {
		return 0
	}
	switch v.(type) {
	case string:
		return 1
	case bool:
		return 2
	default:
		return 3
	}
}
