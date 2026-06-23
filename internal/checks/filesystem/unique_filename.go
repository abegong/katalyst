package filesystem

import (
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// UniqueFilename requires that no two items share a basename (without
// extension). It is collection-scoped.
type UniqueFilename struct{}

func (UniqueFilename) RunCollection(ctx checks.CollectionContext) []checks.Violation {
	groups := map[string][]string{}
	for _, it := range ctx.Items {
		name := it.FilePath
		fileName := filepath.Base(name)
		base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		groups[base] = append(groups[base], it.FilePath)
	}
	return checks.CollisionViolations(groups, "filename")
}

func init() {
	register(checks.Descriptor{
		CheckType: config.CheckFilesystemUniqueFilename,
		Family:    "fileSystem",
		Slug:      "unique-filename",
		Title:     "Unique filename",
		Summary:   "Require that no two items in the collection share a basename.",
		Scope:     "collection",
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_unique_filename`,
	}, nil, func(ch config.CheckInstance) checks.CollectionCheck {
		return UniqueFilename{}
	})
}
