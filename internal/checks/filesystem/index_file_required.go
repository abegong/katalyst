package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// IndexFileRequired requires that every directory containing items also
// contains a file named Name (default "_index.md"). It is collection-scoped.
type IndexFileRequired struct {
	Name string
}

func (c IndexFileRequired) RunCollection(ctx checks.CollectionContext) []checks.Violation {
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
	var out []checks.Violation
	for _, d := range dirs {
		idx := filepath.Join(d, name)
		if _, err := os.Stat(idx); err != nil {
			out = append(out, checks.Violation{
				File:    d,
				Message: fmt.Sprintf("directory is missing required index file %q", name),
			})
		}
	}
	return out
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckFilesystemIndexFileRequired,
		Family:    "fileSystem",
		Slug:      "index-file-required",
		Title:     "Index file required",
		Summary:   "Require that every directory containing items has an index file.",
		Scope:     "collection",
		Fields: []checks.Field{
			{Name: "name", Required: false, Default: "_index.md", Desc: "Index filename that must be present in each item directory."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_index_file_required`,
	}, nil, func(ch config.CheckInstance) checks.CollectionCheck {
		return IndexFileRequired{Name: ch.Name}
	})
}
