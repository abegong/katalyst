package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project/config"
)

// ReferencedFilesExist checks that path-valued frontmatter fields resolve to
// real files, relative to the item's own directory.
type ReferencedFilesExist struct {
	Fields []string
}

func (c ReferencedFilesExist) Run(ctx checks.Context) []checks.Violation {
	dir := filepath.Dir(ctx.FilePath)
	var out []checks.Violation
	for _, field := range c.Fields {
		raw, ok := ctx.Meta[field]
		if !ok {
			continue // absent field is not this check's concern
		}
		paths, ok := stringOrList(raw)
		if !ok {
			ptr := "/" + field
			out = append(out, checks.Violation{
				Path:    ptr,
				Message: fmt.Sprintf("frontmatter field %q must be a path string or list of strings", field),
				Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
			})
			continue
		}
		for _, p := range paths {
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p))); err != nil {
				ptr := "/" + field
				out = append(out, checks.Violation{
					Path:    ptr,
					Message: fmt.Sprintf("referenced file %q does not exist", p),
					Line:    checks.LookupLine(ctx.Doc.Lines, ptr),
				})
			}
		}
	}
	return out
}

// stringOrList coerces a frontmatter value into a list of strings, accepting
// a bare string or a list of strings. Returns false for any other shape.
func stringOrList(v any) ([]string, bool) {
	switch t := v.(type) {
	case string:
		return []string{t}, true
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	case []string:
		return t, true
	default:
		return nil, false
	}
}

func init() {
	register(checks.Descriptor{
		CheckType: config.CheckFilesystemReferencedFiles,
		Family:    "fileSystem",
		Slug:      "referenced-files-exist",
		Title:     "Referenced files exist",
		Summary:   "Require path-valued frontmatter fields to resolve to real files.",
		Fields: []checks.Field{
			{Name: "fields", Required: true, Desc: "Frontmatter keys holding a path (string) or list of paths, resolved relative to the item."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_referenced_files_exist
        fields: [cover, attachments]`,
	}, func(ch config.CheckInstance) checks.Check {
		return ReferencedFilesExist{Fields: ch.Fields}
	}, nil)
}
