package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ItemContext is one item's data, as seen by a collection-scoped check.
type ItemContext struct {
	FilePath string
	Meta     map[string]any
}

// CollectionContext carries every item in a collection, for checks that
// reason across siblings (uniqueness, required index files).
type CollectionContext struct {
	Root  string
	Items []ItemContext
}

// CollectionCheck validates a concern across all items in a collection. It
// runs once per collection, after the per-item pass.
type CollectionCheck interface {
	RunCollection(ctx CollectionContext) []Violation
}

// UniqueFilename requires that no two items share a basename (without
// extension).
type UniqueFilename struct{}

func (UniqueFilename) RunCollection(ctx CollectionContext) []Violation {
	groups := map[string][]string{}
	for _, it := range ctx.Items {
		name := it.FilePath
		fileName := filepath.Base(name)
		base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		groups[base] = append(groups[base], it.FilePath)
	}
	return collisionViolations(groups, "filename")
}

// UniqueField requires that no two items share a value for Field.
type UniqueField struct {
	Field string
}

func (c UniqueField) RunCollection(ctx CollectionContext) []Violation {
	groups := map[string][]string{}
	for _, it := range ctx.Items {
		raw, ok := it.Meta[c.Field]
		if !ok {
			continue
		}
		s, ok := raw.(string)
		if !ok {
			continue
		}
		groups[s] = append(groups[s], it.FilePath)
	}
	return collisionViolations(groups, fmt.Sprintf("%s value", c.Field))
}

// collisionViolations emits one violation per group of two or more paths,
// naming all colliding files. Groups and paths are sorted for determinism.
func collisionViolations(groups map[string][]string, noun string) []Violation {
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

// IndexFileRequired requires that every directory containing items also
// contains a file named Name (default "_index.md").
type IndexFileRequired struct {
	Name string
}

func (c IndexFileRequired) RunCollection(ctx CollectionContext) []Violation {
	name := c.Name
	if name == "" {
		name = "_index.md"
	}
	seen := map[string]bool{}
	var dirs []string
	for _, it := range ctx.Items {
		d := filepath.Dir(it.FilePath)
		if !seen[d] {
			seen[d] = true
			dirs = append(dirs, d)
		}
	}
	sort.Strings(dirs)
	var out []Violation
	for _, d := range dirs {
		idx := filepath.Join(d, name)
		if _, err := os.Stat(idx); err != nil {
			out = append(out, Violation{
				File:    d,
				Message: fmt.Sprintf("directory is missing required index file %q", name),
			})
		}
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
