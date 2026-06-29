package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/abegong/katalyst/internal/codec/markdownbodytext"
	"github.com/abegong/katalyst/internal/storage/filesystemcheck"
)

type runtimeFileCheck struct {
	kind     checks.CheckType
	check    checks.Check
	needsDoc bool
}

func runFilesystemChecks(errOut io.Writer, e *engine) (bool, error) {
	bad := false
	for _, scope := range e.proj.FilesystemCheckScopes() {
		scopeBad, err := runFilesystemScope(errOut, e, scope)
		if err != nil {
			return false, err
		}
		if scopeBad {
			bad = true
		}
	}
	return bad, nil
}

func runFilesystemScope(errOut io.Writer, e *engine, scope filesystemcheck.Scope) (bool, error) {
	expanded, err := filesystemcheck.Expand(scope)
	if err != nil {
		return false, asUsageErr(err)
	}
	fileChecks, err := runtimeFileChecks(scope.Checks)
	if err != nil {
		return false, err
	}
	setChecks, err := e.fileSetChecksFor(scope.Checks)
	if err != nil {
		return false, err
	}

	needsDoc := scopeNeedsDocument(scope.Checks)
	bad := false
	setCtx := checks.FileSetContext{
		Root:      scope.Root,
		Items:     make([]checks.ItemContext, 0, len(expanded.Selected)),
		Unmatched: rels(expanded.Unmatched),
		Include:   scope.Include,
		Exclude:   scope.Exclude,
	}
	for _, file := range expanded.Selected {
		var doc *markdownbodytext.Document
		meta := map[string]any{}
		parseOK := true
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
				if severity != checks.SeverityWarning {
					bad = true
				}
			} else {
				meta = dropKey(doc.Meta, "schema")
			}
		}
		setCtx.Items = append(setCtx.Items, checks.ItemContext{FilePath: file.Path, Meta: meta})
		ctx := checks.FileContext{
			FilePath:       file.Path,
			CollectionRoot: scope.Root,
			Doc:            doc,
			Meta:           meta,
		}
		for _, rc := range fileChecks {
			if rc.needsDoc && !parseOK {
				continue
			}
			for _, v := range rc.check.Run(ctx) {
				printFilesystemViolation(errOut, scope, file.Rel, v)
				if v.Severity != checks.SeverityWarning {
					bad = true
				}
			}
		}
	}
	for _, v := range checks.RunFileSetAll(setCtx, setChecks) {
		path := v.File
		if path == "" {
			path = scope.Name
		}
		printFilesystemViolation(errOut, scope, path, v)
		if v.Severity != checks.SeverityWarning {
			bad = true
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
				kind:     cc.Kind,
				check:    chk,
				needsDoc: checks.NeedsDocument(cc.Kind),
			})
		}
	}
	return out, nil
}

func scopeNeedsDocument(configured []checks.ConfiguredCheck) bool {
	for _, cc := range configured {
		if checks.NeedsDocument(cc.Kind) {
			return true
		}
	}
	return false
}

func rels(files []filesystemcheck.File) []string {
	out := make([]string, len(files))
	for i, file := range files {
		out[i] = file.Rel
	}
	return out
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
