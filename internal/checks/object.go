package checks

import "github.com/katabase-ai/katalyst/internal/validator"

// Object validates frontmatter metadata against JSON Schema.
type Object struct {
	Schema *validator.Schema
}

func (o Object) Run(ctx Context) []Violation {
	result := o.Schema.Validate(ctx.Meta)
	if result.Valid {
		return nil
	}
	out := make([]Violation, 0, len(result.Errors))
	for _, err := range result.Errors {
		out = append(out, Violation{
			Path:    err.Path,
			Message: err.Message,
			Line:    lookupLine(ctx.Doc.Lines, err.Path),
		})
	}
	return out
}
