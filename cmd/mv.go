package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newMvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <src> <dst>",
		Short: "Move or rename a file or directory.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, dst := args[0], args[1]
			if _, err := os.Stat(src); err != nil {
				return usageErr(fmt.Sprintf("mv: %v", err))
			}
			if dstInfo, err := os.Stat(dst); err == nil && dstInfo.IsDir() {
				dst = filepath.Join(dst, filepath.Base(src))
			}
			return os.Rename(src, dst)
		},
	}
}
