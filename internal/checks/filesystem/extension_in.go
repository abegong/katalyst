package filesystem

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/config"
)

// FilesystemExtensionIn checks that extension is in an allowed set.
type FilesystemExtensionIn struct {
	Values []string
}

func (f FilesystemExtensionIn) Run(ctx checks.Context) []checks.Violation {
	ext := strings.ToLower(filepath.Ext(ctx.FilePath))
	for _, allowed := range f.Values {
		a := strings.ToLower(strings.TrimSpace(allowed))
		if !strings.HasPrefix(a, ".") {
			a = "." + a
		}
		if ext == a {
			return nil
		}
	}
	return []checks.Violation{{
		Path:    "/",
		Message: fmt.Sprintf("file extension %q is not in allowed set", ext),
	}}
}

func init() {
	checks.Register(checks.Descriptor{
		CheckType: config.CheckFilesystemExtensionIn,
		Family:    "fileSystem",
		Slug:      "extension-in",
		Title:     "Extension in",
		Summary:   "Allow only specific file extensions.",
		Fields: []checks.Field{
			{Name: "values", Required: true, Desc: "Allowed extensions, including the leading dot."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    pattern: "*"
    checks:
      - kind: filesystem_extension_in
        values: [.md, .markdown]`,
	}, func(ch config.CheckInstance) checks.Check {
		return FilesystemExtensionIn{Values: ch.Values}
	}, nil)
}
