package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/katabase-ai/katabridge/internal/frontmatter"
	"github.com/katabase-ai/katabridge/internal/validator"
	"github.com/spf13/cobra"
)

// Exit codes for `validate`. Loosely modeled on shellcheck and on the
// `jv` CLI from santhosh-tekuri/jsonschema.
const (
	exitOK             = 0
	exitValidationFail = 1
	exitUsage          = 2
)

func newValidateCmd() *cobra.Command {
	var schemaPath string

	c := &cobra.Command{
		Use:   "validate [paths...]",
		Short: "Validate markdown frontmatter against a JSON Schema.",
		Long: `Validate parses YAML frontmatter from each given markdown file
and checks it against the provided JSON Schema.

In v0.1 the schema must be supplied explicitly via --schema. Config-driven
schema association (glob → schema) is tracked in product/decisions-to-make.md
and will land in a later release.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if schemaPath == "" {
				return usageErr("--schema is required (config-driven association is not implemented yet)")
			}

			schema, err := loadSchemaFile(schemaPath)
			if err != nil {
				return usageErr(err.Error())
			}

			anyInvalid := false
			for _, path := range args {
				ok, err := validateFile(cmd.OutOrStdout(), cmd.ErrOrStderr(), schema, path)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", path, err)
					anyInvalid = true
					continue
				}
				if !ok {
					anyInvalid = true
				}
			}

			if anyInvalid {
				// Returning a typed error lets us control the exit code
				// from main without printing the error twice.
				return &exitError{code: exitValidationFail}
			}
			return nil
		},
	}

	c.Flags().StringVarP(&schemaPath, "schema", "s", "", "Path to a JSON Schema file (required)")
	return c
}

// validateFile reads one markdown file, extracts its frontmatter, validates
// it, and writes results to out/errOut. It returns (true, nil) if the file
// is valid, (false, nil) if it has validation errors, or (_, err) if the
// file couldn't be read/parsed at all.
func validateFile(out, errOut io.Writer, schema *validator.Schema, path string) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	doc, err := frontmatter.Parse(src)
	if err != nil {
		return false, err
	}

	if !doc.HasFrontmatter {
		fmt.Fprintf(errOut, "%s: no frontmatter found\n", path)
		return false, nil
	}

	result := schema.Validate(doc.Meta)
	if result.Valid {
		fmt.Fprintf(out, "%s: OK\n", path)
		return true, nil
	}

	for _, e := range result.Errors {
		loc := e.Path
		if loc == "" {
			loc = "/"
		}
		fmt.Fprintf(errOut, "%s: %s: %s\n", path, loc, e.Message)
	}
	return false, nil
}

func loadSchemaFile(path string) (*validator.Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open schema: %w", err)
	}
	defer f.Close()
	return validator.Load(path, f)
}

// usageErr wraps an error so main can exit with code 2 (usage error).
func usageErr(msg string) error {
	return &exitError{code: exitUsage, msg: msg}
}

// exitError carries an explicit process exit code.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string {
	if e.msg == "" {
		return fmt.Sprintf("exit %d", e.code)
	}
	return e.msg
}

// Code returns the desired process exit code.
func (e *exitError) Code() int { return e.code }
