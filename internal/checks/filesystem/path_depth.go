package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// PathDepth bounds directory nesting relative to the collection root. A file
// directly in the collection root has depth 0.
type PathDepth struct {
	Min *int
	Max *int
}

func (c PathDepth) Run(ctx checks.Context) []checks.Violation {
	rel := ctx.FilePath
	if ctx.CollectionRoot != "" {
		if r, err := filepath.Rel(ctx.CollectionRoot, ctx.FilePath); err == nil {
			rel = r
		}
	}
	rel = filepath.ToSlash(rel)
	depth := 0
	for _, p := range strings.Split(rel, "/") {
		if p == "" || p == "." {
			continue
		}
		depth++
	}
	depth-- // last segment is the file itself
	if depth < 0 {
		depth = 0
	}
	var out []checks.Violation
	if c.Min != nil && depth < *c.Min {
		out = append(out, checks.Violation{
			Path:    "/",
			Message: fmt.Sprintf("path depth %d is below the minimum %d", depth, *c.Min),
		})
	}
	if c.Max != nil && depth > *c.Max {
		out = append(out, checks.Violation{
			Path:    "/",
			Message: fmt.Sprintf("path depth %d exceeds the maximum %d", depth, *c.Max),
		})
	}
	return out
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckFilesystemPathDepth,
		Family:    "fileSystem",
		Slug:      "path-depth",
		Title:     "Path Depth",
		Summary:   "Bound directory nesting relative to the collection root.",
		Fields: []checks.Field{
			{Name: "min", Required: false, Desc: "Minimum depth (at least one of min/max)."},
			{Name: "max", Required: false, Desc: "Maximum depth; `0` means a flat collection (at least one of min/max)."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_path_depth
        max: 0`,
	}, func(ch config.CheckInstance) checks.Check {
		return PathDepth{Min: ch.MinInt, Max: ch.MaxInt}
	}, nil)
}
