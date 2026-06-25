package inspect

import "github.com/abegong/katalyst/internal/storage"

// FileTree is the shallow, cheap raw base inspector: a deterministic
// filesystem map from path metadata. It opens no files. Filesystem-specific.
// Subsumes the former filesystem_naming.
type FileTree struct{}

func (FileTree) Name() string { return "file_tree" }

func (FileTree) AppliesTo(t storage.BaseType) bool { return t == storage.Filesystem }

func (FileTree) Inspect(v SourceView, p Params) Evidence {
	return Evidence{Inspector: "file_tree", Scope: v.root, N: v.N(), Data: buildFileTreeSummary(v)}
}

// dominant returns the highest-count key in a histogram, ties broken by key for
// determinism.
func dominant(hist map[string]any) string {
	best, bestN := "", -1
	for _, k := range sortedKeys(hist) {
		if n := hist[k].(int); n > bestN {
			best, bestN = k, n
		}
	}
	return best
}
