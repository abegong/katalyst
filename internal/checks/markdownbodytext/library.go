package markdownbodytext

import "github.com/abegong/katalyst/internal/checks"

// libraryName is the native CheckLibrary that provides this package's check
// types. Library is provenance (who runs the check), orthogonal to a check
// type's family (the source data it reads).
const libraryName = "markdownbodytext"

// library is the native CheckLibrary for the markdownbodytext check types. Native
// libraries run in-process and are always available.
type library struct{}

func (library) Name() string     { return libraryName }
func (library) Available() error { return nil }

func init() { checks.RegisterLibrary(library{}) }

// registerParsed is register for a check type that owns its config parsing.
func registerParsed(d checks.Descriptor, parse checks.Parser, build checks.ArgsBuilder, buildColl checks.CollectionArgsBuilder) {
	d.Library = libraryName
	checks.RegisterParsed(d, parse, build, buildColl)
}
