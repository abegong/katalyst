package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/project/config"
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
		Long: `check parses each selected item's frontmatter (YAML, TOML, or JSON)
and runs the checks configured for its collection under .katalyst/storage/.

Selectors (see docs/content/deep-dives/domain-model.md):

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

			// Collection-scoped checks run once per collection over its FULL
			// item set, independent of how the selector narrowed the per-item
			// pass (a uniqueness verdict is only correct against every item).
			bad, err := runCollectionChecks(errOut, e, selectedCollections(res))
			if err != nil {
				return err
			}
			if bad {
				anyInvalid = true
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

	// A frontmatter-less file is not rejected outright: the configured checks
	// run against it (text/filesystem rules lint the body and path; object
	// checks surface their own "missing field" violations against the nil
	// metadata).
	checkList, err := e.checksFor(item.Collection, doc.Meta)
	if err != nil {
		return false, err
	}

	// The "schema" key is a katalyst directive, not user data. Strip it
	// before validating so schemas with additionalProperties:false don't
	// reject documents that opt into themselves.
	instance := dropKey(doc.Meta, "schema")

	result := checks.RunAll(checks.Context{
		FilePath:       item.Path,
		CollectionRoot: item.Collection.Dir,
		Doc:            doc,
		Meta:           instance,
	}, checkList)

	errCount := 0
	for _, v := range result {
		printViolation(errOut, item.Path, v)
		if v.Severity != checks.SeverityWarning {
			errCount++
		}
	}
	// Warnings are advisory: an item with only warnings still passes.
	if errCount == 0 {
		fmt.Fprintf(out, "%s: OK\n", item.Path)
		return true, nil
	}
	return false, nil
}

// printViolation writes one violation. Errors keep the original
// `path[:line]: /loc: message` form; warnings carry a "warning:" marker so
// they read as advisory and are easy to filter.
func printViolation(w io.Writer, path string, v checks.Violation) {
	loc := v.Path
	if loc == "" {
		loc = "/"
	}
	marker := ""
	if v.Severity == checks.SeverityWarning {
		marker = "warning: "
	}
	if v.Line > 0 {
		fmt.Fprintf(w, "%s:%d: %s%s: %s\n", path, v.Line, marker, loc, v.Message)
	} else {
		fmt.Fprintf(w, "%s: %s%s: %s\n", path, marker, loc, v.Message)
	}
}

// itemStatus runs an item's checks and returns the number of error-severity
// violations (or an error if the file couldn't be read/parsed). Warnings are
// advisory and do not count toward an item's failing status. Used by
// `item list`.
func itemStatus(e *engine, c config.Collection, item project.Item) (int, error) {
	doc, err := parseItem(item.Path)
	if err != nil {
		return 0, err
	}
	checkList, err := e.checksFor(c, doc.Meta)
	if err != nil {
		return 0, err
	}
	instance := dropKey(doc.Meta, "schema")
	result := checks.RunAll(checks.Context{FilePath: item.Path, CollectionRoot: c.Dir, Doc: doc, Meta: instance}, checkList)
	errCount := 0
	for _, v := range result {
		if v.Severity != checks.SeverityWarning {
			errCount++
		}
	}
	return errCount, nil
}

// selectedCollections returns the distinct collections touched by a
// resolution: those selected wholesale and those owning a selected item,
// in name order, so collection-scoped checks run once each, deterministically.
func selectedCollections(res *project.Resolution) []config.Collection {
	byName := map[string]config.Collection{}
	for _, c := range res.Scan {
		byName[c.Name] = c
	}
	for _, it := range res.Items {
		byName[it.Collection.Name] = it.Collection
	}
	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]config.Collection, 0, len(names))
	for _, n := range names {
		out = append(out, byName[n])
	}
	return out
}

// runCollectionChecks runs each collection's collection-scoped checks over
// its full item set. Returns whether any violation was reported.
func runCollectionChecks(errOut io.Writer, e *engine, collections []config.Collection) (bool, error) {
	bad := false
	for _, c := range collections {
		collChecks, err := e.collectionChecksFor(c)
		if err != nil {
			return false, err
		}
		if len(collChecks) == 0 {
			continue
		}
		items, err := e.proj.Items(c)
		if err != nil {
			return false, asUsageErr(err)
		}
		ctx := checks.CollectionContext{Root: c.Dir, Items: make([]checks.ItemContext, 0, len(items))}
		for _, it := range items {
			doc, err := parseItem(it.Path)
			if err != nil {
				fmt.Fprintf(errOut, "%s: %v\n", it.Path, err)
				bad = true
				continue
			}
			ctx.Items = append(ctx.Items, checks.ItemContext{
				FilePath: it.Path,
				Meta:     dropKey(doc.Meta, "schema"),
			})
		}
		for _, v := range checks.RunCollectionAll(ctx, collChecks) {
			marker := ""
			if v.Severity == checks.SeverityWarning {
				marker = "warning: "
			}
			fmt.Fprintf(errOut, "%s: %s%s\n", v.File, marker, v.Message)
			if v.Severity != checks.SeverityWarning {
				bad = true
			}
		}
	}
	return bad, nil
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
