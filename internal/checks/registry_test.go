package checks_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/checks"
	_ "github.com/abegong/katalyst/internal/checks/all" // populate the registry
)

// TestDescriptorMetadata checks each descriptor is internally well-formed so
// the generator has everything it needs. The registry is now the single
// enumeration of check types (config no longer keeps a parallel switch), so the
// former no-orphan parity test against config.normalizeCheck is gone.
func TestDescriptorMetadata(t *testing.T) {
	families := map[string]bool{}
	for _, f := range checks.Families() {
		families[f.ID] = true
	}
	seenSlug := map[string]bool{}
	for _, d := range checks.Descriptors() {
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

// TestDescriptorLibrary enforces that every check type names a registered
// library. With the native families migrated onto CheckLibrary, every check
// type has an owning library, so an empty Library is now a registration bug.
func TestDescriptorLibrary(t *testing.T) {
	for _, d := range checks.Descriptors() {
		if d.Library == "" {
			t.Errorf("kind %q names no library; every check type must register through a CheckLibrary", d.CheckType)
			continue
		}
		if _, ok := checks.LibraryByName(d.Library); !ok {
			t.Errorf("kind %q names library %q, which is not registered", d.CheckType, d.Library)
		}
	}
}

func TestDescriptorConfigurableIn(t *testing.T) {
	if !checks.SupportsConfiguration(checks.CheckFilesystemNameCase, checks.ConfigCollection) {
		t.Fatalf("filesystem_name_case should support collection checks")
	}
	if !checks.SupportsConfiguration(checks.CheckFilesystemNameCase, checks.ConfigFilesystem) {
		t.Fatalf("filesystem_name_case should support filesystem checks")
	}
	if checks.SupportsConfiguration(checks.CheckMarkdownRequiresH1, checks.ConfigFilesystem) {
		t.Fatalf("markdown_requires_h1 should remain collection-only")
	}
	if !checks.SupportsConfiguration(checks.CheckFilesystemUnmatchedFiles, checks.ConfigFilesystem) {
		t.Fatalf("filesystem_unmatched_files should support filesystem checks")
	}
	if checks.SupportsConfiguration(checks.CheckFilesystemUnmatchedFiles, checks.ConfigCollection) {
		t.Fatalf("filesystem_unmatched_files should not support collection checks")
	}
}

func TestDescriptorNeedsDocument(t *testing.T) {
	if checks.NeedsDocument(checks.CheckFilesystemNameCase) {
		t.Fatalf("filesystem_name_case should stay path-only")
	}
	if !checks.NeedsDocument(checks.CheckFilesystemNameMatchesField) {
		t.Fatalf("filesystem_name_matches_field should need document metadata")
	}
	if !checks.NeedsDocument(checks.CheckFilesystemUniqueField) {
		t.Fatalf("filesystem_unique_field should need document metadata")
	}
}
