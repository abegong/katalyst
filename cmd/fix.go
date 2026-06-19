package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/spf13/cobra"
)

func newFixCmd() *cobra.Command {
	var checkOnly bool

	c := &cobra.Command{
		Use:   "fix [selector ...]",
		Short: "Apply deterministic, safe fixes to the selected items.",
		Long: `fix rewrites each selected item's frontmatter in a canonical form:
top-level keys sorted alphabetically, yaml.v3 default block style, and
exactly one trailing newline. The body is preserved verbatim.

fix never invents semantic values: it will not inject placeholders for
missing required keys. See docs/explanation/formatting.md for why.

Selectors follow the same grammar as 'check'. With no selector, every
item in the project is considered.

With --check, no files are modified; instead, items that would change are
printed and the command exits with status 1. Use this in CI.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			res, err := resolveSelectors(projectFor(cfg), args)
			if err != nil {
				return err
			}

			changed := false
			for _, item := range res.Items {
				didChange, err := formatOne(item.Path, checkOnly)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", item.Path, err)
					return &exitError{code: exitValidationFail}
				}
				if didChange {
					changed = true
					fmt.Fprintln(cmd.OutOrStdout(), item.Path)
				}
			}
			if checkOnly && changed {
				return &exitError{code: exitValidationFail}
			}
			return nil
		},
	}

	c.Flags().BoolVar(&checkOnly, "check", false,
		"Don't write; exit 1 if any item would change (for CI).")
	return c
}

// formatOne returns whether path's content would change. When check is
// false, the file is rewritten in place if its formatted form differs.
func formatOne(path string, check bool) (changed bool, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	formatted, err := frontmatter.Format(src)
	if err != nil {
		return false, err
	}
	if bytes.Equal(src, formatted) {
		return false, nil
	}
	if check {
		return true, nil
	}
	// Write atomically: write to a sibling temp file, then rename.
	tmp, err := os.CreateTemp(filepathDir(path), ".katalyst-fix-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(formatted); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}

// filepathDir returns the directory of path, defaulting to "." when path
// has no separator. Used to keep atomic temp files on the same filesystem.
func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
