package cmd

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/checks/plaintext"
	"github.com/abegong/katalyst/internal/config"
	"github.com/abegong/katalyst/internal/frontmatter"
	"github.com/spf13/cobra"
)

func newFixCmd() *cobra.Command {
	var checkOnly bool

	c := &cobra.Command{
		Use:   "fix [selector ...]",
		Short: "Apply deterministic, safe fixes to the selected items.",
		Long: `fix rewrites each selected item's frontmatter in a canonical form:
top-level keys sorted alphabetically, yaml.v3 default block style, and
exactly one trailing newline. The body is preserved verbatim.

fix never invents semantic values: it will not inject placeholders for
missing required keys. See internal/frontmatter/README.md for why.

Selectors follow the same grammar as 'check'. With no selector, every
item in the project is considered.

With --check, no files are modified; instead, items that would change are
printed and the command exits with status 1. Use this in CI.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			res, err := resolveSelectors(projectFor(cfg), args)
			if err != nil {
				return err
			}

			changed := false
			for _, item := range res.Items {
				didChange, err := fixOne(item.Path, item.Collection, checkOnly)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", item.Path, err)
					return &exitError{code: exitValidationFail}
				}
				if didChange {
					changed = true
					fmt.Fprintln(cmd.OutOrStdout(), item.Path)
				}
			}
			if checkOnly && changed {
				return &exitError{code: exitValidationFail}
			}
			return nil
		},
	}

	c.Flags().BoolVar(&checkOnly, "check", false,
		"Don't write; exit 1 if any item would change (for CI).")
	return c
}

// fixOne returns whether path's content would change. It first applies any
// opted-in text_forbids body fixes for the item's collection, then formats the
// frontmatter. When check is false, the file is rewritten in place if the
// result differs.
func fixOne(path string, c config.Collection, check bool) (changed bool, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	fixed, err := applyTextFixes(src, c)
	if err != nil {
		return false, err
	}
	formatted, err := frontmatter.Format(fixed)
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
	tmp, err := os.CreateTemp(filepathDir(path), ".katalyst-fix-*")
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

// applyTextFixes rewrites the body with the collection's opted-in text_forbids
// fixes, then re-checks its own work: if a fix leaves the rule still violated
// (a bad template), it fails rather than writing a still-broken file. Files in
// collections with no text fixes are returned untouched.
func applyTextFixes(src []byte, c config.Collection) ([]byte, error) {
	fixers := textFixers(c)
	if len(fixers) == 0 {
		return src, nil
	}
	doc, err := frontmatter.Parse(src)
	if err != nil {
		return nil, err
	}
	body := doc.Body
	for _, f := range fixers {
		body = f.ApplyFix(body)
	}
	rechecked := &frontmatter.Document{Body: body, BodyLine: doc.BodyLine}
	for _, f := range fixers {
		if len(f.Run(checks.Context{Doc: rechecked})) > 0 {
			return nil, fmt.Errorf("fix did not resolve the violation for /%s/", f.Pattern)
		}
	}
	// Body is a verbatim tail of src, so everything before it is the prefix.
	prefix := src[:len(src)-len(doc.Body)]
	out := make([]byte, 0, len(prefix)+len(body))
	out = append(out, prefix...)
	out = append(out, body...)
	return out, nil
}

// textFixers builds the fixable text_forbids checks configured for a
// collection (those with a non-empty fix template).
func textFixers(c config.Collection) []plaintext.TextForbids {
	var out []plaintext.TextForbids
	for _, ch := range c.Checks {
		if ch.Type == config.CheckTextForbids && ch.Fix != "" {
			out = append(out, plaintext.TextForbids{
				Re:      regexp.MustCompile(ch.Pattern),
				Pattern: ch.Pattern,
				Target:  ch.Target,
				Select:  plaintext.CompileSelect(ch.Select),
				Fix:     ch.Fix,
			})
		}
	}
	return out
}

// filepathDir returns the directory of path, defaulting to "." when path
// has no separator. Used to keep atomic temp files on the same filesystem.
func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
