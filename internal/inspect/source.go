package inspect

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// sourceFile is one file discovered by a SourceView walk: cheap path-level
// metadata only. Content inspectors open selected files explicitly.
type sourceFile struct {
	rel string // path relative to the root, slash-separated
	dir string // directory relative to the root ("." at the top level)
	ext string // lowercased extension, including the dot
}

// readCounter is shared across value copies of a SourceView so tests can assert
// that path-only inspectors open no files.
type readCounter struct {
	count int
}

// SourceView is the raw base layer's addressing surface: a filesystem tree
// walked once into per-file metadata, addressed by backend-native reference
// (the relative path). Path-level inspectors (file_tree) read only this
// metadata and open no files; content inspectors trigger a one-time markdown
// parse. Filesystem-only for now; generalizing the walk across base types is
// future work.
type SourceView struct {
	root  string
	files []sourceFile
	reads *readCounter
}

// NewSourceView walks root, collecting every non-hidden file's path metadata
// without opening it. Hidden entries (dot-prefixed, e.g. .git, .katalyst) are
// skipped as store noise.
func NewSourceView(root string) (SourceView, error) {
	v := SourceView{root: root, reads: &readCounter{}}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		v.files = append(v.files, sourceFile{
			rel: rel,
			dir: path.Dir(rel),
			ext: strings.ToLower(filepath.Ext(rel)),
		})
		return nil
	})
	if err != nil {
		return SourceView{}, err
	}
	return v, nil
}

// Root returns the scanned root path.
func (v SourceView) Root() string { return v.root }

// N is the number of files discovered.
func (v SourceView) N() int { return len(v.files) }

// ParseCount reports how many files the view has opened for content inspection,
// which proves file_tree opens nothing.
func (v SourceView) ParseCount() int { return v.reads.count }

// readFile opens one discovered file by relative path and records the read as
// content inspection work.
func (v SourceView) readFile(rel string) ([]byte, error) {
	v.reads.count++
	return os.ReadFile(filepath.Join(v.root, filepath.FromSlash(rel)))
}

// refsByDir groups every file's relative path by its directory.
func (v SourceView) refsByDir() map[string][]string {
	out := map[string][]string{}
	for _, f := range v.files {
		out[f.dir] = append(out[f.dir], f.rel)
	}
	return out
}
