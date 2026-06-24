// Package storage is the seam between a backend store and the katalyst domain
// model, issue #31's "narrow interface."
//
// # Three concepts
//
//   - StorageType: a known backend kind (filesystem today; sqlite, postgresql,
//     mongodb later). The registry here is the extension point.
//   - StorageInstance (assembled by the internal/project loader): one configured
//     store of a type plus how to reach it, embedding the collections it maps.
//   - CollectionDefinition: the two-way mapping from a store's contents to
//     collections and items. FilesystemCollectionDefinition is the first.
//
// # The two-way contract
//
// A CollectionDefinition maps in both directions: forward (discover the
// collections and items a store holds: Collections, Items, Unmatched) and
// reverse (reconstruct a backend locator from an item identity, Reference).
// The reverse direction is mandatory: `item add notes/dune` needs it to decide
// which file to create. Today an item is identified by one coordinate (its
// filename stem) and Reference is Join(dir, id+ext); richer layouts grow into
// multi-coordinate templates.
//
// # Granularity
//
// Whether a matched store unit becomes an Item or a Collection is a property of
// the StorageType (Granularity), not user configuration. A markdown file is an
// Item (FileIsItem); a SQL table would be a Collection (UnitIsCollection). Item
// and Collection are therefore roles, not file counts.
//
// # Lineage
//
// The design adapts Great Expectations' V3 DataConnector layer: its Datasource
// vs. DataConnector split is this package's StorageInstance vs.
// CollectionDefinition split. Corrections carried from GX's own TODOs: prefer a
// two-way template over inverting a regex, let the pattern own the file
// extension, and keep collection identity separate from within-collection
// coordinates. See docs/content/deep-dives/storage.md.
package storage
