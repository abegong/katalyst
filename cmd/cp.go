package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCpCmd() *cobra.Command {
	var noValidate bool
	var schemaFlag string

	c := &cobra.Command{
		Use:   "cp <src> <dst>",
		Short: "Copy a file, optionally validating markdown writes.",
		Long: `cp copies src to dst.

When dst is an existing directory, the basename of src is appended.
For markdown destinations (*.md), strict mode validates the resulting
document against schema rules before writing.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcPath, dstPath := args[0], args[1]

			info, err := os.Stat(srcPath)
			if err != nil {
				return usageErr(fmt.Sprintf("cp: %v", err))
			}
			if info.IsDir() {
				return usageErr("cp: directory copies are not implemented yet (file paths only)")
			}

			src, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}

			if dstInfo, err := os.Stat(dstPath); err == nil && dstInfo.IsDir() {
				dstPath = filepath.Join(dstPath, filepath.Base(srcPath))
			}

			if isMarkdownPath(dstPath) {
				if err := validateWrite(dstPath, src, schemaFlag, !noValidate); err != nil {
					return &exitError{code: exitValidationFail, msg: err.Error()}
				}
			}

			if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
				return err
			}
			if err := writeFileAtomic(dstPath, src, info.Mode().Perm()); err != nil {
				return err
			}
			return nil
		},
	}

	c.Flags().BoolVar(&noValidate, "no-validate", false, "Skip schema validation before writing markdown destination files")
	c.Flags().StringVarP(&schemaFlag, "schema", "s", "", "Path to a JSON Schema file. Overrides config-based schema resolution")
	return c
}
