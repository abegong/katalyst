package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/katabase-ai/katabridge/internal/frontmatter"
	"github.com/spf13/cobra"
)

func newFmtCmd() *cobra.Command {
	var check bool

	c := &cobra.Command{
		Use:   "fmt [paths...]",
		Short: "Normalize markdown frontmatter (sorts keys, fixes trailing newline).",
		Long: `Format rewrites each file's frontmatter in a canonical form:
top-level keys sorted alphabetically, yaml.v3 default block style, and
exactly one trailing newline on the file. The body is preserved verbatim.

See product/decisions.md (D4) for the rationale.

With --check, no files are modified; instead, paths that would change
are printed and the command exits with status 1. Use this in CI.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			changed := false
			for _, path := range args {
				didChange, err := formatOne(path, check)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", path, err)
					return &exitError{code: exitValidationFail}
				}
				if didChange {
					changed = true
					fmt.Fprintln(cmd.OutOrStdout(), path)
				}
			}
			if check && changed {
				return &exitError{code: exitValidationFail}
			}
			return nil
		},
	}

	c.Flags().BoolVar(&check, "check", false,
		"Don't write; exit 1 if any file would change (for CI).")
	return c
}

// formatOne returns whether path's content would change. When check is
// false, the file is rewritten in place if its formatted form differs.
func formatOne(path string, check bool) (changed bool, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	formatted, err := frontmatter.Format(src)
	if err != nil {
		return false, err
	}
	if bytes.Equal(src, formatted) {
		return false, nil
	}
	if check {
		return true, nil
	}
	// Write atomically: write to a sibling temp file, then rename.
	// This avoids leaving a half-written file behind on crash.
	tmp, err := os.CreateTemp(filepathDir(path), ".katabridge-fmt-*")
	if err != nil {
		return false, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(formatted); err != nil {
		tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return false, err
	}
	return true, nil
}

// filepathDir is a tiny helper that returns the directory of path,
// defaulting to "." when path has no separator. It exists because
// os.CreateTemp("", ...) puts the file in $TMPDIR which would prevent
// an atomic rename across filesystems.
func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
