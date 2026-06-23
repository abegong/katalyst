package inspect

import (
	"fmt"
	"sort"
)

// Profile is one unit's fingerprint for the summarizer: a label (the directory
// or file it describes) and an unordered set of feature tokens. Two profiles
// are compared by Jaccard similarity over their feature sets.
type Profile struct {
	Label    string
	Features []string
}

// summarize collapses profiles into named classes so output is proportional to
// the number of distinct profiles, not the number of units. Profiles whose
// similarity meets the Params tolerance share a class; singletons are reported
// separately as outliers. Shared by the file_tree* and document_shape
// inspectors.
func summarize(profiles []Profile, p Params) map[string]any {
	sorted := append([]Profile(nil), profiles...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Label < sorted[j].Label })

	var classes []class
	if p.mode == budgetMode {
		classes = clusterToBudget(sorted, p.maxClasses)
	} else {
		classes = cluster(sorted, p.threshold)
	}

	classList := []any{}
	outliers := []any{}
	n := 0
	for _, c := range classes {
		if len(c.members) == 1 {
			outliers = append(outliers, map[string]any{
				"label":    c.members[0],
				"features": c.rep,
			})
			continue
		}
		n++
		classList = append(classList, map[string]any{
			"class":    fmt.Sprintf("P%d", n),
			"size":     len(c.members),
			"features": c.rep,
			"members":  c.members,
		})
	}
	return map[string]any{"classes": classList, "outliers": outliers}
}

// class is one cluster: its representative feature set (the first member's) and
// the labels of its members.
type class struct {
	rep     []string
	members []string
}

// cluster greedily groups profiles: each profile joins the first existing class
// whose representative is at least `threshold` similar, else starts a new class.
func cluster(profiles []Profile, threshold float64) []class {
	var classes []class
	for _, pr := range profiles {
		joined := false
		for i := range classes {
			if jaccard(classes[i].rep, pr.Features) >= threshold {
				classes[i].members = append(classes[i].members, pr.Label)
				joined = true
				break
			}
		}
		if !joined {
			classes = append(classes, class{rep: pr.Features, members: []string{pr.Label}})
		}
	}
	return classes
}

// clusterToBudget lowers the similarity threshold (1.00 → 0.00 in 0.05 steps)
// until the cluster count fits maxClasses, returning the tightest grouping that
// does. Threshold 0 merges everything into one class, so this always converges.
func clusterToBudget(profiles []Profile, maxClasses int) []class {
	for step := 100; step >= 0; step -= 5 {
		c := cluster(profiles, float64(step)/100)
		if len(c) <= maxClasses {
			return c
		}
	}
	return cluster(profiles, 0)
}

// jaccard is the Jaccard similarity of two feature sets: |A∩B| / |A∪B|. Two
// empty sets are identical (1).
func jaccard(a, b []string) float64 {
	sa, sb := toSet(a), toSet(b)
	if len(sa) == 0 && len(sb) == 0 {
		return 1
	}
	inter := 0
	for k := range sa {
		if sb[k] {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

func toSet(ss []string) map[string]bool {
	out := make(map[string]bool, len(ss))
	for _, s := range ss {
		out[s] = true
	}
	return out
}
