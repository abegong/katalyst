package cmd_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestInspectors_listsEveryInspectorGroupedByLayer(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list")
	if err != nil {
		t.Fatalf("inspectors list: %v", err)
	}

	for _, want := range []string{"file_tree", "file_content_shape", "object_fields", "markdown_body"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected inspector %q in output", want)
		}
	}

	last := -1
	for _, title := range []string{"Raw base inspectors", "Collection inspectors"} {
		i := strings.Index(stdout, title)
		if i < 0 {
			t.Errorf("expected layer title %q in output", title)
			continue
		}
		if i < last {
			t.Errorf("layer %q out of order", title)
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

func TestInspectorsShow_showsSourceLayerContextAndSiblings(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "show", "file_content_shape")
	if err != nil {
		t.Fatalf("inspectors show file_content_shape: %v", err)
	}
	// The fixture pins the breadcrumb header, the layer intro, and the sibling
	// list.
	snapshot(t, "inspectors/show-file_content_shape.txt", stdout)
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
	if len(got) == 0 {
		t.Fatal("expected at least one descriptor")
	}
	seen := map[string]bool{}
	for i, d := range got {
		seen[d.Name] = true
		if got[i].Layer == "" || got[i].Summary == "" {
			t.Errorf("entry %d (%s): empty layer/summary", i, d.Name)
		}
	}
	for _, want := range []string{"file_tree", "file_content_shape", "object_fields", "markdown_body"} {
		if !seen[want] {
			t.Errorf("expected inspector %q in JSON output", want)
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
	want := inspectorsByLayer(t, "source")
	if len(got) != len(want) {
		t.Fatalf("got %d source descriptors, want %d", len(got), len(want))
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

func inspectorsByLayer(t *testing.T, layer string) []string {
	t.Helper()
	stdout, _, err := runRoot(t, "inspectors", "list", "--json")
	if err != nil {
		t.Fatalf("inspectors list --json: %v", err)
	}
	var got []struct {
		Name  string `json:"name"`
		Layer string `json:"layer"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	var out []string
	for _, d := range got {
		if d.Layer == layer {
			out = append(out, d.Name)
		}
	}
	return out
}
