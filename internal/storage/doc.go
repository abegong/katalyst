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
// # Scope
//
// Whether a matched store unit becomes an Item or a Collection is a property of
// the StorageType, not user configuration. A markdown file is an Item; a SQL
// table would be a Collection. Item and Collection are therefore roles, not file
// counts. See docs/content/deep-dives/domain-model/storage.md.
package storage
