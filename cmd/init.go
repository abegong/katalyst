package cmd

import (
	"fmt"
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
# Schemas live in .katalyst/schemas/<name>.yaml. Storage instances live in
# .katalyst/storage/<name>.yaml, and each instance declares the collections it
# maps. The settings below are optional and shown at their defaults; uncomment
# to change them.
#
# schemas:
#   discovery: convention   # convention | explicit
#   format: yaml            # yaml | json | both
# storage:
#   discovery: convention
#   format: yaml
`

// scaffoldLocalStorage is the default storage instance written by init: the
// local filesystem rooted at the project. There is no implicit instance,
// this file is what makes the default explicit. Collections are declared
// inline here (or split into .katalyst/storage/local/<name>.yaml).
const scaffoldLocalStorage = `# The default storage instance: the local filesystem, rooted at the project.
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

			for _, sub := range []string{"schemas", "storage"} {
				rel := filepath.Join(project.Dir, sub)
				if err := os.MkdirAll(filepath.Join(target, rel), 0o755); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s/\n", rel)
			}

			// Write the default storage instance explicitly; katalyst never
			// synthesizes one at runtime.
			storageRel := filepath.Join(project.Dir, "storage", "local.yaml")
			if err := os.WriteFile(filepath.Join(target, storageRel), []byte(scaffoldLocalStorage), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", storageRel)

			cfgRel := filepath.Join(project.Dir, "config.yaml")
			if err := os.WriteFile(filepath.Join(target, cfgRel), []byte(scaffoldConfig), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", cfgRel)
			return nil
		},
	}

	c.Flags().StringVar(&dir, "dir", "", "Directory to prepare (default: current directory)")
	return c
}
