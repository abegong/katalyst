package cmd_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
)

func TestInspectors_listsEveryInspectorGroupedByLayer(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list")
	if err != nil {
		t.Fatalf("inspectors list: %v", err)
	}

	for _, d := range inspect.Descriptors() {
		if !strings.Contains(stdout, d.Name) {
			t.Errorf("expected inspector %q in output", d.Name)
		}
	}

	last := -1
	for _, l := range inspect.Layers() {
		i := strings.Index(stdout, l.Title)
		if i < 0 {
			t.Errorf("expected layer title %q in output", l.Title)
			continue
		}
		if i < last {
			t.Errorf("layer %q out of order", l.Title)
		}
		last = i
	}
}

// The full catalog's layout (layer headings, column alignment) is pinned as a
// snapshot; TestInspectors_listsEveryInspectorGroupedByLayer keeps the
// registry-coverage and layer-order invariant against the live registry.
func TestInspectorsList_textContract(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list")
	if err != nil {
		t.Fatalf("inspectors list: %v", err)
	}
	snapshot(t, "inspectors/list.txt", stdout)
}

func TestInspectorsShow_showsDetail(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "show", "object_fields")
	if err != nil {
		t.Fatalf("inspectors show object_fields: %v", err)
	}
	snapshot(t, "inspectors/show-object_fields.txt", stdout)
}

func TestInspectorsShow_showsLayerContextAndSiblings(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "show", "document_shape")
	if err != nil {
		t.Fatalf("inspectors show document_shape: %v", err)
	}
	// The fixture pins the breadcrumb header, the layer intro, and the sibling
	// list.
	snapshot(t, "inspectors/show-document_shape.txt", stdout)
}

func TestInspectorsShow_unknown_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "inspectors", "show", "no_such_inspector")
	if err == nil {
		t.Fatalf("expected error for unknown inspector")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestInspectorsList_layerFiltersList(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list", "--layer", "collection")
	if err != nil {
		t.Fatalf("inspectors list --layer collection: %v", err)
	}
	// The fixture pins the filtered output: only the collection layer, no
	// source inspectors leak in.
	snapshot(t, "inspectors/list-layer-collection.txt", stdout)
}

func TestInspectorsList_unknownLayer_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "inspectors", "list", "--layer", "nope")
	if err == nil {
		t.Fatalf("expected error for unknown layer")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestInspectorsList_jsonArrayCoversEveryDescriptor(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list", "--json")
	if err != nil {
		t.Fatalf("inspectors list --json: %v", err)
	}
	var got []struct {
		Name    string `json:"name"`
		Layer   string `json:"layer"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got) != len(inspect.Descriptors()) {
		t.Fatalf("got %d descriptors, want %d", len(got), len(inspect.Descriptors()))
	}
	for i, d := range inspect.Descriptors() {
		if got[i].Name != d.Name {
			t.Errorf("entry %d: got %q, want %q", i, got[i].Name, d.Name)
		}
		if got[i].Layer == "" || got[i].Summary == "" {
			t.Errorf("entry %d (%s): empty layer/summary", i, d.Name)
		}
	}
	if !strings.Contains(stdout, `"layer"`) {
		t.Errorf("expected snake_case layer key")
	}
}

func TestInspectorsList_layerJSONFiltersToLayer(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list", "--layer", "source", "--json")
	if err != nil {
		t.Fatalf("inspectors list --layer source --json: %v", err)
	}
	var got []struct {
		Name  string `json:"name"`
		Layer string `json:"layer"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got) != len(layerInspectors("source")) {
		t.Fatalf("got %d source descriptors, want %d", len(got), len(layerInspectors("source")))
	}
	for _, d := range got {
		if d.Layer != "source" {
			t.Errorf("got non-source layer %q", d.Layer)
		}
	}
}

func TestInspectorsShow_jsonObject(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "show", "file_tree", "--json")
	if err != nil {
		t.Fatalf("inspectors show file_tree --json: %v", err)
	}
	var got struct {
		Name  string `json:"name"`
		Layer string `json:"layer"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got.Name != "file_tree" || got.Layer != "source" {
		t.Errorf("got %+v, want file_tree/source", got)
	}
}

func TestInspectors_bare_printsHelpNotList(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors")
	if err != nil {
		t.Fatalf("inspectors: %v", err)
	}
	for _, want := range []string{"Usage:", "list", "show"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected help to mention %q, got: %q", want, stdout)
		}
	}
	if strings.Contains(stdout, "object_fields") {
		t.Errorf("bare inspectors listed inspectors instead of printing help: %q", stdout)
	}
}

// layerInspectors returns the registered inspector names in a layer.
func layerInspectors(layer string) []string {
	var out []string
	for _, d := range inspect.Descriptors() {
		if d.Layer == layer {
			out = append(out, d.Name)
		}
	}
	return out
}
