package checks_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	_ "github.com/abegong/katalyst/internal/checks/all" // populate the registry
	"github.com/abegong/katalyst/internal/config"
)

// fakeSchema is a compiled schema that returns a fixed violation list.
type fakeSchema struct{ out []checks.Violation }

func (f fakeSchema) Check(checks.Context) []checks.Violation { return f.out }

// fakeLib is a minimal SchemaLibrary for exercising the registry.
type fakeLib struct {
	name   string
	schema checks.Schema
}

func (f fakeLib) Name() string     { return f.name }
func (f fakeLib) Available() error { return nil }
func (f fakeLib) CompileSchema(string, []byte) (checks.Schema, error) {
	return f.schema, nil
}

func TestRegisterLibrary_roundTrip(t *testing.T) {
	lib := fakeLib{name: "test-roundtrip"}
	checks.RegisterLibrary(lib)

	got, ok := checks.LibraryByName("test-roundtrip")
	if !ok {
		t.Fatalf("LibraryByName: registered library not found")
	}
	if got.Name() != "test-roundtrip" {
		t.Errorf("Name = %q, want test-roundtrip", got.Name())
	}

	found := false
	for _, l := range checks.Libraries() {
		if l.Name() == "test-roundtrip" {
			found = true
		}
	}
	if !found {
		t.Errorf("Libraries() does not contain the registered library")
	}
}

func TestRegisterLibrary_duplicatePanics(t *testing.T) {
	checks.RegisterLibrary(fakeLib{name: "test-dup"})
	defer func() {
		if recover() == nil {
			t.Errorf("expected panic on duplicate library name")
		}
	}()
	checks.RegisterLibrary(fakeLib{name: "test-dup"})
}

// A native check type names no library yet, so LibraryFor returns no owner.
func TestLibraryFor_nativeKindHasNoLibrary(t *testing.T) {
	if lib, ok := checks.LibraryFor(config.CheckMarkdownSingleH1); ok {
		t.Errorf("LibraryFor(native kind) = %q, want none", lib.Name())
	}
}

func TestLibraryFor_unknownKind(t *testing.T) {
	if _, ok := checks.LibraryFor(config.CheckType("not_a_real_kind")); ok {
		t.Errorf("LibraryFor(unknown kind) returned a library")
	}
}

func TestSchema_checkRoundTrip(t *testing.T) {
	want := []checks.Violation{{Path: "/x", Message: "bad"}}
	lib := fakeLib{name: "test-schema", schema: fakeSchema{out: want}}

	s, err := lib.CompileSchema("n", []byte("{}"))
	if err != nil {
		t.Fatalf("CompileSchema: %v", err)
	}
	got := s.Check(checks.Context{})
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("Check = %+v, want %+v", got, want)
	}
}
