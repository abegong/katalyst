package storage

// BaseType is a known backend kind capable of holding collections and items.
type BaseType string

// Filesystem is the only backend implemented today.
const Filesystem BaseType = "filesystem"

// registered is the set of backend kinds with an implementation. It is the
// extension point: a new BaseType is added here when its
// CollectionDefinition lands.
var registered = map[BaseType]bool{
	Filesystem: true,
}

// Known reports whether a BaseType has an implementation. The project loader
// carries the type as a plain string and leaves this validation to the storage
// layer, so the storage registry remains the source of truth for backend kinds.
func Known(t BaseType) bool { return registered[t] }

// Scope records the scope at which a backend's matched units attach to
// the domain model. It is a property of the BaseType, not user
// configuration: a markdown filesystem makes each file an Item, while a tabular
// backend would make each table a Collection and each row an Item.
type Scope int

const (
	// FileIsItem: one file is one Item; a directory of files is a Collection.
	FileIsItem Scope = iota
	// UnitIsCollection: one store unit (a table/file) is a Collection; its
	// rows are Items. Reserved for future tabular backends.
	UnitIsCollection
)

// Reference is a backend-native locator: a file path today, a table name or
// object key later. Kept opaque so non-filesystem backends are not forced into
// path semantics. The CollectionDefinition contract that produces and consumes
// it lives in internal/storage/collection.
type Reference string
