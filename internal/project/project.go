// Package project is the v0 collection/item domain layer: it loads a project's
// .katalyst/ configuration (see loader.go) and provides selector parsing, item
// enumeration, and reverse id→path resolution on top of it. The path↔item-
// identity mapping itself lives behind the internal/storage seam; this package
// selects the right CollectionDefinition and orchestrates it. See
// docs/content/deep-dives/domain-model/_index.md (selectors, collections, items)
// and docs/content/deep-dives/domain-model/storage.md (the seam).
package project

import (
	"fmt"
	"os"

	"github.com/abegong/katalyst/internal/storage/collection"
	"github.com/abegong/katalyst/internal/storage/collection/filesystem"
)

// Project is a loaded configuration plus the operations the CLI needs to
// turn selectors into concrete items on disk.
type Project struct {
	cfg *Config
}

// New wraps a loaded config.
func New(cfg *Config) *Project { return &Project{cfg: cfg} }

// Config returns the underlying configuration.
func (p *Project) Config() *Config { return p.cfg }

// Item is one resolved item. It is the collection layer's Item, re-exported so
// callers (and the cmd layer) keep using project.Item unchanged.
type Item = collection.Item

// def builds the filesystem CollectionDefinition for this project's config.
// Today every configured storage instance is filesystem-backed, and the loaded
// collection directories have already been resolved against their instance root.
func (p *Project) def() *filesystem.Definition {
	return filesystem.New(p.cfg.Root, p.cfg.Collections)
}

// Collections returns all collections in name order.
func (p *Project) Collections() []Collection { return p.cfg.Collections }

// Collection looks up one collection by name.
func (p *Project) Collection(name string) (Collection, bool) {
	return p.cfg.Collection(name)
}

// ItemPath computes the on-disk path for an item id within a collection
// (reverse resolution: notes/dune → <dir>/dune.md). It delegates to the
// filesystem definition's Reference; the mapping itself lives in the storage
// seam.
func ItemPath(c Collection, id string) string {
	ref, _ := filesystem.New("", nil).Reference(c, id)
	return string(ref)
}

// Items lists the items in a collection: files under its directory that
// match its pattern, sorted by id. A missing directory yields no items.
func (p *Project) Items(c Collection) ([]Item, error) {
	return p.def().Items(c)
}

// Unmatched lists files inside a collection's directory that do NOT match
// its pattern. These are reported as errors by `check` (cf.
// docs/content/reference/configuration.md). Paths are returned relative to Dir.
func (p *Project) Unmatched(c Collection) ([]string, error) {
	refs, err := p.def().Unmatched(c)
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, nil
	}
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = string(r)
	}
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
	ref, err := p.def().Reference(c, id)
	if err != nil {
		return Item{}, err
	}
	path := string(ref)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return Item{}, &UsageError{Msg: fmt.Sprintf("unknown item %q in collection %q", id, collection)}
	}
	return Item{Collection: c, ID: id, Path: path}, nil
}
