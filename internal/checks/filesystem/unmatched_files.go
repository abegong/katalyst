package filesystem

import (
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
)

// UnmatchedFiles reports files in a filesystem scope that match neither
// include nor exclude patterns.
type UnmatchedFiles struct{}

func (UnmatchedFiles) RunCollection(ctx checks.CollectionContext) []checks.Violation {
	out := make([]checks.Violation, 0, len(ctx.Unmatched))
	for _, rel := range ctx.Unmatched {
		out = append(out, checks.Violation{
			File:    rel,
			Message: fmt.Sprintf("unmatched file (matches no include pattern %s and no exclude pattern %s)", patternList(ctx.Include), patternList(ctx.Exclude)),
		})
	}
	return out
}

func patternList(patterns []string) string {
	if len(patterns) == 0 {
		return "[]"
	}
	return "[" + strings.Join(patterns, ", ") + "]"
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckFilesystemUnmatchedFiles,
		Family:    "fileSystem",
		Targets:   []string{checks.TargetFilesystem},
		Slug:      "unmatched-files",
		Title:     "Unmatched files",
		Summary:   "Report regular files under a filesystem scope that match neither include nor exclude patterns.",
		Scope:     "collection",
		ConfigExample: `filesystemChecks:
  - path: docs
    include: ["**/*.md"]
    exclude: ["**/_generated/**"]
    checks:
      - kind: filesystem_unmatched_files`,
	}, checks.NoArgs, nil, func(any) checks.CollectionCheck {
		return UnmatchedFiles{}
	})
}
