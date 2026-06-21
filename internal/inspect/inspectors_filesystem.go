package inspect

import (
	"path"
	"regexp"
	"strings"
)

var (
	kebabPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	snakePattern = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)
)

// FilesystemNaming reports filename and path conventions across the corpus: a
// casing histogram of basenames, how many paths contain spaces, the extension
// histogram, and the deepest nesting seen.
type FilesystemNaming struct{}

func (FilesystemNaming) Name() string { return "filesystem_naming" }

func (FilesystemNaming) Inspect(c Corpus) Evidence {
	casing := map[string]int{"kebab": 0, "snake": 0, "other": 0}
	exts := map[string]int{}
	withSpaces, maxDepth := 0, 0
	for _, f := range c.Files {
		rel := f.Rel
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
	extData := make(map[string]any, len(exts))
	for e, n := range exts {
		extData[e] = n
	}
	data := map[string]any{
		"casing":      map[string]any{"kebab": casing["kebab"], "snake": casing["snake"], "other": casing["other"]},
		"with_spaces": withSpaces,
		"extensions":  extData,
		"max_depth":   maxDepth,
	}
	return Evidence{Inspector: "filesystem_naming", Scope: c.Scope, N: len(c.Files), Data: data}
}
