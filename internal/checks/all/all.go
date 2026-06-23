// Package all blank-imports every check-type family so their init() functions
// register with the core checks registry. Import it for its side effects
// wherever the full catalog must be present: the engine, the docs generator,
// the check-types command, and registry tests:
//
//	import _ "github.com/abegong/katalyst/internal/checks/all"
//
// The core checks package imports none of the families (the dependency runs the
// other way), so without this aggregator the registry would be empty.
package all

import (
	_ "github.com/abegong/katalyst/internal/checks/filesystem"
	_ "github.com/abegong/katalyst/internal/checks/markdownbodytext"
	_ "github.com/abegong/katalyst/internal/checks/plaintext"
	_ "github.com/abegong/katalyst/internal/checks/structuredobject"
)
