package storage

import "github.com/abegong/katalyst/internal/config"

// Item is one resolved item: a member of a collection, located in its backing
// store. It lives here (rather than in internal/project) so a
// CollectionDefinition can return items without internal/project and
// internal/storage importing each other. internal/project re-exports it as a
// type alias.
type Item struct {
	Collection config.Collection
	// ID is the collection-relative identifier, the filename stem for the
	// flat filesystem case, a richer set of coordinates for layouts that grow.
	ID string
	// Path is the absolute path to the item file (a filesystem Reference,
	// resolved).
	Path string
}
