package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/abegong/katalyst/internal/config"
	"github.com/spf13/cobra"
)

// scaffoldConfig is the commented-template config.yaml written by init.
// Every setting is shown at its default and commented out, so a fresh
// project loads as an empty, valid configuration while the file documents
// the available knobs. See product/specs/project-layout-spec.md.
const scaffoldConfig = `# katalyst project configuration.
#
# Schemas live in .katalyst/schemas/<name>.yaml and collections in
# .katalyst/collections/<name>.yaml, discovered by filename. The settings
# below are optional and shown at their defaults; uncomment to change them.
#
# schemas:
#   discovery: convention   # convention | explicit
#   format: yaml            # yaml | json | both
# collections:
#   discovery: convention
#   format: yaml
`

func newInitCmd() *cobra.Command {
	var dir string

	c := &cobra.Command{
		Use:   "init",
		Short: "Prepare the current directory as a katalyst project.",
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
			katalystDir := filepath.Join(target, config.Dir)
			if _, err := os.Stat(katalystDir); err == nil {
				return usageErr(fmt.Sprintf("%s already exists; refusing to overwrite", katalystDir))
			}

			for _, sub := range []string{"schemas", "collections"} {
				rel := filepath.Join(config.Dir, sub)
				if err := os.MkdirAll(filepath.Join(target, rel), 0o755); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s/\n", rel)
			}

			cfgRel := filepath.Join(config.Dir, "config.yaml")
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
