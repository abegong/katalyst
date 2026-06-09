package checks

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FilenameMatchesSlug checks that the markdown filename equals the slug value.
type FilenameMatchesSlug struct {
	Field string
}

func (f FilenameMatchesSlug) Run(ctx Context) []Violation {
	ptr := "/" + f.Field
	raw, ok := ctx.Meta[f.Field]
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing frontmatter field %q", f.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	slug, ok := raw.(string)
	if !ok {
		return []Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("frontmatter field %q must be a string", f.Field),
			Line:    lookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	fileName := filepath.Base(ctx.FilePath)
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if base == slug {
		return nil
	}
	return []Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("slug %q must match filename %q", slug, base),
		Line:    lookupLine(ctx.Doc.Lines, ptr),
	}}
}
