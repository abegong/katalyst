package argcheck_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/checks/argcheck"
)

func TestRequireString(t *testing.T) {
	if err := argcheck.RequireString("object_required_field", "field", ""); err == nil {
		t.Fatal("expected error for empty value")
	} else if got, want := err.Error(), `object_required_field requires "field"`; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
	if err := argcheck.RequireString("k", "field", "x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequireStrings(t *testing.T) {
	if err := argcheck.RequireStrings("filesystem_extension_in", "values", nil); err == nil {
		t.Fatal("expected error for empty slice")
	} else if got, want := err.Error(), `filesystem_extension_in requires "values"`; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestRequireOneOfFields(t *testing.T) {
	err := argcheck.RequireOneOfFields("filesystem_name_affix", false, "prefix", "suffix")
	if err == nil {
		t.Fatal("expected error when no field present")
	}
	if got, want := err.Error(), `filesystem_name_affix requires "prefix" or "suffix"`; got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
	if err := argcheck.RequireOneOfFields("k", true, "prefix", "suffix"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOneOf(t *testing.T) {
	if err := argcheck.OneOf("filesystem_name_matches_field", "transform", "shout", "none", "slugify"); err == nil {
		t.Fatal("expected error for disallowed value")
	}
	if err := argcheck.OneOf("k", "transform", "slugify", "none", "slugify"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := argcheck.OneOf("k", "transform", "", "none", "slugify"); err != nil {
		t.Fatalf("empty value should pass: %v", err)
	}
}
