package filesystem

import (
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
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
	registerParsed(checks.Descriptor{
		CheckType:      checks.CheckFilesystemUniqueFilename,
		Family:         "fileSystem",
		ConfigurableIn: []string{checks.ConfigCollection, checks.ConfigFilesystem},
		Slug:           "unique-filename",
		Title:          "Unique filename",
		Summary:        "Require that no two items in the collection share a basename.",
		Scope:          "collection",
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_unique_filename`,
	}, checks.NoArgs, nil, func(any) checks.CollectionCheck {
		return UniqueFilename{}
	})
}
