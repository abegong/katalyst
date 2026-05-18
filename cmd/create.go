package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var noValidate bool
	var schemaFlag string
	var body string
	var overwrite bool

	c := &cobra.Command{
		Use:   "create <path> [key=value ...]",
		Short: "Create a markdown item with frontmatter attributes.",
		Long: `create writes a new markdown file with YAML frontmatter.

Assignments are YAML-decoded, so numbers/booleans/arrays are supported:
  katabridge create notes/dune.md title=Dune year=1965 tags='[sci-fi,classic]'`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			assignments := args[1:]

			if _, err := os.Stat(path); err == nil && !overwrite {
				return usageErr(path + " already exists (use --overwrite to replace)")
			}

			meta := map[string]any{}
			for _, a := range assignments {
				k, v, err := parseAssignment(a)
				if err != nil {
					return usageErr(err.Error())
				}
				meta[k] = v
			}

			out, err := composeMarkdown(meta, []byte(body))
			if err != nil {
				return err
			}

			if isMarkdownPath(path) {
				if err := validateWrite(path, out, schemaFlag, !noValidate); err != nil {
					return &exitError{code: exitValidationFail, msg: err.Error()}
				}
			}

			if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
				return err
			}
			return writeFileAtomic(path, out, 0o644)
		},
	}

	c.Flags().BoolVar(&noValidate, "no-validate", false, "Skip schema validation before writing")
	c.Flags().StringVarP(&schemaFlag, "schema", "s", "", "Path to a JSON Schema file. Overrides config-based schema resolution")
	c.Flags().StringVar(&body, "body", "", "Body content to place after frontmatter")
	c.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite destination file if it already exists")
	return c
}
