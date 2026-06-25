// Package collection is the backend-neutral contract for reading a collection's
// items from a storage backend: the CollectionDefinition interface and the thin
// Item it yields. Per-backend implementations live in subpackages (filesystem
// today); listing and predicate subpackages hold the item-list pipeline and
// metadata predicate grammar. Readers use the markdown body text codec to decode
// item content.
package collection

import (
	"github.com/abegong/katalyst/internal/storage"
)

// Item is one resolved item: a member of a collection, located in its backing
// store. It carries no content — locating is the backend reader's job and
// parsing is the markdown body text codec's; Item only addresses. internal/project
// re-exports it as a type alias.
type Item struct {
	Collection Collection
	// ID is the collection-relative identifier, the filename stem for the
	// flat filesystem case, a richer set of coordinates for layouts that grow.
	ID string
	// Path is the absolute path to the item file (a filesystem Reference,
	// resolved).
	Path string
}

// CollectionDefinition is the two-way mapping between one storage backend and
// the collection/item domain model. The forward direction discovers structure
// (Collections, Items, Unmatched); the reverse direction reconstructs a backend
// locator from an item identity (Reference). Both directions are mandatory.
type CollectionDefinition interface {
	// Granularity reports the scope where this backend's units attach to the model.
	Granularity() storage.Granularity

	// Collections returns the collections this definition maps. One definition
	// may yield more than one collection.
	Collections() []Collection

	// Items lists the items in a collection (forward discovery).
	Items(Collection) ([]Item, error)

	// Unmatched lists backend references inside a collection's scope that do
	// not map to any item. Surfacing them is deliberate: silent skips hide
	// configuration drift.
	Unmatched(Collection) ([]storage.Reference, error)

	// Reference reconstructs the backend locator for an item identity (reverse
	// resolution). It is what `item add notes/dune` uses to decide which file
	// to create.
	Reference(c Collection, id string) (storage.Reference, error)
}
