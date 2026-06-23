package checks_test

import (
	"errors"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
)

// fakeOOPSchema models a batched out-of-process run: its findings name the file
// they came from via Violation.File, the way a tool linting a whole collection
// in one invocation attributes results back to individual items.
type fakeOOPSchema struct{ out []checks.Violation }

func (f fakeOOPSchema) Check(checks.Context) []checks.Violation { return f.out }

// fakeOOPLibrary models an out-of-process SchemaLibrary whose binary may be
// missing (Available returns an error).
type fakeOOPLibrary struct {
	name   string
	avail  error
	schema checks.Schema
}

func (f fakeOOPLibrary) Name() string     { return f.name }
func (f fakeOOPLibrary) Available() error { return f.avail }
func (f fakeOOPLibrary) CompileSchema(string, []byte) (checks.Schema, error) {
	return f.schema, nil
}

// schemaCheck adapts a Schema to a Check, the way a schema-backed library's
// check type does (see jsonschema.Object).
type schemaCheck struct{ s checks.Schema }

func (c schemaCheck) Run(ctx checks.Context) []checks.Violation { return c.s.Check(ctx) }

// An out-of-process library carries its availability through the interface, so
// the engine can refuse to run when the tool is missing.
func TestSchemaLibrary_availabilitySurfaces(t *testing.T) {
	want := errors.New("vale: command not found")
	checks.RegisterLibrary(fakeOOPLibrary{name: "oop-unavailable", avail: want})

	lib, ok := checks.LibraryByName("oop-unavailable")
	if !ok {
		t.Fatal("library not registered")
	}
	if lib.Available() == nil {
		t.Fatal("expected Available to surface the missing-binary error")
	}
}

// A batched library's findings map back to individual files through
// Violation.File, preserved as they flow through the engine's aggregation.
func TestViolationFile_survivesAggregation(t *testing.T) {
	schema := fakeOOPSchema{out: []checks.Violation{
		{File: "notes/a.md", Message: "prose tell"},
		{File: "notes/b.md", Message: "passive voice"},
	}}

	got := checks.RunAll(checks.Context{}, []checks.Check{schemaCheck{s: schema}})

	files := map[string]string{}
	for _, v := range got {
		files[v.File] = v.Message
	}
	if files["notes/a.md"] != "prose tell" || files["notes/b.md"] != "passive voice" {
		t.Errorf("findings did not map back to files: %+v", got)
	}
}
