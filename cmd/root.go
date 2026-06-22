// Package cmd contains the cobra command tree for the katalyst CLI.
package cmd

import "github.com/spf13/cobra"

// Version is the CLI version. Overridden at build time via -ldflags.
var Version = "0.0.0-dev"

// NewRootCmd builds the root command and attaches all subcommands.
//
// Using a constructor (rather than a package-level var) keeps tests
// hermetic: each test can build its own command tree with its own flags
// and I/O streams.
func NewRootCmd() *cobra.Command {
	// Preserve insertion order in help output so command grouping/order can
	// communicate the intended workflow instead of alphabetical sorting.
	cobra.EnableCommandSorting = false

	root := &cobra.Command{
		Use:   "katalyst",
		Short: "Inspect, check, and fix content consistency rules.",
		Long: `katalyst is a content consistency layer for agent memory,
knowledge bases, and other curated content systems. it helps you inspect,
check, and fix content and metadata conventions.

Project:
  GitHub: https://github.com/abegong/katalyst
  Docs:   https://abegong.github.io/katalyst/`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Route flag-parse failures (unknown flag, missing value) through the
	// project's usage-error machinery so they exit 2 in the standard voice
	// instead of Cobra's default exit-1 text. Subcommands inherit this.
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return usageErr(err.Error())
	})

	// The command tree is two grammars (see cmd/AGENTS.md and
	// docs/deep-dives/command-organization.md): verbs operate over content via
	// selectors; resource nouns carry CRUD sub-verbs. Grouping the help output
	// makes that split visible rather than alphabetizing them together.
	root.AddGroup(
		&cobra.Group{ID: "verbs", Title: "Verbs:"},
		&cobra.Group{ID: "resources", Title: "Resources:"},
	)

	addGrouped(root, "verbs",
		newInitCmd(),
		newInspectCmd(),
		newCheckCmd(),
		newFixCmd(),
	)
	addGrouped(root, "resources",
		newCheckTypesCmd(),
		newCollectionCmd(),
		newInspectorsCmd(),
		newItemCmd(),
		newSchemaCmd(),
	)

	return root
}

// addGrouped attaches each command to root under the given help group.
func addGrouped(root *cobra.Command, groupID string, cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.GroupID = groupID
		root.AddCommand(c)
	}
}
