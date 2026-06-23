// Package filesystem holds the check types that validate filename and path
// conventions for items, plus the collection-scoped filesystem checks
// (unique_filename, index_file_required). Each check type lives in its own file
// with its Descriptor and self-registration.
package filesystem

import (
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
)

// Target names the slice of an item's path a name/path rule tests. The zero
// value ("") means TargetFilename.
const (
	TargetFilename     = "filename"      // basename without extension
	TargetFilenameExt  = "filename-ext"  // basename with extension
	TargetParentDir    = "parent-dir"    // immediate parent directory name
	TargetPathSegments = "path-segments" // every dir segment + the basename
)

// resolveTarget returns the path slice(s) a rule tests for the given target.
// path-segments is inclusive: every directory segment from the collection
// root down, plus the basename without extension. Other targets return a
// single value.
func resolveTarget(ctx checks.Context, target string) []string {
	fileName := filepath.Base(ctx.FilePath)
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	switch target {
	case TargetFilenameExt:
		return []string{fileName}
	case TargetParentDir:
		return []string{filepath.Base(filepath.Dir(ctx.FilePath))}
	case TargetPathSegments:
		return pathSegments(ctx)
	default: // TargetFilename
		return []string{base}
	}
}

// pathSegments returns each directory segment of the collection-relative
// path plus the basename without extension. Segments are computed relative
// to CollectionRoot when set, otherwise relative to the file's own parent.
func pathSegments(ctx checks.Context) []string {
	rel := ctx.FilePath
	if ctx.CollectionRoot != "" {
		if r, err := filepath.Rel(ctx.CollectionRoot, ctx.FilePath); err == nil {
			rel = r
		}
	}
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	out := make([]string, 0, len(parts))
	for i, p := range parts {
		if p == "" || p == "." || p == ".." {
			continue
		}
		if i == len(parts)-1 {
			p = strings.TrimSuffix(p, filepath.Ext(p))
		}
		out = append(out, p)
	}
	return out
}

// targetNoun is the human label for a target, used in messages.
func targetNoun(target string) string {
	switch target {
	case TargetFilenameExt:
		return "filename"
	case TargetParentDir:
		return "parent directory"
	case TargetPathSegments:
		return "path segment"
	default:
		return "filename"
	}
}
