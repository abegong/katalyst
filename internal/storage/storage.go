// Package storage is the seam between a backend store and the katalyst domain
// model. A CollectionDefinition is the two-way mapping a backend implements:
// forward (discover the collections and items a store holds) and reverse
// (reconstruct a backend locator from an item identity). The filesystem is the
// only backend today; the registry below is where a second one (SQLite, S3, a
// database) attaches without the check engine, the CRUD verbs, or selector
// parsing needing to know.
//
// See docs/content/deep-dives/storage.md for the conceptual model and the
// Great Expectations lineage it adapts.
package storage

import "github.com/abegong/katalyst/internal/config"

// StorageType is a known backend kind capable of holding collections and items.
type StorageType string

// Filesystem is the only backend implemented today.
const Filesystem StorageType = "filesystem"

// registered is the set of backend kinds with an implementation. It is the
// extension point: a new StorageType is added here when its
// CollectionDefinition lands.
var registered = map[StorageType]bool{
	Filesystem: true,
}

// Known reports whether a StorageType has an implementation. Config carries the
// type as a plain string and leaves this validation to the storage layer, so
// that config never has to import storage.
func Known(t StorageType) bool { return registered[t] }

// Granularity is the level at which a backend's matched units attach to the
// domain model. It is a property of the StorageType, not user configuration: a
// markdown filesystem makes each file an Item, while a tabular backend would
// make each table a Collection and each row an Item.
type Granularity int

const (
	// FileIsItem: one file is one Item; a directory of files is a Collection.
	FileIsItem Granularity = iota
	// UnitIsCollection: one store unit (a table/file) is a Collection; its
	// rows are Items. Reserved for future tabular backends.
	UnitIsCollection
)

// Reference is a backend-native locator — a file path today, a table name or
// object key later. Kept opaque so non-filesystem backends are not forced into
// path semantics.
type Reference string

// CollectionDefinition is the two-way mapping between one storage backend and
// the collection/item domain model. The forward direction discovers structure
// (Collections, Items, Unmatched); the reverse direction reconstructs a backend
// locator from an item identity (Reference). Both directions are mandatory.
type CollectionDefinition interface {
	// Granularity reports how this backend's units attach to the model.
	Granularity() Granularity

	// Collections returns the collections this definition maps. One definition
	// may yield more than one collection.
	Collections() []config.Collection

	// Items lists the items in a collection (forward discovery).
	Items(config.Collection) ([]Item, error)

	// Unmatched lists backend references inside a collection's scope that do
	// not map to any item. Surfacing them is deliberate: silent skips hide
	// configuration drift.
	Unmatched(config.Collection) ([]Reference, error)

	// Reference reconstructs the backend locator for an item identity (reverse
	// resolution). It is what `item add notes/dune` uses to decide which file
	// to create.
	Reference(c config.Collection, id string) (Reference, error)
}
