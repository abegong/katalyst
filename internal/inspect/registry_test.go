package inspect_test

import (
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
)

// TestRegistryParity is the no-orphan guarantee: every inspector instance has a
// Descriptor in its layer and vice versa, so an inspector cannot ship
// undocumented.
func TestRegistryParity(t *testing.T) {
	byLayer := map[string]map[string]bool{"source": {}, "collection": {}}
	for _, d := range inspect.Descriptors() {
		layer := byLayer[d.Layer]
		if layer == nil {
			t.Fatalf("descriptor %q has unknown layer %q", d.Name, d.Layer)
		}
		if layer[d.Name] {
			t.Errorf("duplicate descriptor %q", d.Name)
		}
		layer[d.Name] = true
	}

	source := map[string]bool{}
	for _, ins := range inspect.SourceInspectors() {
		source[ins.Name()] = true
		if !byLayer["source"][ins.Name()] {
			t.Errorf("source inspector %q has no source Descriptor", ins.Name())
		}
	}
	for n := range byLayer["source"] {
		if !source[n] {
			t.Errorf("source descriptor %q has no inspector", n)
		}
	}

	collection := map[string]bool{}
	for _, ins := range inspect.CollectionInspectors() {
		collection[ins.Name()] = true
		if !byLayer["collection"][ins.Name()] {
			t.Errorf("collection inspector %q has no collection Descriptor", ins.Name())
		}
	}
	for n := range byLayer["collection"] {
		if !collection[n] {
			t.Errorf("collection descriptor %q has no inspector", n)
		}
	}
}

// TestDescriptorMetadata checks each descriptor is internally well-formed.
func TestDescriptorMetadata(t *testing.T) {
	families := map[string]bool{}
	for _, f := range inspect.Families() {
		families[f.ID] = true
	}
	// Slug must be unique within a layer so per-inspector docs pages
	// (reference/inspectors/<layer>/<slug>.md) never collide.
	slugs := map[string]bool{}
	for _, d := range inspect.Descriptors() {
		if d.Layer != "source" && d.Layer != "collection" {
			t.Errorf("inspector %q has unknown layer %q", d.Name, d.Layer)
		}
		if d.Family == "" || !families[d.Family] {
			t.Errorf("inspector %q has unknown family %q", d.Name, d.Family)
		}
		if d.Summary == "" {
			t.Errorf("inspector %q has empty summary", d.Name)
		}
		if d.Title == "" {
			t.Errorf("inspector %q has empty title", d.Name)
		}
		key := d.Layer + "/" + d.Slug
		if d.Slug == "" || slugs[key] {
			t.Errorf("inspector %q has empty or duplicate layer/slug %q", d.Name, key)
		}
		slugs[key] = true
	}
}
