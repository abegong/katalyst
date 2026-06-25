package inspect

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
)

// sourceFile is one file discovered by a SourceView walk: cheap path-level
// metadata only. Markdown content is parsed lazily (see markdown).
type sourceFile struct {
	rel string // path relative to the root, slash-separated
	dir string // directory relative to the root ("." at the top level)
	ext string // lowercased extension, including the dot
}

// sourceDoc is a parsed markdown file in a SourceView.
type sourceDoc struct {
	rel string
	dir string
	doc *markdownbodytext.Document // nil when the file failed to read or parse
}

// mdCache is the lazily-populated markdown parse, shared across value copies of
// a SourceView via a pointer so the parse happens at most once per view.
type mdCache struct {
	loaded bool
	count  int
	docs   []sourceDoc
}

// SourceView is the raw-source layer's addressing surface: a filesystem tree
// walked once into per-file metadata, addressed by backend-native reference
// (the relative path). Path-level inspectors (file_tree) read only this
// metadata and open no files; content inspectors trigger a one-time markdown
// parse. Filesystem-only for now; generalizing the walk into the storage layer
// is future work.
type SourceView struct {
	root  string
	files []sourceFile
	md    *mdCache
}

// NewSourceView walks root, collecting every non-hidden file's path metadata
// without opening it. Hidden entries (dot-prefixed, e.g. .git, .katalyst) are
// skipped as store noise.
func NewSourceView(root string) (SourceView, error) {
	v := SourceView{root: root, md: &mdCache{}}
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

// ParseCount reports how many files the view has opened to parse, 0 until a
// content inspector triggers the markdown parse, which proves file_tree opens
// nothing.
func (v SourceView) ParseCount() int { return v.md.count }

// readFile opens one discovered file by relative path and records the read as
// content inspection work.
func (v SourceView) readFile(rel string) ([]byte, error) {
	v.md.count++
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

// markdown lazily reads and parses every .md file once, caching on the shared
// mdCache so repeated content inspectors don't re-read disk.
func (v SourceView) markdown() []sourceDoc {
	if v.md.loaded {
		return v.md.docs
	}
	for _, f := range v.files {
		if f.ext != ".md" {
			continue
		}
		v.md.count++
		sd := sourceDoc{rel: f.rel, dir: f.dir}
		if src, err := os.ReadFile(filepath.Join(v.root, filepath.FromSlash(f.rel))); err == nil {
			if doc, perr := markdownbodytext.Parse(src); perr == nil {
				sd.doc = doc
			}
		}
		v.md.docs = append(v.md.docs, sd)
	}
	v.md.loaded = true
	return v.md.docs
}
