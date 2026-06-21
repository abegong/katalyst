package inspect

import (
	"sort"
	"strings"
)

// WalkParse reports corpus-level parse stats: how many files were found, how
// many parsed, how many carry frontmatter, and which failed. A parse failure
// is evidence (a file that claims no usable metadata), not an error to hide.
type WalkParse struct{}

func (WalkParse) Name() string { return "walk_parse" }

func (WalkParse) Inspect(c Corpus) Evidence {
	parsed, withFrontmatter := 0, 0
	var failures []string
	for _, f := range c.Files {
		if f.ParseErr != nil {
			failures = append(failures, f.Rel)
			continue
		}
		parsed++
		if f.Doc != nil && f.Doc.HasFrontmatter {
			withFrontmatter++
		}
	}
	data := map[string]any{
		"files":            len(c.Files),
		"parsed":           parsed,
		"failed":           len(failures),
		"with_frontmatter": withFrontmatter,
	}
	if len(failures) > 0 {
		data["failures"] = failures
	}
	return Evidence{Inspector: "walk_parse", Scope: c.Scope, N: len(c.Files), Data: data}
}

// FrontmatterShape groups files by the sorted set of their frontmatter keys —
// the fingerprint an agent clusters into candidate collections. Identical
// fingerprints are grouped here (a deterministic operation); deciding that two
// near-but-distinct groups are one collection is the agent's judgment, not this
// inspector's. Observed per-key types ship alongside as adjacent evidence.
type FrontmatterShape struct{}

func (FrontmatterShape) Name() string { return "frontmatter_shape" }

func (FrontmatterShape) Inspect(c Corpus) Evidence {
	groupCounts := map[string]int{}
	groupKeys := map[string][]string{}
	fieldTypes := map[string]map[string]bool{}

	for _, f := range c.Files {
		keys := sortedKeys(meta(f))
		fp := strings.Join(keys, ",")
		groupCounts[fp]++
		groupKeys[fp] = keys
		for k, v := range meta(f) {
			if fieldTypes[k] == nil {
				fieldTypes[k] = map[string]bool{}
			}
			fieldTypes[k][jsonType(v)] = true
		}
	}

	fingerprints := sortedKeys(groupCounts)
	// Most common group first; ties broken by fingerprint for determinism.
	sort.SliceStable(fingerprints, func(i, j int) bool {
		return groupCounts[fingerprints[i]] > groupCounts[fingerprints[j]]
	})
	groups := make([]any, 0, len(fingerprints))
	for _, fp := range fingerprints {
		keys := groupKeys[fp]
		if keys == nil {
			keys = []string{}
		}
		groups = append(groups, map[string]any{
			"fingerprint": fp,
			"keys":        keys,
			"count":       groupCounts[fp],
		})
	}

	fields := make(map[string]any, len(fieldTypes))
	for k, types := range fieldTypes {
		fields[k] = sortedKeys(types)
	}

	return Evidence{
		Inspector: "frontmatter_shape",
		Scope:     c.Scope,
		N:         len(c.Files),
		Data:      map[string]any{"groups": groups, "fields": fields},
	}
}
