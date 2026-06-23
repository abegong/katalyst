package structuredobject

import (
	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
	"github.com/abegong/katalyst/internal/validator"
)

// Object validates frontmatter metadata against JSON Schema.
type Object struct {
	Schema *validator.Schema
}

func (o Object) Run(ctx checks.Context) []checks.Violation {
	result := o.Schema.Validate(ctx.Meta)
	if result.Valid {
		return nil
	}
	out := make([]checks.Violation, 0, len(result.Errors))
	for _, err := range result.Errors {
		out = append(out, checks.Violation{
			Path:    err.Path,
			Message: err.Message,
			Line:    checks.LookupLine(ctx.Doc.Lines, err.Path),
		})
	}
	return out
}

// The object check has no Builder: the engine constructs it specially because
// it needs a schema compiled (and cached) from --schema, an inline key, or the
// collection's config. It still registers its Descriptor so it is documented
// and parity-checked.
func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckObject,
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
