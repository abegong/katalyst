package checks_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	_ "github.com/abegong/katalyst/internal/checks/all" // populate the registry
)

func TestKindConstantsAreRegistered(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "kinds.go", nil, 0)
	if err != nil {
		t.Fatalf("parse kinds.go: %v", err)
	}

	seen := 0
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", name.Name, err)
				}
				seen++
				if !checks.Known(checks.CheckType(value)) {
					t.Errorf("%s = %q is not registered", name.Name, value)
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no check kind constants found in kinds.go")
	}
}
