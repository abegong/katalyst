package inspect_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
)

// TestRegistryParity is the no-orphan guarantee: every inspector in All() has a
// Descriptor and vice versa, so an inspector cannot ship undocumented.
func TestRegistryParity(t *testing.T) {
	descriptors := map[string]bool{}
	for _, d := range inspect.Descriptors() {
		if descriptors[d.Name] {
			t.Errorf("duplicate descriptor %q", d.Name)
		}
		descriptors[d.Name] = true
	}

	inspectors := map[string]bool{}
	for _, ins := range inspect.All() {
		n := ins.Name()
		if inspectors[n] {
			t.Errorf("duplicate inspector %q", n)
		}
		inspectors[n] = true
		if !descriptors[n] {
			t.Errorf("inspector %q has no Descriptor in registry.go", n)
		}
	}
	for n := range descriptors {
		if !inspectors[n] {
			t.Errorf("descriptor %q has no inspector in All()", n)
		}
	}
}

// TestDescriptorMetadata checks each descriptor is internally well-formed.
func TestDescriptorMetadata(t *testing.T) {
	families := map[string]bool{}
	for _, f := range inspect.Families() {
		families[f.ID] = true
	}
	// Slug must be unique within a family so per-inspector docs pages
	// (reference/inspectors/<family>/<slug>.md) never collide.
	slugs := map[string]bool{}
	for _, d := range inspect.Descriptors() {
		if d.Family == "" || !families[d.Family] {
			t.Errorf("inspector %q has unknown family %q", d.Name, d.Family)
		}
		if d.Summary == "" {
			t.Errorf("inspector %q has empty summary", d.Name)
		}
		if d.Title == "" {
			t.Errorf("inspector %q has empty title", d.Name)
		}
		if d.Slug == "" {
			t.Errorf("inspector %q has empty slug", d.Name)
		}
		key := d.Family + "/" + d.Slug
		if slugs[key] {
			t.Errorf("inspector %q has duplicate family/slug %q", d.Name, key)
		}
		slugs[key] = true
	}
}
