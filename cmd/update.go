package cmd

import (
	"fmt"
	"os"

	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var noValidate bool
	var schemaFlag string

	c := &cobra.Command{
		Use:   "update <path> key=value [key=value...]",
		Short: "Update frontmatter attributes on a markdown item.",
		Long: `update modifies top-level frontmatter attributes in-place.

Values are YAML-decoded, so numbers/booleans/arrays are supported:
  katalyst update notes/dune.md year=1965 published=true tags='[sci-fi,classic]'`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			assignments := args[1:]

			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			doc, err := frontmatter.Parse(src)
			if err != nil {
				return err
			}
			if !doc.HasFrontmatter {
				return usageErr(fmt.Sprintf("%s: no frontmatter found", path))
			}

			meta := make(map[string]any, len(doc.Meta))
			for k, v := range doc.Meta {
				meta[k] = v
			}
			for _, a := range assignments {
				k, v, err := parseAssignment(a)
				if err != nil {
					return usageErr(err.Error())
				}
				meta[k] = v
			}

			out, err := composeMarkdown(meta, doc.Body)
			if err != nil {
				return err
			}

			if err := validateWrite(path, out, schemaFlag, !noValidate); err != nil {
				return &exitError{code: exitValidationFail, msg: err.Error()}
			}

			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			return writeFileAtomic(path, out, info.Mode().Perm())
		},
	}

	c.Flags().BoolVar(&noValidate, "no-validate", false, "Skip schema validation before writing")
	c.Flags().StringVarP(&schemaFlag, "schema", "s", "", "Path to a JSON Schema file. Overrides config-based schema resolution")
	return c
}
