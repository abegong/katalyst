package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Argument-arity validators that replace Cobra's "accepts N arg(s), received M"
// text with the project's usage-error grammar (exit 2). usage is the command's
// argument spec, e.g. "inspect <path>", rendered into the usage hint. See
// cmd/AGENTS.md, "Error messages", for the standard.

func exactArgs(n int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error { return checkArity(len(args), n, n, usage) }
}

func minArgs(n int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error { return checkArity(len(args), n, -1, usage) }
}

func maxArgs(n int, usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error { return checkArity(len(args), 0, n, usage) }
}

// checkArity returns a standard usage error when got is outside [min, max]
// (max < 0 means unbounded).
func checkArity(got, min, max int, usage string) error {
	switch {
	case got < min:
		return usageErr("missing argument(s) (usage: katalyst " + usage + ")")
	case max >= 0 && got > max:
		return usageErr("too many arguments (usage: katalyst " + usage + ")")
	default:
		return nil
	}
}

// unknownCollectionErr is the standard not-found message for a collection,
// carrying a discovery hint.
func unknownCollectionErr(name string) error {
	return usageErr(fmt.Sprintf("unknown collection %q (try `katalyst collection list`)", name))
}
