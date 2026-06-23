package inspect

import (
	"path"
	"strings"
)

// fileMetadata reports path-level conventions over a set of references
// (relative paths): a casing histogram of filename stems, how many contain
// spaces, an extension histogram, and the deepest nesting seen. It opens no
// files. This is the file_metadata primitive behind the file_tree inspector.
func fileMetadata(refs []string) map[string]any {
	casing := map[string]int{"kebab": 0, "snake": 0, "other": 0}
	exts := map[string]int{}
	withSpaces, maxDepth := 0, 0
	for _, rel := range refs {
		if strings.Contains(rel, " ") {
			withSpaces++
		}
		base := path.Base(rel)
		ext := strings.ToLower(path.Ext(base))
		exts[ext]++
		stem := strings.TrimSuffix(base, path.Ext(base))
		switch {
		case kebabPattern.MatchString(stem):
			casing["kebab"]++
		case snakePattern.MatchString(stem):
			casing["snake"]++
		default:
			casing["other"]++
		}
		if depth := strings.Count(rel, "/"); depth > maxDepth {
			maxDepth = depth
		}
	}
	return map[string]any{
		"casing":      toAnyMap(casing),
		"with_spaces": withSpaces,
		"extensions":  toAnyMap(exts),
		"max_depth":   maxDepth,
	}
}
