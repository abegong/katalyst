// Package collection is the home for collection-scoped logic within a project.
//
// A collection is a named group of items sharing structure (see the glossary).
// It nests under project, mirroring the domain containment Project ⊃ Collection.
// Its query subpackage (internal/project/collection/query) holds the filter and
// sort grammar that operations scoped to a single collection are expressed in.
//
// This package currently carries no code of its own; collection types still live
// in config and storage. It exists as the seam those will migrate toward.
package collection
