package cmd

import (
	"bytes"
	"fmt"

	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
	"github.com/abegong/katalyst/internal/fix"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection/filesystem"
	"github.com/spf13/cobra"
)

func newFixCmd() *cobra.Command {
	var checkOnly bool
	var planFlags projectPlanFlags

	c := &cobra.Command{
		Use:   "fix [selector ...]",
		Short: "Apply deterministic, safe fixes to the selected items",
		Long: `fix rewrites each selected item's frontmatter in a canonical form:
top-level keys sorted alphabetically, yaml.v3 default block style, and
exactly one trailing newline. The body is preserved verbatim.

fix works on an item's text form, so its collection must expose a text body.
Every filesystem collection does; a sqlite collection does when it maps a
content column. A collection of attributes alone has nothing for fix to rewrite.

fix never invents semantic values: it will not inject placeholders for
missing required keys. See docs/content/deep-dives/domain-model/fix.md for why.

Selectors follow the same grammar as 'check'. With no selector, every
item in the project is considered.

With --check, no files are modified; instead, items that would change are
printed and the command exits with status 1. Use this in CI.`,
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
			if err := validateFixAuthority(plan); err != nil {
				return asUsageErr(err)
			}
			root := projectFor(plan.Root)
			res, err := resolveSelectors(root, args)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				res = filterRootFixResolutionForDelegates(plan, res)
			}

			changed := false
			didChange, err := fixResolution(cmd, root, res, checkOnly)
			if err != nil {
				return err
			}
			if didChange {
				changed = true
			}
			if len(args) == 0 {
				didChange, err = fixDelegates(cmd, plan, checkOnly)
				if err != nil {
					return err
				}
				if didChange {
					changed = true
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
	addProjectPlanFlags(c, &planFlags)
	return c
}

func validateFixAuthority(plan *project.Plan) error {
	for _, delegate := range plan.Delegates {
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityFix) == project.AuthorityCompose {
			return fmt.Errorf("nested config %s: fix authority %q is not supported", delegate.Delegate.Path, project.AuthorityCompose)
		}
	}
	return nil
}

func filterRootFixResolutionForDelegates(plan *project.Plan, res *project.Resolution) *project.Resolution {
	if len(plan.Delegates) == 0 {
		return res
	}
	filtered := &project.Resolution{
		Items: make([]project.Item, 0, len(res.Items)),
		Scan:  res.Scan,
	}
	for _, item := range res.Items {
		if rootFixDelegated(plan, item.Path) {
			continue
		}
		filtered.Items = append(filtered.Items, item)
	}
	return filtered
}

func rootFixDelegated(plan *project.Plan, path string) bool {
	for _, delegate := range plan.Delegates {
		if !delegate.Active {
			continue
		}
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityFix) != project.AuthorityFileNearest {
			continue
		}
		if project.DelegateContains(delegate.Delegate, plan.Root.Root, path) {
			return true
		}
	}
	return false
}

func fixDelegates(cmd *cobra.Command, plan *project.Plan, checkOnly bool) (bool, error) {
	changed := false
	for _, delegate := range plan.Delegates {
		if !delegate.Active || delegate.Config == nil {
			continue
		}
		if plan.Root.NestedConfigs.AuthorityFor(delegate.Delegate, project.AuthorityFix) != project.AuthorityFileNearest {
			continue
		}
		child := projectFor(delegate.Config)
		res, err := resolveSelectors(child, nil)
		if err != nil {
			return false, err
		}
		didChange, err := fixResolution(cmd, child, res, checkOnly)
		if err != nil {
			return false, err
		}
		if didChange {
			changed = true
		}
	}
	return changed, nil
}

func fixResolution(cmd *cobra.Command, p *project.Project, res *project.Resolution, checkOnly bool) (bool, error) {
	changed := false
	for _, item := range res.Items {
		didChange, err := fixOne(p, item, checkOnly)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s: %v\n", item.Path, err)
			return false, &exitError{code: exitValidationFail}
		}
		if didChange {
			changed = true
			fmt.Fprintln(cmd.OutOrStdout(), item.Path)
		}
	}
	return changed, nil
}

// fixOne reports whether item's content would change. It reads through the
// project (so each backend decodes its own storage), computes the fixed content
// with the backend-agnostic fix engine, and unless check is set persists it. The
// split is deliberate: deciding what to write is fix's, writing it is the
// backend's.
//
// The gate is a capability, not a backend name: fix operates on an item's text
// form, so a collection that maps no text body has nothing for it to rewrite.
func fixOne(p *project.Project, item project.Item, check bool) (changed bool, err error) {
	c := item.Collection
	if !c.HasTextContent() {
		return false, fmt.Errorf("fix requires a text content mapping; collection %q maps only attributes", c.Name)
	}
	content, err := p.ReadItem(item)
	if err != nil {
		return false, err
	}
	result, err := fix.Apply(content.Raw, c)
	if err != nil {
		return false, err
	}
	if bytes.Equal(content.Raw, result) {
		return false, nil
	}
	if check {
		return true, nil
	}
	if err := persistFix(p, item, result); err != nil {
		return false, err
	}
	return true, nil
}

// persistFix writes the fixed bytes back through the item's backend.
//
// A filesystem item is stored as the document, so the bytes go down verbatim in
// an atomic replace. A SQLite item is stored as columns, and only its body can
// have changed: the frontmatter fix.Apply canonicalized was synthesized from the
// row's attributes moments earlier by the SQLite reader, so it is canonical
// already. Passing nil metadata keeps the UPDATE on the content column and
// leaves the attribute columns untouched.
func persistFix(p *project.Project, item project.Item, result []byte) error {
	if storage.BaseType(item.Collection.StorageType) != storage.SQLite {
		return filesystem.Write(item.Path, result)
	}
	doc, err := markdownbodytext.Parse(result)
	if err != nil {
		return err
	}
	return p.UpdateItem(item.Collection, item.ID, nil, doc.Body)
}
