package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/abegong/katalyst/internal/project"
	"github.com/spf13/cobra"
)

// scaffoldConfig is the commented-template config.yaml written by init.
// Every setting is shown at its default and commented out, so a fresh
// project loads as an empty, valid configuration while the file documents
// the available knobs.
const scaffoldConfig = `# katalyst project configuration.
#
# Schemas live in .katalyst/schemas/<name>.yaml. Bases live in
# .katalyst/bases/<name>.yaml, and each base declares the collections it maps.
# The settings below are optional and shown at their defaults; uncomment to
# change them.
#
# schemas:
#   discovery: convention   # convention | explicit
#   format: yaml            # yaml | json | both
# bases:
#   discovery: convention
#   format: yaml
`

// scaffoldLocalBase is the default base written by init: the local filesystem
// rooted at the project. There is no implicit base, this file is what makes the
// default explicit. Collections are declared inline here (or split into
// .katalyst/bases/local/<name>.yaml).
const scaffoldLocalBase = `# The default base: the local filesystem, rooted at the project.
# Declare collections under "collections:", e.g.
#
#   collections:
#     notes:
#       path: notes          # directory, relative to root
#       schema: note         # a schema name from .katalyst/schemas/
type: filesystem
root: .
collections: {}
`

func newInitCmd() *cobra.Command {
	var dir string
	var nested bool

	c := &cobra.Command{
		Use:   "init",
		Short: "Prepare the current directory as a katalyst project",
		Args:  maxArgs(0, "init"),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := dir
			if target == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				target = wd
			}

			// Refuse to touch an existing project. Checking the .katalyst/
			// dir up front keeps init all-or-nothing.
			katalystDir := filepath.Join(target, project.Dir)
			if _, err := os.Stat(katalystDir); err == nil {
				return usageErr(fmt.Sprintf("%s already exists; refusing to overwrite", katalystDir))
			}
			parentRoot, parentErr := project.FindParentRoot(target)
			hasParent := parentErr == nil
			if parentErr != nil && !errors.Is(parentErr, project.ErrNotFound) {
				return parentErr
			}
			if hasParent && !nested {
				parentConfig, err := filepath.Rel(target, filepath.Join(parentRoot, project.Dir))
				if err != nil {
					parentConfig = filepath.Join(parentRoot, project.Dir)
				}
				return usageErr(fmt.Sprintf("found parent Katalyst project at %s; refusing to create an implicit nested project (rerun with: katalyst init --nested)", filepath.ToSlash(parentConfig)))
			}

			for _, sub := range []string{"schemas", "bases"} {
				rel := filepath.Join(project.Dir, sub)
				if err := os.MkdirAll(filepath.Join(target, rel), 0o755); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s/\n", rel)
			}

			// Write the default base explicitly; katalyst never synthesizes one
			// at runtime.
			baseRel := filepath.Join(project.Dir, "bases", "local.yaml")
			if err := os.WriteFile(filepath.Join(target, baseRel), []byte(scaffoldLocalBase), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", baseRel)

			cfgRel := filepath.Join(project.Dir, "config.yaml")
			if err := os.WriteFile(filepath.Join(target, cfgRel), []byte(scaffoldConfig), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", cfgRel)
			if hasParent {
				printNestedInitSnippet(cmd.OutOrStdout(), parentRoot, target)
			}
			return nil
		},
	}

	c.Flags().StringVar(&dir, "dir", "", "Directory to prepare (default: current directory)")
	c.Flags().BoolVar(&nested, "nested", false, "Create a nested project inside an existing Katalyst project")
	return c
}

func printNestedInitSnippet(out io.Writer, parentRoot, target string) {
	if resolved, err := filepath.EvalSymlinks(target); err == nil {
		target = resolved
	}
	rel, err := filepath.Rel(parentRoot, target)
	if err != nil {
		rel = target
	}
	rel = filepath.ToSlash(rel)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "nested project created")
	fmt.Fprintln(out, "- commands run from this subtree use the child config")
	fmt.Fprintln(out, "- commands run from the parent root keep using the parent config")
	fmt.Fprintln(out, "- parent-root runs ignore the child config until the parent delegates to it")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "add this to the parent config to delegate authority:")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "nestedConfigs:")
	fmt.Fprintln(out, "  discovery: explicit")
	fmt.Fprintln(out, "  delegates:")
	fmt.Fprintf(out, "    - path: %s\n", rel)
	fmt.Fprintln(out, "      config: .katalyst")
	fmt.Fprintln(out, "      authority:")
	fmt.Fprintln(out, "        collections: file_nearest")
	fmt.Fprintln(out, "        collectionChecks: file_nearest")
	fmt.Fprintln(out, "        schemas: file_nearest")
}
