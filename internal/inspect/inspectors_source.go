package inspect

import (
	"path"
	"strings"

	"github.com/abegong/katalyst/internal/storage"
)

// FileTree is the shallow, cheap raw-source inspector: a per-directory
// path-level profile (file types, naming, depth) summarized into classes. It
// opens no files. Filesystem-specific. Subsumes the former filesystem_naming.
type FileTree struct{}

func (FileTree) Name() string { return "file_tree" }

func (FileTree) AppliesTo(t storage.StorageType) bool { return t == storage.Filesystem }

func (FileTree) Inspect(v SourceView, p Params) Evidence {
	byDir := v.refsByDir()
	profiles := make([]Profile, 0, len(byDir))
	for _, dir := range sortedKeys(byDir) {
		profiles = append(profiles, Profile{Label: dir, Features: dirFeatures(byDir[dir])})
	}
	return Evidence{Inspector: "file_tree", Scope: v.root, N: v.N(), Data: summarize(profiles, p)}
}

// FileTreeContent is the deep raw-source inspector: it parses markdown and
// profiles each directory by content shape (frontmatter key union, parse /
// frontmatter presence), summarized into classes. Subsumes the former
// walk_parse. Filesystem-specific.
type FileTreeContent struct{}

func (FileTreeContent) Name() string { return "file_tree_content" }

func (FileTreeContent) AppliesTo(t storage.StorageType) bool { return t == storage.Filesystem }

func (FileTreeContent) Inspect(v SourceView, p Params) Evidence {
	byDir := map[string][]sourceDoc{}
	for _, sd := range v.markdown() {
		byDir[sd.dir] = append(byDir[sd.dir], sd)
	}
	profiles := make([]Profile, 0, len(byDir))
	for _, dir := range sortedKeys(byDir) {
		profiles = append(profiles, Profile{Label: dir, Features: contentFeatures(byDir[dir])})
	}
	return Evidence{Inspector: "file_tree_content", Scope: v.root, N: v.N(), Data: summarize(profiles, p)}
}

// DocumentShape clusters markdown files into candidate collections on a
// composite fingerprint: frontmatter keys, body section skeleton, and file
// type/naming, so a class agrees on metadata AND structure AND convention, not
// frontmatter alone. The renamed, widened frontmatter_shape. Filesystem-specific.
type DocumentShape struct{}

func (DocumentShape) Name() string { return "document_shape" }

func (DocumentShape) AppliesTo(t storage.StorageType) bool { return t == storage.Filesystem }

func (DocumentShape) Inspect(v SourceView, p Params) Evidence {
	docs := v.markdown()
	profiles := make([]Profile, 0, len(docs))
	for _, sd := range docs {
		profiles = append(profiles, Profile{Label: sd.rel, Features: shapeFeatures(sd)})
	}
	return Evidence{Inspector: "document_shape", Scope: v.root, N: len(docs), Data: summarize(profiles, p)}
}

// dirFeatures turns one directory's file list into summarizer feature tokens:
// the extensions present, the dominant filename casing, and a spaces marker.
func dirFeatures(refs []string) []string {
	meta := fileMetadata(refs)
	var feats []string
	for _, e := range sortedKeys(meta["extensions"].(map[string]any)) {
		feats = append(feats, "ext:"+e)
	}
	feats = append(feats, "casing:"+dominant(meta["casing"].(map[string]any)))
	if meta["with_spaces"].(int) > 0 {
		feats = append(feats, "spaces")
	}
	return feats
}

// contentFeatures fingerprints a directory by markdown content shape: the union
// of frontmatter keys plus parse/frontmatter presence markers.
func contentFeatures(docs []sourceDoc) []string {
	keys := map[string]bool{}
	parsed, withFM := 0, 0
	for _, sd := range docs {
		if sd.doc == nil {
			continue
		}
		parsed++
		if sd.doc.HasFrontmatter {
			withFM++
			for k := range sd.doc.Meta {
				keys[k] = true
			}
		}
	}
	var feats []string
	for _, k := range sortedKeys(keys) {
		feats = append(feats, "fmkey:"+k)
	}
	if parsed > 0 {
		feats = append(feats, "parsed")
	}
	if withFM > 0 {
		feats = append(feats, "frontmatter")
	}
	return feats
}

// shapeFeatures builds a file's composite fingerprint across three dimensions:
// file type/naming, frontmatter keys, and body section skeleton.
func shapeFeatures(sd sourceDoc) []string {
	feats := []string{
		"ext:" + strings.ToLower(path.Ext(sd.rel)),
		"casing:" + nameCasing(sd.rel),
	}
	if sd.doc == nil {
		return feats
	}
	for _, k := range sortedKeys(sd.doc.Meta) {
		feats = append(feats, "fmkey:"+k)
	}
	for _, h := range headings(sd.doc.Body) {
		if h.level >= 2 {
			feats = append(feats, "sec:"+h.text)
		}
	}
	return feats
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

// nameCasing classifies a file's stem as kebab, snake, or other.
func nameCasing(rel string) string {
	stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	switch {
	case kebabPattern.MatchString(stem):
		return "kebab"
	case snakePattern.MatchString(stem):
		return "snake"
	default:
		return "other"
	}
}
