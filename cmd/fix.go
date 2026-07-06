package cmd

import (
	"bytes"
	"fmt"
	"os"

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
			res, err := resolveSelectors(projectFor(plan.Root), args)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				res = filterRootFixResolutionForDelegates(plan, res)
			}

			changed := false
			didChange, err := fixResolution(cmd, res, checkOnly)
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
		res, err := resolveSelectors(projectFor(delegate.Config), nil)
		if err != nil {
			return false, err
		}
		didChange, err := fixResolution(cmd, res, checkOnly)
		if err != nil {
			return false, err
		}
		if didChange {
			changed = true
		}
	}
	return changed, nil
}

func fixResolution(cmd *cobra.Command, res *project.Resolution, checkOnly bool) (bool, error) {
	changed := false
	for _, item := range res.Items {
		didChange, err := fixOne(item.Path, item.Collection, checkOnly)
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

// fixOne reports whether path's content would change. It computes the fixed
// content with the backend-agnostic fix engine and, unless check is set,
// persists it through the filesystem backend (an atomic replace). The split is
// deliberate: deciding what to write is fix's, writing it is the backend's.
func fixOne(path string, c project.Collection, check bool) (changed bool, err error) {
	if c.StorageType == string(storage.SQLite) {
		return false, fmt.Errorf("fix is not supported for sqlite collections yet")
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	result, err := fix.Apply(src, c)
	if err != nil {
		return false, err
	}
	if bytes.Equal(src, result) {
		return false, nil
	}
	if check {
		return true, nil
	}
	if err := filesystem.Write(path, result); err != nil {
		return false, err
	}
	return true, nil
}
