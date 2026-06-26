// Package filesystem is the filesystem backend's CollectionDefinition: it maps a
// directory tree onto collections of markdown files (one file is one item, its
// id the filename stem) and persists item writes. The content decode/encode is
// the markdown body text codec's; this package owns the structural read and the
// on-disk IO.
package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection"
	"github.com/bmatcuk/doublestar/v4"
)

// Definition maps a directory tree onto collections of markdown files: one file
// is one item, its id is the filename stem. It is the CollectionDefinition for
// BaseType filesystem.
//
// The per-collection methods operate on the absolute Dir already resolved on
// each collection.Collection, so root is unused today; it is retained because a
// filesystem base is identified by its root and the project loader resolves
// collection directories against it.
type Definition struct {
	root        string
	collections []collection.Collection
}

// New builds a filesystem definition for the given collections, rooted at root.
func New(root string, collections []collection.Collection) *Definition {
	return &Definition{root: root, collections: collections}
}

// Scope reports item scope for the markdown filesystem.
func (f *Definition) Scope() storage.Scope { return storage.FileIsItem }

// Collections returns the collections this definition maps.
func (f *Definition) Collections() []collection.Collection { return f.collections }

// Items lists the items in a collection: files under its directory that match
// its pattern, sorted by id. A missing directory yields no items.
func (f *Definition) Items(c collection.Collection) ([]collection.Item, error) {
	if info, err := os.Stat(c.Dir); err != nil || !info.IsDir() {
		return nil, nil
	}
	matches, err := doublestar.Glob(os.DirFS(c.Dir), c.Pattern)
	if err != nil {
		return nil, fmt.Errorf("collection %q: %w", c.Name, err)
	}
	sort.Strings(matches)
	items := make([]collection.Item, 0, len(matches))
	for _, rel := range matches {
		id := rel[:len(rel)-len(c.Ext())]
		items = append(items, collection.Item{
			Collection: c,
			ID:         filepath.ToSlash(id),
			Path:       filepath.Join(c.Dir, rel),
		})
	}
	return items, nil
}

// Unmatched lists files inside a collection's directory that do NOT match its
// pattern, as references relative to Dir. A missing directory yields nothing.
func (f *Definition) Unmatched(c collection.Collection) ([]storage.Reference, error) {
	info, err := os.Stat(c.Dir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}
	var out []storage.Reference
	walkErr := filepath.WalkDir(c.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(c.Dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		ok, _ := doublestar.Match(c.Pattern, rel)
		if !ok {
			out = append(out, storage.Reference(rel))
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("collection %q: %w", c.Name, walkErr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

// Reference reconstructs the absolute file path for an item id within a
// collection (reverse resolution: notes/dune → <dir>/dune.md). Filesystem
// reconstruction cannot fail, but the interface allows backends that can.
func (f *Definition) Reference(c collection.Collection, id string) (storage.Reference, error) {
	return storage.Reference(filepath.Join(c.Dir, filepath.FromSlash(id)+c.Ext())), nil
}
