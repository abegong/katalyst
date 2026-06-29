package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
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

type pathDepthArgs struct {
	Min *float64 `yaml:"min"`
	Max *float64 `yaml:"max"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckFilesystemPathDepth,
		Family:    "fileSystem",
		Targets:   []string{checks.TargetCollection, checks.TargetFilesystem},
		Slug:      "path-depth",
		Title:     "Path depth",
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
	}, checks.ParseInto(func(a pathDepthArgs) error {
		return argcheck.RequireOneOfFields("filesystem_path_depth", a.Min != nil || a.Max != nil, "min", "max")
	}), func(a any) checks.Check {
		x := a.(pathDepthArgs)
		return PathDepth{Min: intPtr(x.Min), Max: intPtr(x.Max)}
	}, nil)
}
