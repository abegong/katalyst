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

	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection"
	"github.com/abegong/katalyst/internal/storage/collection/filesystem"
	sqlitestore "github.com/abegong/katalyst/internal/storage/collection/sqlite"
	"github.com/abegong/katalyst/internal/storage/filesystemcheck"
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

// ItemContent is the decoded content for one item.
type ItemContent struct {
	Raw []byte
	Doc *markdownbodytext.Document
}

func (p *Project) baseInstance(name string) (BaseInstance, bool) {
	for _, inst := range p.cfg.Bases {
		if inst.Name == name {
			return inst, true
		}
	}
	return BaseInstance{}, false
}

func (p *Project) def(c Collection) (collection.CollectionDefinition, error) {
	inst, ok := p.baseInstance(c.Base)
	if !ok {
		return nil, fmt.Errorf("collection %q: unknown base %q", c.Name, c.Base)
	}
	switch storage.BaseType(inst.Type) {
	case storage.Filesystem:
		return filesystem.New(inst.Root, inst.Collections), nil
	case storage.SQLite:
		return sqlitestore.New(inst.Root, inst.Collections), nil
	default:
		return nil, fmt.Errorf("collection %q: unsupported base type %q", c.Name, inst.Type)
	}
}

// Collections returns all collections in name order.
func (p *Project) Collections() []Collection { return p.cfg.Collections }

// FilesystemCheckScopes returns all filesystem-attached check scopes.
func (p *Project) FilesystemCheckScopes() []filesystemcheck.Scope {
	return p.cfg.FilesystemCheckScopes()
}

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
	def, err := p.def(c)
	if err != nil {
		return nil, err
	}
	return def.Items(c)
}

// Unmatched lists files inside a collection's directory that do NOT match
// its pattern. These are reported as errors by `check` (cf.
// docs/content/reference/configuration.md). Paths are returned relative to Dir.
func (p *Project) Unmatched(c Collection) ([]string, error) {
	def, err := p.def(c)
	if err != nil {
		return nil, err
	}
	refs, err := def.Unmatched(c)
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
	def, err := p.def(c)
	if err != nil {
		return Item{}, err
	}
	ref, err := def.Reference(c, id)
	if err != nil {
		return Item{}, err
	}
	path := string(ref)
	if c.StorageType == string(storage.SQLite) {
		def, err := p.def(c)
		if err != nil {
			return Item{}, err
		}
		exists, err := def.(*sqlitestore.Definition).Exists(c, id)
		if err != nil {
			return Item{}, err
		}
		if !exists {
			return Item{}, &UsageError{Msg: fmt.Sprintf("unknown item %q in collection %q", id, collection)}
		}
		return Item{Collection: c, ID: id, Path: path}, nil
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return Item{}, &UsageError{Msg: fmt.Sprintf("unknown item %q in collection %q", id, collection)}
	}
	return Item{Collection: c, ID: id, Path: path}, nil
}

// Reference resolves an item id to a backend-native reference string.
func (p *Project) Reference(c Collection, id string) (string, error) {
	def, err := p.def(c)
	if err != nil {
		return "", err
	}
	ref, err := def.Reference(c, id)
	return string(ref), err
}

// ReadItem reads and decodes an item through its storage backend.
func (p *Project) ReadItem(item Item) (ItemContent, error) {
	switch storage.BaseType(item.Collection.StorageType) {
	case storage.SQLite:
		def, err := p.def(item.Collection)
		if err != nil {
			return ItemContent{}, err
		}
		raw, doc, err := def.(*sqlitestore.Definition).Read(item)
		return ItemContent{Raw: raw, Doc: doc}, err
	default:
		src, err := os.ReadFile(item.Path)
		if err != nil {
			return ItemContent{}, err
		}
		doc, err := markdownbodytext.Parse(src)
		if err != nil {
			return ItemContent{}, err
		}
		return ItemContent{Raw: src, Doc: doc}, nil
	}
}

// ItemExists reports whether id already exists in c.
func (p *Project) ItemExists(c Collection, id string) (bool, error) {
	switch storage.BaseType(c.StorageType) {
	case storage.SQLite:
		def, err := p.def(c)
		if err != nil {
			return false, err
		}
		return def.(*sqlitestore.Definition).Exists(c, id)
	default:
		ref, err := p.Reference(c, id)
		if err != nil {
			return false, err
		}
		info, err := os.Stat(ref)
		return err == nil && !info.IsDir(), nil
	}
}

// AddItem creates a new item in c.
func (p *Project) AddItem(c Collection, id string, meta map[string]any, body []byte) error {
	switch storage.BaseType(c.StorageType) {
	case storage.SQLite:
		def, err := p.def(c)
		if err != nil {
			return err
		}
		return def.(*sqlitestore.Definition).Add(c, id, meta, body)
	default:
		return fmt.Errorf("filesystem item writes are handled by cmd")
	}
}

// UpdateItem updates an existing item in c.
func (p *Project) UpdateItem(c Collection, id string, meta map[string]any, body []byte) error {
	switch storage.BaseType(c.StorageType) {
	case storage.SQLite:
		def, err := p.def(c)
		if err != nil {
			return err
		}
		return def.(*sqlitestore.Definition).Update(c, id, meta, body)
	default:
		return fmt.Errorf("filesystem item writes are handled by cmd")
	}
}

// DeleteItem deletes an existing item.
func (p *Project) DeleteItem(item Item) error {
	switch storage.BaseType(item.Collection.StorageType) {
	case storage.SQLite:
		def, err := p.def(item.Collection)
		if err != nil {
			return err
		}
		return def.(*sqlitestore.Definition).Delete(item.Collection, item.ID)
	default:
		return os.Remove(item.Path)
	}
}
