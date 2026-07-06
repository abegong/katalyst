package cmd

import (
	"github.com/spf13/cobra"
)

type projectPlanFlags struct {
	configPath          string
	projectDir          string
	disableNestedConfig bool
}

func newProjectCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "project",
		Short: "Commands to inspect the active project",
	}
	c.AddCommand(newProjectPlanCmd())
	return c
}

func addProjectPlanFlags(c *cobra.Command, flags *projectPlanFlags) {
	c.Flags().StringVar(&flags.configPath, "config", "", "Project root, .katalyst directory, or .katalyst/config.yaml to load")
	c.Flags().StringVar(&flags.projectDir, "project", "", "Directory used to select the active project")
	c.Flags().BoolVar(&flags.disableNestedConfig, "disable-nested-config", false, "Ignore nestedConfigs in the active root")
}
