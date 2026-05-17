package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	s := &cobra.Command{
		Use:   "schema",
		Short: "Manage schemas.",
	}
	s.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List schemas registered in the config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("schema list: not implemented yet")
		},
	})
	return s
}
