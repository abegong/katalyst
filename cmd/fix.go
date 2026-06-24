package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/abegong/katalyst/internal/fix"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/storage/collection/filesystem"
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
missing required keys. See docs/content/deep-dives/formatting.md for why.

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
				didChange, err := fixOne(item.Path, item.Collection, checkOnly)
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

// fixOne reports whether path's content would change. It computes the fixed
// content with the backend-agnostic fix engine and, unless check is set,
// persists it through the filesystem backend (an atomic replace). The split is
// deliberate: deciding what to write is fix's, writing it is the backend's.
func fixOne(path string, c project.Collection, check bool) (changed bool, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	result, err := fix.Apply(src, c)
	if err != nil {
		return false, err
	}
	if bytes.Equal(src, result) {
		return false, nil
	}
	if check {
		return true, nil
	}
	if err := filesystem.Write(path, result); err != nil {
		return false, err
	}
	return true, nil
}
