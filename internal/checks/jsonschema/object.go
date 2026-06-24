package jsonschema

import (
	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project/config"
)

// Object validates frontmatter metadata against a compiled JSON Schema. It is
// the per-item check type the JSON Schema library provides.
type Object struct {
	Schema checks.Schema
}

func (o Object) Run(ctx checks.Context) []checks.Violation {
	return o.Schema.Check(ctx)
}

// The object check has no registry Builder: the engine constructs it specially
// because it needs a schema compiled (and cached) from --schema, an inline
// key, or the collection's config (see Resolve). It still registers its
// Descriptor so it is documented and parity-checked, and registers the JSON
// Schema library that owns it.
func init() {
	checks.RegisterLibrary(Library{})
	checks.Register(checks.Descriptor{
		CheckType: config.CheckObject,
		Library:   "json-schema",
		Family:    "structuredObject",
		Slug:      "object",
		Title:     "Object validation",
		Summary:   "Validate frontmatter metadata against a named JSON Schema from `schemas:`.",
		Fields: []checks.Field{
			{Name: "schema", Required: true, Desc: "Name of an entry in `schemas:`."},
		},
		ConfigExample: `schemas:
  book: ./schemas/book.json
collections:
  notes:
    path: notes
    checks:
      - kind: object
        schema: book`,
	}, nil, nil)
}
