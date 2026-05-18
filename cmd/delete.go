package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	c := &cobra.Command{
		Use:   "delete <path> [path...]",
		Short: "Delete item files.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, p := range args {
				info, err := os.Stat(p)
				if err != nil {
					if force && os.IsNotExist(err) {
						continue
					}
					return usageErr(fmt.Sprintf("delete: %v", err))
				}
				if info.IsDir() {
					return usageErr(fmt.Sprintf("delete: %s is a directory (files only)", p))
				}
				if err := os.Remove(p); err != nil {
					return err
				}
			}
			return nil
		},
	}

	c.Flags().BoolVarP(&force, "force", "f", false, "Ignore missing files")
	return c
}
