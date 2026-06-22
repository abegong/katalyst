package cmd_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/inspect"
)

func TestInspectors_listsEveryInspectorGroupedByFamily(t *testing.T) {
	// No project on disk: inspectors reads the engine registry, not config.
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

	// Family titles appear in Families() order.
	last := -1
	for _, fam := range inspect.Families() {
		i := strings.Index(stdout, fam.Title)
		if i < 0 {
			t.Errorf("expected family title %q in output", fam.Title)
			continue
		}
		if i < last {
			t.Errorf("family %q out of order", fam.Title)
		}
		last = i
	}
}

func TestInspectorsShow_showsDetail(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "show", "object_field_frequency")
	if err != nil {
		t.Fatalf("inspectors show object_field_frequency: %v", err)
	}
	for _, want := range []string{
		"object_field_frequency",           // inspector id
		"Report, per frontmatter key",      // purpose
		"--inspector object_field_frequency", // usage hint
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in detail output, got: %q", want, stdout)
		}
	}
}

func TestInspectorsShow_showsFamilyContextAndSiblings(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "show", "object_field_types")
	if err != nil {
		t.Fatalf("inspectors show object_field_types: %v", err)
	}
	// Breadcrumb + family intro give the docs-traversal context.
	if !strings.Contains(stdout, "Object › Field Types") {
		t.Errorf("expected breadcrumb header, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Object inspectors report the distribution") {
		t.Errorf("expected family intro, got: %q", stdout)
	}
	// Siblings list points at the rest of the family.
	if !strings.Contains(stdout, "object_field_frequency") {
		t.Errorf("expected a sibling inspector, got: %q", stdout)
	}
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

func TestInspectorsList_familyFiltersList(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list", "--family", "markdown")
	if err != nil {
		t.Fatalf("inspectors list --family markdown: %v", err)
	}
	if !strings.Contains(stdout, "Markdown") {
		t.Errorf("expected Markdown heading, got: %q", stdout)
	}
	if strings.Contains(stdout, "Structural") || strings.Contains(stdout, "Filesystem") {
		t.Errorf("expected only the markdown family, got: %q", stdout)
	}
	if !strings.Contains(stdout, "markdown_sections") {
		t.Errorf("expected a markdown inspector, got: %q", stdout)
	}
	if strings.Contains(stdout, "walk_parse") {
		t.Errorf("did not expect a structural inspector, got: %q", stdout)
	}
}

func TestInspectorsList_unknownFamily_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "inspectors", "list", "--family", "nope")
	if err == nil {
		t.Fatalf("expected error for unknown family")
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
		Family  string `json:"family"`
		Slug    string `json:"slug"`
		Title   string `json:"title"`
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
			t.Errorf("entry %d: got inspector %q, want %q", i, got[i].Name, d.Name)
		}
		if got[i].Summary == "" {
			t.Errorf("entry %d (%s): empty summary", i, d.Name)
		}
	}

	// Wire-shape guarantee: snake_case keys.
	if !strings.Contains(stdout, `"name"`) || !strings.Contains(stdout, `"summary"`) {
		t.Errorf("expected snake_case name/summary keys")
	}
}

func TestInspectorsList_familyJSONFiltersToFamily(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "list", "--family", "filesystem", "--json")
	if err != nil {
		t.Fatalf("inspectors list --family filesystem --json: %v", err)
	}
	var got []struct {
		Name   string `json:"name"`
		Family string `json:"family"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got) != len(familyInspectors("filesystem")) {
		t.Fatalf("got %d filesystem descriptors, want %d", len(got), len(familyInspectors("filesystem")))
	}
	for _, d := range got {
		if d.Family != "filesystem" {
			t.Errorf("got non-filesystem family %q", d.Family)
		}
	}
}

func TestInspectorsShow_jsonObject(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "inspectors", "show", "filesystem_naming", "--json")
	if err != nil {
		t.Fatalf("inspectors show filesystem_naming --json: %v", err)
	}
	var got struct {
		Name   string `json:"name"`
		Family string `json:"family"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if got.Name != "filesystem_naming" {
		t.Errorf("got inspector %q, want filesystem_naming", got.Name)
	}
}

// TestInspectors_bare_printsHelpNotList pins the grammar rule: a resource noun
// invoked bare prints help and never silently lists (see cmd/AGENTS.md).
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
	// The catalog must not leak: bare `inspectors` is not an action.
	if strings.Contains(stdout, "walk_parse") {
		t.Errorf("bare inspectors listed inspectors instead of printing help: %q", stdout)
	}
}

// familyInspectors returns the registered inspectors in a family, for test
// expectations.
func familyInspectors(family string) []string {
	var out []string
	for _, d := range inspect.Descriptors() {
		if d.Family == family {
			out = append(out, d.Name)
		}
	}
	return out
}
