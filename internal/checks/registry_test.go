package checks

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
)

// TestDescriptorParity is the no-orphan guarantee: every check type
// dispatched in config.normalizeCheck's switch must have a Descriptor, and
// every Descriptor must correspond to a dispatched check type. A new check type
// added to the switch without a registry entry (or vice versa) fails here, so a
// check type cannot ship undocumented.
func TestDescriptorParity(t *testing.T) {
	dispatched := dispatchedKinds(t)

	registered := map[string]bool{}
	for _, d := range Descriptors() {
		k := string(d.CheckType)
		if registered[k] {
			t.Errorf("duplicate descriptor for kind %q", k)
		}
		registered[k] = true
	}

	for k := range dispatched {
		if !registered[k] {
			t.Errorf("kind %q is dispatched in config.normalizeCheck but has no Descriptor in registry.go", k)
		}
	}
	for k := range registered {
		if !dispatched[k] {
			t.Errorf("kind %q has a Descriptor but is not dispatched in config.normalizeCheck", k)
		}
	}
}

// TestDescriptorMetadata checks each descriptor is internally well-formed so
// the generator has everything it needs.
func TestDescriptorMetadata(t *testing.T) {
	families := map[string]bool{}
	for _, f := range Families() {
		families[f.ID] = true
	}
	seenSlug := map[string]bool{}
	for _, d := range Descriptors() {
		if d.Family == "" || !families[d.Family] {
			t.Errorf("kind %q has unknown family %q", d.CheckType, d.Family)
		}
		if d.Slug == "" {
			t.Errorf("kind %q has empty slug", d.CheckType)
		}
		key := d.Family + "/" + d.Slug
		if seenSlug[key] {
			t.Errorf("duplicate page path %q", key)
		}
		seenSlug[key] = true
		if d.Title == "" {
			t.Errorf("kind %q has empty title", d.CheckType)
		}
		if d.Summary == "" {
			t.Errorf("kind %q has empty summary", d.CheckType)
		}
		if d.ConfigExample == "" {
			t.Errorf("kind %q has empty config example", d.CheckType)
		}
	}
}

// dispatchedKinds parses ../config/config.go and returns the set of check
// kind string values that appear as case labels in normalizeCheck's switch.
func dispatchedKinds(t *testing.T) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "../config/config.go", nil, 0)
	if err != nil {
		t.Fatalf("parse config.go: %v", err)
	}

	// Map every CheckType constant name to its string value.
	values := map[string]string{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
				continue
			}
			lit, ok := vs.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			val, err := strconv.Unquote(lit.Value)
			if err != nil {
				continue
			}
			values[vs.Names[0].Name] = val
		}
	}

	// Find normalizeCheck and walk its switch case labels.
	kinds := map[string]bool{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "normalizeCheck" {
			continue
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			cc, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range cc.List {
				ident, ok := expr.(*ast.Ident)
				if !ok {
					continue // skip the `case "":` literal
				}
				if val, ok := values[ident.Name]; ok {
					kinds[val] = true
				}
			}
			return true
		})
	}

	if len(kinds) == 0 {
		t.Fatal("found no dispatched kinds in normalizeCheck; parser logic is broken")
	}
	return kinds
}
