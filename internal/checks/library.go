package checks

import (
	"sort"
)

// This file holds the CheckLibrary abstraction: the provider behind every check
// type. A library bundles one or more check types from one source. Native
// libraries (filesystem, plaintext, ...) wrap hand-written check types and need
// nothing more than CheckLibrary. Schema-backed libraries (json-schema today,
// vale next) also implement SchemaLibrary: they compile a named Schema from
// source bytes and run items against it.
//
// Library and family are orthogonal. A family is a check type's source-data
// kind (Descriptor.Family); a library is who supplies and runs the engine
// (Descriptor.Library). A single family spans libraries: structuredObject holds
// both object (json-schema) and object_required_field (the native
// structuredobject library).

// CheckLibrary is a provider of check types.
type CheckLibrary interface {
	// Name is the stable id used in diagnostics and docs ("filesystem",
	// "json-schema", "vale"). It never changes once published.
	Name() string

	// Available reports whether the library can run. Native and in-process
	// libraries return nil; an out-of-process library probes for its binary and
	// an acceptable version. A non-nil error fails the run.
	Available() error
}

// SchemaLibrary is a CheckLibrary that compiles named schemas from source
// bytes. The engine caches the compiled result per (library, path).
type SchemaLibrary interface {
	CheckLibrary
	// CompileSchema compiles one named schema. name identifies it for error
	// reporting and $ref resolution; src is the raw file content.
	CompileSchema(name string, src []byte) (Schema, error)
}

// Schema is one compiled artifact (a JSON Schema, a resolved Vale config) ready
// to evaluate items. It pulls the slice of the item it needs (Meta, body, path)
// out of the Context itself.
type Schema interface {
	Check(ctx Context) []Violation
}

var (
	libraries []CheckLibrary
	byLibrary = map[string]int{}
)

// RegisterLibrary records a check library. A library calls this from an init()
// in its own package, alongside the checks.Register calls for the check types
// it provides. Duplicate names panic, a programming error caught at startup.
func RegisterLibrary(lib CheckLibrary) {
	name := lib.Name()
	if _, dup := byLibrary[name]; dup {
		panic("checks: duplicate library registration for " + name)
	}
	byLibrary[name] = len(libraries)
	libraries = append(libraries, lib)
}

// Libraries returns every registered library in name order.
func Libraries() []CheckLibrary {
	out := make([]CheckLibrary, len(libraries))
	copy(out, libraries)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// LibraryByName returns the registered library with the given name, or
// (nil, false) if none is registered.
func LibraryByName(name string) (CheckLibrary, bool) {
	i, ok := byLibrary[name]
	if !ok {
		return nil, false
	}
	return libraries[i], true
}

// LibraryFor returns the library that owns a check type, resolved through the
// type's Descriptor.Library. It returns (nil, false) for an unknown kind or a
// kind whose Descriptor names no library (every native check type until it is
// migrated onto a library).
func LibraryFor(kind CheckType) (CheckLibrary, bool) {
	i, ok := byKind[kind]
	if !ok {
		return nil, false
	}
	name := registrations[i].desc.Library
	if name == "" {
		return nil, false
	}
	return LibraryByName(name)
}
