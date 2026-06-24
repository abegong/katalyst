package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/abegong/katalyst/internal/project/config"
	"github.com/bmatcuk/doublestar/v4"
)

// FilesystemCollectionDefinition maps a directory tree onto collections of
// markdown files: one file is one item, its id is the filename stem. It is the
// CollectionDefinition for StorageType filesystem.
//
// The per-collection methods operate on the absolute Dir already resolved on
// each config.Collection, so root is unused today; it is retained because a
// filesystem instance is identified by its root and Phase 2's BuildInstance
// resolves collection directories against it.
type FilesystemCollectionDefinition struct {
	root        string
	collections []config.Collection
}

// NewFilesystem builds a filesystem definition for the given collections,
// rooted at root.
func NewFilesystem(root string, collections []config.Collection) *FilesystemCollectionDefinition {
	return &FilesystemCollectionDefinition{root: root, collections: collections}
}

// Granularity is FileIsItem for the markdown filesystem.
func (f *FilesystemCollectionDefinition) Granularity() Granularity { return FileIsItem }

// Collections returns the collections this definition maps.
func (f *FilesystemCollectionDefinition) Collections() []config.Collection { return f.collections }

// Items lists the items in a collection: files under its directory that match
// its pattern, sorted by id. A missing directory yields no items.
func (f *FilesystemCollectionDefinition) Items(c config.Collection) ([]Item, error) {
	if info, err := os.Stat(c.Dir); err != nil || !info.IsDir() {
		return nil, nil
	}
	matches, err := doublestar.Glob(os.DirFS(c.Dir), c.Pattern)
	if err != nil {
		return nil, fmt.Errorf("collection %q: %w", c.Name, err)
	}
	sort.Strings(matches)
	items := make([]Item, 0, len(matches))
	for _, rel := range matches {
		id := rel[:len(rel)-len(c.Ext())]
		items = append(items, Item{
			Collection: c,
			ID:         filepath.ToSlash(id),
			Path:       filepath.Join(c.Dir, rel),
		})
	}
	return items, nil
}

// Unmatched lists files inside a collection's directory that do NOT match its
// pattern, as references relative to Dir. A missing directory yields nothing.
func (f *FilesystemCollectionDefinition) Unmatched(c config.Collection) ([]Reference, error) {
	info, err := os.Stat(c.Dir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}
	var out []Reference
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
			out = append(out, Reference(rel))
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
func (f *FilesystemCollectionDefinition) Reference(c config.Collection, id string) (Reference, error) {
	return Reference(filepath.Join(c.Dir, filepath.FromSlash(id)+c.Ext())), nil
}
