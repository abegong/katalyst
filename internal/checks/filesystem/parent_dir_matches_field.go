package filesystem

import (
	"fmt"
	"path/filepath"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project/config"
)

// ParentDirMatchesField checks that the parent directory name equals a
// frontmatter field.
type ParentDirMatchesField struct {
	Field string
}

func (c ParentDirMatchesField) Run(ctx checks.Context) []checks.Violation {
	ptr := "/" + c.Field
	raw, ok := ctx.Meta[c.Field]
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("missing frontmatter field %q", c.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	want, ok := raw.(string)
	if !ok {
		return []checks.Violation{{
			Path:    ptr,
			Message: fmt.Sprintf("frontmatter field %q must be a string", c.Field),
			Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
		}}
	}
	parent := filepath.Base(filepath.Dir(ctx.FilePath))
	if parent == want {
		return nil
	}
	return []checks.Violation{{
		Path:    ptr,
		Message: fmt.Sprintf("parent directory %q must match field %q (%q)", parent, c.Field, want),
		Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
	}}
}

func init() {
	register(checks.Descriptor{
		CheckType: config.CheckFilesystemParentDirMatchesFld,
		Family:    "fileSystem",
		Slug:      "parent-dir-matches-field",
		Title:     "Parent directory matches field",
		Summary:   "Require the parent directory name to equal a frontmatter field.",
		Fields: []checks.Field{
			{Name: "field", Required: true, Desc: "Frontmatter key compared to the parent directory name."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_parent_dir_matches_field
        field: category`,
	}, func(ch config.CheckInstance) checks.Check {
		return ParentDirMatchesField{Field: ch.Field}
	}, nil)
}
