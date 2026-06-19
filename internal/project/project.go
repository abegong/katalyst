// Package project sits on top of internal/config and provides the v0
// collection/item domain layer: selector parsing, item enumeration, and
// reverse id→path resolution. See product/cli-spec.md ("Selector
// grammar", "Config").
package project

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/katabase-ai/katalyst/internal/config"
)

// Project is a loaded configuration plus the operations the CLI needs to
// turn selectors into concrete items on disk.
type Project struct {
	cfg *config.Config
}

// New wraps a loaded config.
func New(cfg *config.Config) *Project { return &Project{cfg: cfg} }

// Config returns the underlying configuration.
func (p *Project) Config() *config.Config { return p.cfg }

// Item is one resolved item: a file in a collection's directory.
type Item struct {
	Collection config.Collection
	// ID is the collection-relative identifier (the filename stem for the
	// flat single-directory case).
	ID string
	// Path is the absolute path to the item file.
	Path string
}

// Collections returns all collections in name order.
func (p *Project) Collections() []config.Collection { return p.cfg.Collections }

// Collection looks up one collection by name.
func (p *Project) Collection(name string) (config.Collection, bool) {
	return p.cfg.Collection(name)
}

// ItemPath computes the on-disk path for an item id within a collection
// (reverse resolution: notes/dune → <dir>/dune.md).
func ItemPath(c config.Collection, id string) string {
	return filepath.Join(c.Dir, filepath.FromSlash(id)+c.Ext())
}

// Items lists the items in a collection: files under its directory that
// match its pattern, sorted by id. A missing directory yields no items.
func (p *Project) Items(c config.Collection) ([]Item, error) {
	if info, err := os.Stat(c.Dir); err != nil || !info.IsDir() {
		// No directory yet → no items.
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

// Unmatched lists files inside a collection's directory that do NOT match
// its pattern. These are reported as errors by `check` (cf.
// product/decisions.md D2). Paths are returned relative to Dir.
func (p *Project) Unmatched(c config.Collection) ([]string, error) {
	info, err := os.Stat(c.Dir)
	if err != nil || !info.IsDir() {
		// No directory yet → nothing unmatched.
		return nil, nil
	}
	var out []string
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
			out = append(out, rel)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("collection %q: %w", c.Name, walkErr)
	}
	sort.Strings(out)
	return out, nil
}

// ItemAt resolves an existing item from its collection and id. It returns
// a usage error when the collection is unknown or the item file does not
// exist.
func (p *Project) ItemAt(collection, id string) (Item, error) {
	c, ok := p.cfg.Collection(collection)
	if !ok {
		return Item{}, &UsageError{Msg: fmt.Sprintf("unknown collection %q", collection)}
	}
	path := ItemPath(c, id)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return Item{}, &UsageError{Msg: fmt.Sprintf("unknown item %q in collection %q", id, collection)}
	}
	return Item{Collection: c, ID: id, Path: path}, nil
}
