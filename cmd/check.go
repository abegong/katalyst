package cmd

import (
	"fmt"
	"io"

	"github.com/katabase-ai/katalyst/internal/checks"
	"github.com/katabase-ai/katalyst/internal/config"
	"github.com/katabase-ai/katalyst/internal/project"
	"github.com/spf13/cobra"
)

// Exit codes. Loosely modeled on shellcheck and on the `jv` CLI from
// santhosh-tekuri/jsonschema.
const (
	exitOK             = 0
	exitValidationFail = 1
	exitUsage          = 2
)

func newCheckCmd() *cobra.Command {
	var schemaFlag string

	c := &cobra.Command{
		Use:   "check [selector ...]",
		Short: "Run configured checks against the selected items.",
		Long: `check parses YAML frontmatter from each selected item and runs the
checks configured for its collection under .katalyst/collections/.

Selectors (see docs/reference/glossary.md):

  (none)                the whole project (every collection)
  <collection>          one collection (all its items)
  <collection>/<item>   one item

Object-schema resolution, highest precedence first:

  1. --schema <path>      (applies to every selected item)
  2. inline "schema:" key in the item's frontmatter (a name from config)
  3. the collection's configured object checks

Files inside a collection directory that do not match its pattern are
reported as unmatched references (errors).`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := newEngine(schemaFlag)
			if err != nil {
				return err
			}
			res, err := resolveSelectors(e.proj, args)
			if err != nil {
				return err
			}

			anyInvalid := false
			out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()

			for _, item := range res.Items {
				ok, err := checkItem(out, errOut, e, item)
				if err != nil {
					fmt.Fprintf(errOut, "%s: %v\n", item.Path, err)
					anyInvalid = true
					continue
				}
				if !ok {
					anyInvalid = true
				}
			}

			// Unmatched references in wholesale-selected collections.
			for _, c := range res.Scan {
				unmatched, err := e.proj.Unmatched(c)
				if err != nil {
					return asUsageErr(err)
				}
				for _, rel := range unmatched {
					fmt.Fprintf(errOut, "%s/%s: unmatched file (does not match pattern %q)\n", c.Path, rel, c.Pattern)
					anyInvalid = true
				}
			}

			if anyInvalid {
				return &exitError{code: exitValidationFail}
			}
			return nil
		},
	}

	c.Flags().StringVarP(&schemaFlag, "schema", "s", "",
		"Path to a JSON Schema file. Overrides config-based resolution for every selected item.")
	return c
}

// checkItem reads one item, resolves its checks, runs them, and writes
// results. Returns (true, nil) if valid, (false, nil) on validation
// errors, or (_, err) if the file couldn't be read/parsed.
func checkItem(out, errOut io.Writer, e *engine, item project.Item) (bool, error) {
	doc, err := parseItem(item.Path)
	if err != nil {
		return false, err
	}
	if !doc.HasFrontmatter {
		fmt.Fprintf(errOut, "%s: no frontmatter found\n", item.Path)
		return false, nil
	}

	checkList, err := e.checksFor(item.Collection, doc.Meta)
	if err != nil {
		return false, err
	}

	// The "schema" key is a katalyst directive, not user data. Strip it
	// before validating so schemas with additionalProperties:false don't
	// reject documents that opt into themselves.
	instance := dropKey(doc.Meta, "schema")

	result := checks.RunAll(checks.Context{
		FilePath: item.Path,
		Doc:      doc,
		Meta:     instance,
	}, checkList)
	if len(result) == 0 {
		fmt.Fprintf(out, "%s: OK\n", item.Path)
		return true, nil
	}

	for _, v := range result {
		loc := v.Path
		if loc == "" {
			loc = "/"
		}
		if v.Line > 0 {
			fmt.Fprintf(errOut, "%s:%d: %s: %s\n", item.Path, v.Line, loc, v.Message)
		} else {
			fmt.Fprintf(errOut, "%s: %s: %s\n", item.Path, loc, v.Message)
		}
	}
	return false, nil
}

// itemStatus runs an item's checks and returns the number of violations
// (or an error if the file couldn't be read/parsed). Used by `item list`.
func itemStatus(e *engine, c config.Collection, item project.Item) (int, error) {
	doc, err := parseItem(item.Path)
	if err != nil {
		return 0, err
	}
	if !doc.HasFrontmatter {
		return 1, nil
	}
	checkList, err := e.checksFor(c, doc.Meta)
	if err != nil {
		return 0, err
	}
	instance := dropKey(doc.Meta, "schema")
	result := checks.RunAll(checks.Context{FilePath: item.Path, Doc: doc, Meta: instance}, checkList)
	return len(result), nil
}

// usageErr wraps a message so main exits with code 2 (usage error).
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

// dropKey returns a shallow copy of m without the named key.
func dropKey(m map[string]any, key string) map[string]any {
	if _, present := m[key]; !present {
		return m
	}
	out := make(map[string]any, len(m)-1)
	for k, v := range m {
		if k == key {
			continue
		}
		out[k] = v
	}
	return out
}
