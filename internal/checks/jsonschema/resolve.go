package jsonschema

import (
	"fmt"

	"github.com/abegong/katalyst/internal/checks"
)

// SchemaRef is a resolved object schema: its display name (used as the
// compiled schema's identifier and cache lookup) and its absolute file path.
type SchemaRef struct {
	Name string
	Path string
}

// Resolve returns the object schemas that apply to an item, in precedence
// order (highest first):
//
//  1. A forced --schema path, applied to every item regardless of config.
//  2. An inline "schema:" name in the item's frontmatter.
//  3. The collection's configured object checks.
//
// The forced path and inline name are this library's sugar; schemaPath resolves
// a schema name to its absolute path ("" when undefined). An inline or
// collection name that is not defined under .katalyst/schemas/ is an error.
func Resolve(forcedPath, inlineName string, effective []checks.ConfiguredCheck, schemaPath func(string) string) ([]SchemaRef, error) {
	switch {
	case forcedPath != "":
		return []SchemaRef{{Name: forcedPath, Path: forcedPath}}, nil
	case inlineName != "":
		path := schemaPath(inlineName)
		if path == "" {
			return nil, fmt.Errorf("inline schema %q is not defined under .katalyst/schemas/", inlineName)
		}
		return []SchemaRef{{Name: inlineName, Path: path}}, nil
	default:
		var refs []SchemaRef
		for _, cc := range effective {
			if cc.Kind != checks.CheckObject {
				continue
			}
			path := schemaPath(cc.Schema)
			if path == "" {
				return nil, fmt.Errorf("collection object schema %q is not defined under .katalyst/schemas/", cc.Schema)
			}
			refs = append(refs, SchemaRef{Name: cc.Schema, Path: path})
		}
		return refs, nil
	}
}
