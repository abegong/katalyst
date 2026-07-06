package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/project"
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
	var planFlags projectPlanFlags

	c := &cobra.Command{
		Use:   "check [selector ...]",
		Short: "Run configured checks against the selected items",
		Long: `check parses each selected item's frontmatter (YAML, TOML, or JSON)
and runs the checks configured for its collection under .katalyst/bases/.

Selectors (see docs/content/deep-dives/domain-model/_index.md):

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
			plan, err := project.BuildPlan(project.PlanOptions{
				ConfigPath:          planFlags.configPath,
				ProjectDir:          planFlags.projectDir,
				DisableNestedConfig: planFlags.disableNestedConfig,
			})
			if err != nil {
				return asUsageErr(err)
			}
			e, err := newEngineForConfig(plan.Root, schemaFlag, "")
			if err != nil {
				return err
			}
			anyInvalid := false
			out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()

			if len(args) == 0 {
				bad, err := runRootFilesystemChecks(errOut, e, plan)
				if err != nil {
					return err
				}
				if bad {
					anyInvalid = true
				}
				bad, err = runDelegatedChecks(out, errOut, plan, schemaFlag)
				if err != nil {
					return err
				}
				if bad {
					anyInvalid = true
				}
			}
			res, err := resolveSelectors(e.proj, args)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				res = filterRootResolutionForDelegates(plan, res)
			}

			for _, item := range res.Items {
				include := rootCollectionCheckFilterForPath(plan, item.Path)
				if include != nil && !collectionHasApplicableConfiguredChecks(item.Collection, include) {
					continue
				}
				ok, err := checkItemWithConfigAndFilter(out, errOut, e, item, "", include)
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
					if rootUnmatchedDelegated(plan, c, rel) {
						continue
					}
					fmt.Fprintf(errOut, "%s/%s: unmatched file (does not match pattern %q)\n", c.Path, rel, c.Pattern)
					anyInvalid = true
				}
			}

			// Collection-scoped checks run once per collection over its FULL
			// item set, independent of how the selector narrowed the per-item
			// pass (a uniqueness verdict is only correct against every item).
			bad, err := runRootCollectionChecks(errOut, e, selectedCollections(res), plan)
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
	addProjectPlanFlags(c, &planFlags)
	return c
}

func runDelegatedChecks(out, errOut io.Writer, plan *project.Plan, schemaFlag string) (bool, error) {
	bad := false
	for _, delegate := range plan.Delegates {
		if !delegate.Active || delegate.Config == nil {
			continue
		}
		e, err := newEngineForConfig(delegate.Config, schemaFlag, "")
		if err != nil {
			return false, err
		}
		includeCollectionCheck := delegatedCheckFilter(plan.Root.NestedConfigs, delegate.Delegate, project.AuthorityCollectionChecks)
		includeFilesystemCheck := delegatedCheckFilter(plan.Root.NestedConfigs, delegate.Delegate, project.AuthorityFilesystemChecks)
		configPath := filepath.ToSlash(filepath.Join(delegate.Delegate.Path, delegate.Delegate.Config))
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityFilesystemChecks) != project.AuthorityRootNearest {
			scopeBad, err := runFilesystemChecksWithConfigAndFilter(errOut, e, configPath, includeFilesystemCheck)
			if err != nil {
				return false, err
			}
			if scopeBad {
				bad = true
			}
		}
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityCollections) == project.AuthorityRootNearest &&
			plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityCollectionChecks) == project.AuthorityRootNearest &&
			plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthoritySchemas) == project.AuthorityRootNearest {
			continue
		}
		res, err := resolveSelectors(e.proj, nil)
		if err != nil {
			return false, err
		}
		for _, item := range res.Items {
			ok, err := checkItemWithConfigAndFilter(out, errOut, e, item, configPath, includeCollectionCheck)
			if err != nil {
				fmt.Fprintf(errOut, "%s: %v\n", item.Path, err)
				bad = true
				continue
			}
			if !ok {
				bad = true
			}
		}
		for _, c := range res.Scan {
			unmatched, err := e.proj.Unmatched(c)
			if err != nil {
				return false, asUsageErr(err)
			}
			for _, rel := range unmatched {
				fmt.Fprintf(errOut, "%s/%s: unmatched file (does not match pattern %q)\n", c.Path, rel, c.Pattern)
				fmt.Fprintf(errOut, "  config: %s\n", configPath)
				bad = true
			}
		}
		collBad, err := runCollectionChecksWithConfigAndFilter(errOut, e, selectedCollections(res), configPath, includeCollectionCheck)
		if err != nil {
			return false, err
		}
		if collBad {
			bad = true
		}
	}
	return bad, nil
}

func filterRootResolutionForDelegates(plan *project.Plan, res *project.Resolution) *project.Resolution {
	if len(plan.Delegates) == 0 {
		return res
	}
	filtered := &project.Resolution{
		Items: make([]project.Item, 0, len(res.Items)),
		Scan:  res.Scan,
	}
	for _, item := range res.Items {
		if rootItemDelegated(plan, item.Path) {
			continue
		}
		filtered.Items = append(filtered.Items, item)
	}
	return filtered
}

func rootItemDelegated(plan *project.Plan, path string) bool {
	for _, delegate := range plan.Delegates {
		if !delegate.Active {
			continue
		}
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityCollections) == project.AuthorityRootNearest &&
			plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthoritySchemas) == project.AuthorityRootNearest {
			continue
		}
		if project.DelegateContains(delegate.Delegate, plan.Root.Root, path) {
			return true
		}
	}
	return false
}

func rootCollectionCheckFilterForPath(plan *project.Plan, path string) checkFilter {
	if !pathInActiveDelegate(plan, path) {
		return nil
	}
	return func(cc checks.ConfiguredCheck) bool {
		return rootCollectionCheckApplies(plan, path, cc)
	}
}

func pathInActiveDelegate(plan *project.Plan, path string) bool {
	for _, delegate := range plan.Delegates {
		if delegate.Active && project.DelegateContains(delegate.Delegate, plan.Root.Root, path) {
			return true
		}
	}
	return false
}

func rootCollectionCheckApplies(plan *project.Plan, path string, cc checks.ConfiguredCheck) bool {
	for _, delegate := range plan.Delegates {
		if !delegate.Active || !project.DelegateContains(delegate.Delegate, plan.Root.Root, path) {
			continue
		}
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityCollections) != project.AuthorityRootNearest {
			return false
		}
		if rootAuthorityForConfiguredCheck(plan.Root.NestedConfigs, delegate.Delegate, project.AuthorityCollectionChecks, cc) == project.AuthorityFileNearest {
			return false
		}
	}
	return true
}

func collectionHasApplicableConfiguredChecks(c project.Collection, include checkFilter) bool {
	for _, cc := range c.Checks {
		if include(cc) {
			return true
		}
	}
	for _, variant := range c.Variants {
		for _, cc := range variant.Checks {
			if include(cc) {
				return true
			}
		}
	}
	return false
}

func rootUnmatchedDelegated(plan *project.Plan, c project.Collection, rel string) bool {
	for _, delegate := range plan.Delegates {
		if !delegate.Active {
			continue
		}
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityCollections) == project.AuthorityRootNearest {
			continue
		}
		path := filepath.Join(c.Dir, filepath.FromSlash(rel))
		if project.DelegateContains(delegate.Delegate, plan.Root.Root, path) {
			return true
		}
	}
	return false
}

func rootAuthorityForConfiguredCheck(settings project.NestedConfigSettings, delegate project.NestedDelegate, subsystem project.AuthoritySubsystem, cc checks.ConfiguredCheck) project.AuthorityPolicy {
	family := ""
	if desc, ok := checks.DescriptorFor(cc.Kind); ok {
		family = desc.Family
	}
	return settings.AuthorityForCheck(delegate, subsystem, string(cc.Kind), family)
}

func delegatedCheckFilter(settings project.NestedConfigSettings, delegate project.NestedDelegate, subsystem project.AuthoritySubsystem) checkFilter {
	return func(cc checks.ConfiguredCheck) bool {
		return rootAuthorityForConfiguredCheck(settings, delegate, subsystem, cc) != project.AuthorityRootNearest
	}
}

// checkItem reads one item, resolves its checks, runs them, and writes
// results. Returns (true, nil) if valid, (false, nil) on validation
// errors, or (_, err) if the file couldn't be read/parsed.
func checkItem(out, errOut io.Writer, e *engine, item project.Item) (bool, error) {
	return checkItemWithConfig(out, errOut, e, item, "")
}

func checkItemWithConfig(out, errOut io.Writer, e *engine, item project.Item, configPath string) (bool, error) {
	return checkItemWithConfigAndFilter(out, errOut, e, item, configPath, nil)
}

func checkItemWithConfigAndFilter(out, errOut io.Writer, e *engine, item project.Item, configPath string, include checkFilter) (bool, error) {
	content, err := e.proj.ReadItem(item)
	if err != nil {
		return false, err
	}
	doc := content.Doc

	// A frontmatter-less file is not rejected outright: the configured checks
	// run against it (text/filesystem rules lint the body and path; object
	// checks surface their own "missing field" violations against the nil
	// metadata).
	checkList, err := e.checksForFiltered(item.Collection, doc.Meta, include)
	if err != nil {
		return false, err
	}
	if include != nil && len(checkList) == 0 {
		return true, nil
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
		if configPath != "" {
			fmt.Fprintf(errOut, "  config: %s\n", configPath)
		}
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
func itemStatus(e *engine, c project.Collection, item project.Item) (int, error) {
	content, err := e.proj.ReadItem(item)
	if err != nil {
		return 0, err
	}
	doc := content.Doc
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
func selectedCollections(res *project.Resolution) []project.Collection {
	byName := map[string]project.Collection{}
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
	out := make([]project.Collection, 0, len(names))
	for _, n := range names {
		out = append(out, byName[n])
	}
	return out
}

// runCollectionChecks runs each collection's collection-scoped checks over
// its full item set. Returns whether any violation was reported.
func runCollectionChecks(errOut io.Writer, e *engine, collections []project.Collection) (bool, error) {
	return runCollectionChecksWithConfig(errOut, e, collections, "")
}

func runCollectionChecksWithConfig(errOut io.Writer, e *engine, collections []project.Collection, configPath string) (bool, error) {
	return runCollectionChecksWithConfigAndFilter(errOut, e, collections, configPath, nil)
}

func runCollectionChecksWithConfigAndFilter(errOut io.Writer, e *engine, collections []project.Collection, configPath string, include checkFilter) (bool, error) {
	return runCollectionChecksFiltered(errOut, e, collections, configPath, include, nil)
}

func runRootCollectionChecks(errOut io.Writer, e *engine, collections []project.Collection, plan *project.Plan) (bool, error) {
	return runCollectionChecksFiltered(errOut, e, collections, "", nil, func(cc checks.ConfiguredCheck, path string) bool {
		return rootCollectionCheckApplies(plan, path, cc)
	})
}

func runCollectionChecksFiltered(errOut io.Writer, e *engine, collections []project.Collection, configPath string, include checkFilter, includeItem func(checks.ConfiguredCheck, string) bool) (bool, error) {
	bad := false
	for _, c := range collections {
		configured := filterConfiguredChecks(c.Checks, include)
		if err := ensureLibrariesAvailable(configured); err != nil {
			return false, err
		}
		collChecks := make([]struct {
			configured checks.ConfiguredCheck
			check      checks.CollectionCheck
		}, 0, len(configured))
		for _, cc := range configured {
			if col, ok := checks.BuildCollection(cc.Kind, cc.Args); ok {
				collChecks = append(collChecks, struct {
					configured checks.ConfiguredCheck
					check      checks.CollectionCheck
				}{configured: cc, check: col})
			}
		}
		if len(collChecks) == 0 {
			continue
		}
		items, err := e.proj.Items(c)
		if err != nil {
			return false, asUsageErr(err)
		}
		itemCtxs := make([]checks.ItemContext, 0, len(items))
		for _, it := range items {
			if includeItem != nil && !anyCollectionCheckApplies(collChecks, includeItem, it.Path) {
				continue
			}
			content, err := e.proj.ReadItem(it)
			if err != nil {
				fmt.Fprintf(errOut, "%s: %v\n", it.Path, err)
				bad = true
				continue
			}
			doc := content.Doc
			itemCtxs = append(itemCtxs, checks.ItemContext{
				FilePath: it.Path,
				Meta:     dropKey(doc.Meta, "schema"),
			})
		}
		for _, collCheck := range collChecks {
			ctx := checks.CollectionContext{Root: c.Dir, Items: make([]checks.ItemContext, 0, len(itemCtxs))}
			for _, itemCtx := range itemCtxs {
				if includeItem != nil && !includeItem(collCheck.configured, itemCtx.FilePath) {
					continue
				}
				ctx.Items = append(ctx.Items, itemCtx)
			}
			for _, v := range checks.RunCollectionAll(ctx, []checks.CollectionCheck{collCheck.check}) {
				marker := ""
				if v.Severity == checks.SeverityWarning {
					marker = "warning: "
				}
				fmt.Fprintf(errOut, "%s: %s%s\n", v.File, marker, v.Message)
				if configPath != "" {
					fmt.Fprintf(errOut, "  config: %s\n", configPath)
				}
				if v.Severity != checks.SeverityWarning {
					bad = true
				}
			}
		}
	}
	return bad, nil
}

func anyCollectionCheckApplies(collChecks []struct {
	configured checks.ConfiguredCheck
	check      checks.CollectionCheck
}, includeItem func(checks.ConfiguredCheck, string) bool, path string) bool {
	for _, collCheck := range collChecks {
		if includeItem(collCheck.configured, path) {
			return true
		}
	}
	return false
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
