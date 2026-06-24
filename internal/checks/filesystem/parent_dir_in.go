package filesystem

import (
	"fmt"
	"path/filepath"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/argcheck"
)

// FilesystemParentDirIn checks that parent directory name is in allowed values.
type FilesystemParentDirIn struct {
	Values []string
}

func (f FilesystemParentDirIn) Run(ctx checks.Context) []checks.Violation {
	parent := filepath.Base(filepath.Dir(ctx.FilePath))
	for _, allowed := range f.Values {
		if parent == allowed {
			return nil
		}
	}
	return []checks.Violation{{
		Path:    "/",
		Message: fmt.Sprintf("parent directory %q is not in allowed set", parent),
	}}
}

type parentDirInArgs struct {
	Values []string `yaml:"values"`
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType: checks.CheckFilesystemParentDirIn,
		Family:    "fileSystem",
		Slug:      "parent-dir-in",
		Title:     "Parent directory in",
		Summary:   "Require that the file's parent directory name is in an allowed set.",
		Fields: []checks.Field{
			{Name: "values", Required: true, Desc: "Allowed parent directory names."},
		},
		ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_parent_dir_in
        values: [books, people]`,
	}, checks.ParseInto(func(a parentDirInArgs) error {
		return argcheck.RequireStrings("filesystem_parent_dir_in", "values", a.Values)
	}), func(a any) checks.Check {
		return FilesystemParentDirIn{Values: a.(parentDirInArgs).Values}
	}, nil)
}
