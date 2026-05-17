package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var recursive bool
	var force bool

	c := &cobra.Command{
		Use:   "rm <path> [path...]",
		Short: "Remove files or directories.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, p := range args {
				info, err := os.Stat(p)
				if err != nil {
					if force && os.IsNotExist(err) {
						continue
					}
					return usageErr(fmt.Sprintf("rm: %v", err))
				}

				if info.IsDir() {
					if !recursive {
						return usageErr(fmt.Sprintf("rm: %s is a directory (use -r)", p))
					}
					if err := os.RemoveAll(p); err != nil {
						return err
					}
					continue
				}
				if err := os.Remove(p); err != nil {
					return err
				}
			}
			return nil
		},
	}

	c.Flags().BoolVarP(&recursive, "recursive", "r", false, "Remove directories and their contents recursively")
	c.Flags().BoolVarP(&force, "force", "f", false, "Ignore missing files and never prompt")
	return c
}
