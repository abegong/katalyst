package jsonschema

import (
	"fmt"

	"github.com/abegong/katalyst/internal/project/config"
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
// The forced path and inline name are this library's sugar; cfg resolves a
// schema name to its absolute path. An inline or collection name that is not
// defined under .katalyst/schemas/ is an error.
func Resolve(forcedPath, inlineName string, effective []config.CheckInstance, cfg *config.Config) ([]SchemaRef, error) {
	switch {
	case forcedPath != "":
		return []SchemaRef{{Name: forcedPath, Path: forcedPath}}, nil
	case inlineName != "":
		path := cfg.SchemaPath(inlineName)
		if path == "" {
			return nil, fmt.Errorf("inline schema %q is not defined under .katalyst/schemas/", inlineName)
		}
		return []SchemaRef{{Name: inlineName, Path: path}}, nil
	default:
		var refs []SchemaRef
		for _, ch := range effective {
			if ch.Type != config.CheckObject {
				continue
			}
			path := cfg.SchemaPath(ch.Schema)
			if path == "" {
				return nil, fmt.Errorf("collection object schema %q is not defined under .katalyst/schemas/", ch.Schema)
			}
			refs = append(refs, SchemaRef{Name: ch.Schema, Path: path})
		}
		return refs, nil
	}
}
