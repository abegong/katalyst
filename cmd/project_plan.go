package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/abegong/katalyst/internal/project"
	"github.com/spf13/cobra"
)

func newProjectPlanCmd() *cobra.Command {
	var flags projectPlanFlags
	c := &cobra.Command{
		Use:   "plan",
		Short: "Print the resolved project authority plan",
		Args:  maxArgs(0, "project plan"),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := project.BuildPlan(project.PlanOptions{
				ConfigPath:          flags.configPath,
				ProjectDir:          flags.projectDir,
				DisableNestedConfig: flags.disableNestedConfig,
			})
			if err != nil {
				return asUsageErr(err)
			}
			printProjectPlan(cmd.OutOrStdout(), plan)
			return nil
		},
	}
	addProjectPlanFlags(c, &flags)
	return c
}

func printProjectPlan(w io.Writer, plan *project.Plan) {
	printSectionHeader(w, "Project plan")
	fmt.Fprintf(w, "- root: %s\n", project.Dir)
	if len(plan.Delegates) == 0 {
		fmt.Fprintln(w, "- nested configs: none")
		return
	}
	fmt.Fprintf(w, "- nested configs: %d\n", len(plan.Delegates))
	for _, delegate := range plan.Delegates {
		configPath := filepath.ToSlash(filepath.Join(delegate.Delegate.Path, delegate.Delegate.Config))
		status := "inactive"
		if delegate.Active {
			status = "active"
		}
		fmt.Fprintf(w, "\n%s\n", configPath)
		fmt.Fprintln(w, "----")
		fmt.Fprintf(w, "- status: %s\n", status)
		for _, line := range authorityLines(plan.Root.NestedConfigs, delegate.Delegate) {
			fmt.Fprintf(w, "- %s\n", line)
		}
	}
}

func authorityLines(settings project.NestedConfigSettings, delegate project.NestedDelegate) []string {
	subsystems := project.AuthoritySubsystems()
	sort.Slice(subsystems, func(i, j int) bool { return subsystems[i] < subsystems[j] })
	out := make([]string, 0, len(subsystems))
	for _, subsystem := range subsystems {
		out = append(out, fmt.Sprintf("%s: %s", subsystem, settings.AuthorityFor(delegate, subsystem)))
	}
	return out
}
