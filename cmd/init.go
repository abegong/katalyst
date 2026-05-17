package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold a katabridge.yaml and an example schema.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("init: not implemented yet")
		},
	}
}
