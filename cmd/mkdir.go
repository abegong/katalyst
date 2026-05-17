package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newMkdirCmd() *cobra.Command {
	var parents bool

	c := &cobra.Command{
		Use:   "mkdir <dir> [dir...]",
		Short: "Create directories.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, dir := range args {
				var err error
				if parents {
					err = os.MkdirAll(dir, 0o755)
				} else {
					err = os.Mkdir(dir, 0o755)
				}
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	c.Flags().BoolVarP(&parents, "parents", "p", false, "No error if existing, make parent directories as needed")
	return c
}
