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
	root := &cobra.Command{
		Use:   "katalyst",
		Short: "Define and enforce schemas for markdown frontmatter.",
		Long: `katalyst validates structured metadata (frontmatter) on
markdown files against JSON Schema documents.`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newInitCmd(),
		newValidateCmd(),
		newSchemaCmd(),
		newFmtCmd(),
		newCreateCmd(),
		newReadCmd(),
		newUpdateCmd(),
		newDeleteCmd(),
	)

	return root
}
