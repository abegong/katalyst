package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Scaffold contents are kept as package-level strings (rather than
// embedded files) so they're trivial to evolve in tests. They are
// intentionally minimal and self-consistent: running `validate` on the
// scaffold immediately after `init` succeeds.
const (
	scaffoldConfig = `# katabridge configuration.
# See product/decisions.md for the schema and rule semantics.

schemas:
  book: ./schemas/book.json

rules:
  - paths: "notes/**/*.md"
    schema: book
`

	scaffoldSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "book",
  "type": "object",
  "required": ["title", "year"],
  "properties": {
    "title": { "type": "string", "minLength": 1 },
    "year":  { "type": "integer", "minimum": 0 },
    "tags":  { "type": "array", "items": { "type": "string" } }
  }
}
`

	// Keys are sorted alphabetically so the scaffold is already in
	// `katabridge fmt` canonical form — see TestInit_scaffoldIsCanonical.
	scaffoldExample = `---
tags:
  - example
title: Example
year: 2026
---
# Example

This file's frontmatter is validated by ` + "`schemas/book.json`" + `.
`
)

func newInitCmd() *cobra.Command {
	var dir string

	c := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a katabridge.yaml, an example schema, and an example document.",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := dir
			if target == "" {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				target = wd
			}

			files := []struct {
				rel     string
				content string
			}{
				{"katabridge.yaml", scaffoldConfig},
				{"schemas/book.json", scaffoldSchema},
				{"notes/example.md", scaffoldExample},
			}

			// Refuse to overwrite anything. Atomic-ish: we check all
			// destinations before writing any, so a partial scaffold
			// can't happen due to a single pre-existing file.
			for _, f := range files {
				p := filepath.Join(target, f.rel)
				if _, err := os.Stat(p); err == nil {
					return usageErr(fmt.Sprintf("%s already exists; refusing to overwrite", p))
				}
			}

			for _, f := range files {
				p := filepath.Join(target, f.rel)
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(p, []byte(f.content), 0o644); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", f.rel)
			}
			return nil
		},
	}

	c.Flags().StringVar(&dir, "dir", "", "Directory to scaffold into (default: current directory)")
	return c
}
