package filesystem

import "github.com/abegong/katalyst/internal/checks"

// libraryName is the native CheckLibrary that provides this package's check
// types. Library is provenance (who runs the check), orthogonal to a check
// type's family (the source data it reads).
const libraryName = "filesystem"

// library is the native CheckLibrary for the filesystem check types. Native
// libraries run in-process and are always available.
type library struct{}

func (library) Name() string     { return libraryName }
func (library) Available() error { return nil }

func init() { checks.RegisterLibrary(library{}) }

// register records a check type owned by this library, stamping
// Descriptor.Library so each check type's init need not repeat it.
func register(d checks.Descriptor, build checks.Builder, buildColl checks.CollectionBuilder) {
	d.Library = libraryName
	checks.Register(d, build, buildColl)
}

// registerParsed is register for a check type that owns its config parsing.
func registerParsed(d checks.Descriptor, parse checks.Parser, build checks.ArgsBuilder, buildColl checks.CollectionArgsBuilder) {
	d.Library = libraryName
	checks.RegisterParsed(d, parse, build, buildColl)
}
