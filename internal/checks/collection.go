package checks

import (
	"fmt"
	"sort"
	"strings"
)

// ItemContext is one item's data, as seen by a collection-scoped check.
type ItemContext struct {
	FilePath string
	Meta     map[string]any
}

// FileSetContext carries every selected file in a set, for checks that reason
// across siblings (uniqueness, required index files, unmatched files).
type FileSetContext struct {
	Root      string
	Items     []ItemContext
	Unmatched []string
	Include   []string
	Exclude   []string
}

// CollectionContext is the historical name for FileSetContext.
type CollectionContext = FileSetContext

// CollectionCheck validates a concern across all items in a collection. It
// runs once per collection, after the per-item pass.
type CollectionCheck interface {
	RunCollection(ctx CollectionContext) []Violation
}

// FileSetCheck is the product-facing name for a set-level runtime check.
type FileSetCheck = CollectionCheck

// CollisionViolations emits one violation per group of two or more paths,
// naming all colliding files. Groups and paths are sorted for determinism. It
// is the shared helper behind the uniqueness checks (unique_filename in the
// filesystem family, unique_field in the structuredobject family).
func CollisionViolations(groups map[string][]string, noun string) []Violation {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []Violation
	for _, k := range keys {
		paths := groups[k]
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		out = append(out, Violation{
			File:    paths[0],
			Message: fmt.Sprintf("duplicate %s %q shared by %s", noun, k, strings.Join(paths, ", ")),
		})
	}
	return out
}

// RunCollectionAll runs every collection check and flattens the violations.
func RunCollectionAll(ctx CollectionContext, list []CollectionCheck) []Violation {
	out := make([]Violation, 0)
	for _, c := range list {
		out = append(out, c.RunCollection(ctx)...)
	}
	return out
}

// RunFileSetAll runs every file-set check and flattens the violations.
func RunFileSetAll(ctx FileSetContext, list []FileSetCheck) []Violation {
	return RunCollectionAll(ctx, list)
}
