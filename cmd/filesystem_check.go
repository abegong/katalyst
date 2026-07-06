package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/storage/filesystemcheck"
)

type runtimeFileCheck struct {
	configured checks.ConfiguredCheck
	check      checks.Check
	needsDoc   bool
}

type runtimeFileSetCheck struct {
	configured checks.ConfiguredCheck
	check      checks.CollectionCheck
}

func runFilesystemChecks(errOut io.Writer, e *engine) (bool, error) {
	return runFilesystemChecksWithConfig(errOut, e, "")
}

func runRootFilesystemChecks(errOut io.Writer, e *engine, plan *project.Plan) (bool, error) {
	bad := false
	for _, scope := range e.proj.FilesystemCheckScopes() {
		scopeBad, err := runFilesystemScopeFiltered(errOut, e, scope, "", func(cc checks.ConfiguredCheck, path string) bool {
			return rootFilesystemCheckApplies(plan, path, cc)
		})
		if err != nil {
			return false, err
		}
		if scopeBad {
			bad = true
		}
	}
	return bad, nil
}

func runFilesystemChecksWithConfig(errOut io.Writer, e *engine, configPath string) (bool, error) {
	return runFilesystemChecksWithConfigAndFilter(errOut, e, configPath, nil)
}

func runFilesystemChecksWithConfigAndFilter(errOut io.Writer, e *engine, configPath string, include checkFilter) (bool, error) {
	bad := false
	for _, scope := range e.proj.FilesystemCheckScopes() {
		scope.Checks = filterConfiguredChecks(scope.Checks, include)
		if len(scope.Checks) == 0 {
			continue
		}
		scopeBad, err := runFilesystemScopeWithConfig(errOut, e, scope, configPath)
		if err != nil {
			return false, err
		}
		if scopeBad {
			bad = true
		}
	}
	return bad, nil
}

func rootFilesystemCheckApplies(plan *project.Plan, path string, cc checks.ConfiguredCheck) bool {
	for _, delegate := range plan.Delegates {
		if !delegate.Active || !project.DelegateContains(delegate.Delegate, plan.Root.Root, path) {
			continue
		}
		if rootAuthorityForConfiguredCheck(plan.Root.NestedConfigs, delegate.Delegate, project.AuthorityFilesystemChecks, cc) == project.AuthorityFileNearest {
			return false
		}
	}
	return true
}

func runFilesystemScope(errOut io.Writer, e *engine, scope filesystemcheck.Scope) (bool, error) {
	return runFilesystemScopeWithConfig(errOut, e, scope, "")
}

func runFilesystemScopeWithConfig(errOut io.Writer, e *engine, scope filesystemcheck.Scope, configPath string) (bool, error) {
	return runFilesystemScopeFiltered(errOut, e, scope, configPath, nil)
}

func runFilesystemScopeFiltered(errOut io.Writer, e *engine, scope filesystemcheck.Scope, configPath string, includeFile func(checks.ConfiguredCheck, string) bool) (bool, error) {
	expanded, err := filesystemcheck.Expand(scope)
	if err != nil {
		return false, asUsageErr(err)
	}
	fileChecks, err := runtimeFileChecks(scope.Checks)
	if err != nil {
		return false, err
	}
	setChecks, err := runtimeFileSetChecks(scope.Checks)
	if err != nil {
		return false, err
	}

	bad := false
	itemCtxs := make([]checks.ItemContext, 0, len(expanded.Selected))
	for _, file := range expanded.Selected {
		var doc *markdownbodytext.Document
		meta := map[string]any{}
		parseOK := true
		needsDoc := scopeNeedsDocumentForFile(scope.Checks, includeFile, file.Path)
		if needsDoc {
			src, err := os.ReadFile(file.Path)
			if err != nil {
				return false, asUsageErr(err)
			}
			doc, err = markdownbodytext.Parse(src)
			if err != nil {
				parseOK = false
				severity := checks.SeverityError
				if scope.ParseFailures == filesystemcheck.ParseFailuresWarning {
					severity = checks.SeverityWarning
				}
				printFilesystemViolation(errOut, scope, file.Rel, checks.Violation{
					Path:     "/",
					Message:  fmt.Sprintf("parse document: %v", err),
					Severity: severity,
				})
				if configPath != "" {
					fmt.Fprintf(errOut, "  config: %s\n", configPath)
				}
				if severity != checks.SeverityWarning {
					bad = true
				}
			} else {
				meta = dropKey(doc.Meta, "schema")
			}
		}
		itemCtxs = append(itemCtxs, checks.ItemContext{FilePath: file.Path, Meta: meta})
		ctx := checks.FileContext{
			FilePath:       file.Path,
			CollectionRoot: scope.Root,
			Doc:            doc,
			Meta:           meta,
		}
		for _, rc := range fileChecks {
			if includeFile != nil && !includeFile(rc.configured, file.Path) {
				continue
			}
			if rc.needsDoc && !parseOK {
				continue
			}
			for _, v := range rc.check.Run(ctx) {
				printFilesystemViolation(errOut, scope, file.Rel, v)
				if configPath != "" {
					fmt.Fprintf(errOut, "  config: %s\n", configPath)
				}
				if v.Severity != checks.SeverityWarning {
					bad = true
				}
			}
		}
	}
	for _, setCheck := range setChecks {
		setCtx := checks.FileSetContext{
			Root:    scope.Root,
			Include: scope.Include,
			Exclude: scope.Exclude,
		}
		for _, itemCtx := range itemCtxs {
			if includeFile != nil && !includeFile(setCheck.configured, itemCtx.FilePath) {
				continue
			}
			setCtx.Items = append(setCtx.Items, itemCtx)
		}
		for _, file := range expanded.Unmatched {
			if includeFile != nil && !includeFile(setCheck.configured, file.Path) {
				continue
			}
			setCtx.Unmatched = append(setCtx.Unmatched, file.Rel)
		}
		for _, v := range checks.RunFileSetAll(setCtx, []checks.CollectionCheck{setCheck.check}) {
			path := v.File
			if path == "" {
				path = scope.Name
			}
			printFilesystemViolation(errOut, scope, path, v)
			if configPath != "" {
				fmt.Fprintf(errOut, "  config: %s\n", configPath)
			}
			if v.Severity != checks.SeverityWarning {
				bad = true
			}
		}
	}
	return bad, nil
}

func runtimeFileChecks(configured []checks.ConfiguredCheck) ([]runtimeFileCheck, error) {
	if err := ensureLibrariesAvailable(configured); err != nil {
		return nil, err
	}
	var out []runtimeFileCheck
	for _, cc := range configured {
		if chk, ok := checks.Build(cc.Kind, cc.Args); ok {
			out = append(out, runtimeFileCheck{
				configured: cc,
				check:      chk,
				needsDoc:   checks.NeedsDocument(cc.Kind),
			})
		}
	}
	return out, nil
}

func runtimeFileSetChecks(configured []checks.ConfiguredCheck) ([]runtimeFileSetCheck, error) {
	if err := ensureLibrariesAvailable(configured); err != nil {
		return nil, err
	}
	var out []runtimeFileSetCheck
	for _, cc := range configured {
		if chk, ok := checks.BuildCollection(cc.Kind, cc.Args); ok {
			out = append(out, runtimeFileSetCheck{configured: cc, check: chk})
		}
	}
	return out, nil
}

func scopeNeedsDocumentForFile(configured []checks.ConfiguredCheck, includeFile func(checks.ConfiguredCheck, string) bool, path string) bool {
	for _, cc := range configured {
		if includeFile != nil && !includeFile(cc, path) {
			continue
		}
		if checks.NeedsDocument(cc.Kind) {
			return true
		}
	}
	return false
}

func printFilesystemViolation(w io.Writer, scope filesystemcheck.Scope, path string, v checks.Violation) {
	loc := v.Path
	if loc == "" {
		loc = "/"
	}
	marker := ""
	if v.Severity == checks.SeverityWarning {
		marker = "warning: "
	}
	if v.Line > 0 {
		fmt.Fprintf(w, "filesystem %s: %s:%d: %s%s: %s\n", scope.Name, path, v.Line, marker, loc, v.Message)
		return
	}
	fmt.Fprintf(w, "filesystem %s: %s: %s%s: %s\n", scope.Name, path, marker, loc, v.Message)
}
