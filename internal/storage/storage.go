package storage

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

// Reference is a backend-native locator: a file path today, a table name or
// object key later. Kept opaque so non-filesystem backends are not forced into
// path semantics. The CollectionDefinition contract that produces and consumes
// it lives in internal/storage/collection.
type Reference string
