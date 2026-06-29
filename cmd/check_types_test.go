package cmd_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCheckTypes_listsEveryTypeGroupedByFamily(t *testing.T) {
	// No project on disk: check-types reads the engine registry, not config.
	chdir(t, t.TempDir())

	stdout, _, err := runRoot(t, "check-types", "list")
	if err != nil {
		t.Fatalf("check-types list: %v", err)
	}

	for _, want := range []string{"object_required_field", "markdown_requires_h1", "filesystem_name_case", "text_requires"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected check type %q in output", want)
		}
	}

	// Family titles appear in Families() order.
	last := -1
	for _, title := range []string{
		"Structured object",
		"Markdown body text",
		"File system",
		"Plain text",
	} {
		i := strings.Index(stdout, title)
		if i < 0 {
			t.Errorf("expected family title %q in output", title)
			continue
		}
		if i < last {
			t.Errorf("family %q out of order", title)
		}
		last = i
	}
}

// TestCheckTypes_rulesAliasStillWorks pins the deprecated `rules` alias: it must
// resolve to the same command so existing usage keeps working.
func TestCheckTypes_rulesAliasStillWorks(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "list")
	if err != nil {
		t.Fatalf("rules list (alias): %v", err)
	}
	if !strings.Contains(stdout, "object_required_field") {
		t.Errorf("alias output missing a known check type, got: %q", stdout)
	}
}

// The full catalog's bulleted layout (family headings, required/optional split,
// dash placeholders for no-field checks) is pinned as a snapshot;
// TestCheckTypes_listsEveryTypeGroupedByFamily keeps the registry-coverage and
// family-order invariant against the live registry.
func TestCheckTypesList_textContract(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "list")
	if err != nil {
		t.Fatalf("check-types list: %v", err)
	}
	snapshot(t, "check-types/list.txt", stdout)
}

func TestCheckTypesShow_showsDetail(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "show", "object_required_field")
	if err != nil {
		t.Fatalf("check-types show object_required_field: %v", err)
	}
	snapshot(t, "check-types/show-object_required_field.txt", stdout)
}

func TestCheckTypesShow_unknown_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "check-types", "show", "no_such_type")
	if err == nil {
		t.Fatalf("expected error for unknown check type")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestCheckTypesList_familyFiltersList(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "list", "--family", "markdownBodyText")
	if err != nil {
		t.Fatalf("check-types list --family markdownBodyText: %v", err)
	}
	// The fixture pins the filtered output: only the markdown body text family,
	// no object/filesystem types leak in.
	snapshot(t, "check-types/list-family-markdown.txt", stdout)
}

func TestCheckTypesList_unknownFamily_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "check-types", "list", "--family", "nope")
	if err == nil {
		t.Fatalf("expected error for unknown family")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestCheckTypesList_familyJSONFiltersToFamily(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "list", "--family", "fileSystem", "--json")
	if err != nil {
		t.Fatalf("check-types list --family fileSystem --json: %v", err)
	}
	var got []struct {
		CheckType string `json:"check_type"`
		Family    string `json:"family"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	want := checkTypesByFamily(t, "fileSystem")
	if len(got) != len(want) {
		t.Fatalf("got %d fileSystem descriptors, want %d", len(got), len(want))
	}
	for _, d := range got {
		if d.Family != "fileSystem" {
			t.Errorf("got non-fileSystem family %q", d.Family)
		}
	}
}

// TestCheckTypes_bare_printsHelpNotList pins the grammar rule: a resource noun
// invoked bare prints help and never silently lists (see cmd/AGENTS.md). It
// must show its sub-verbs and not the catalog a `list` would print.
func TestCheckTypes_bare_printsHelpNotList(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types")
	if err != nil {
		t.Fatalf("check-types: %v", err)
	}
	for _, want := range []string{"Usage:", "list", "show"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected help to mention %q, got: %q", want, stdout)
		}
	}
	// The catalog must not leak: bare `check-types` is not an action.
	if strings.Contains(stdout, "object_required_field") {
		t.Errorf("bare check-types listed types instead of printing help: %q", stdout)
	}
}

func TestCheckTypesShow_showsFamilyContextAndSiblings(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "show", "object_field_enum")
	if err != nil {
		t.Fatalf("check-types show object_field_enum: %v", err)
	}
	// The fixture pins the breadcrumb header, the family intro, and the sibling
	// list that give the docs-traversal context.
	snapshot(t, "check-types/show-object_field_enum.txt", stdout)
}

func TestCheckTypesShow_noFieldTypeStatesSo(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "show", "markdown_single_h1")
	if err != nil {
		t.Fatalf("check-types show markdown_single_h1: %v", err)
	}
	// A no-field check states it has no configuration keys.
	snapshot(t, "check-types/show-markdown_single_h1.txt", stdout)
}

func TestCheckTypesList_jsonArrayShape(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "list", "--json")
	if err != nil {
		t.Fatalf("check-types list --json: %v", err)
	}

	var got []struct {
		CheckType string   `json:"check_type"`
		Family    string   `json:"family"`
		Targets   []string `json:"targets"`
		Fields    []struct {
			Name string `json:"name"`
		} `json:"fields"`
		ConfigExample string `json:"config_example"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one descriptor")
	}
	seen := map[string]bool{}
	targets := map[string][]string{}
	for i, d := range got {
		seen[d.CheckType] = true
		targets[d.CheckType] = d.Targets
		if got[i].ConfigExample == "" {
			t.Errorf("entry %d (%s): empty config_example", i, d.CheckType)
		}
	}
	for _, want := range []string{"object_required_field", "markdown_requires_h1", "filesystem_name_case", "text_requires"} {
		if !seen[want] {
			t.Errorf("expected descriptor %q in JSON output", want)
		}
	}

	// Wire-shape guarantees: snake_case keys, no null fields, no empty default.
	if !strings.Contains(stdout, `"config_example"`) {
		t.Errorf("expected snake_case config_example key")
	}
	if !strings.Contains(stdout, `"check_type"`) {
		t.Errorf("expected snake_case check_type key")
	}
	if strings.Contains(stdout, `"fields": null`) {
		t.Errorf("a no-field check type emitted null instead of []")
	}
	if !strings.Contains(stdout, `"fields": []`) {
		t.Errorf("expected at least one no-field check type to emit []")
	}
	if strings.Contains(stdout, `"default": ""`) {
		t.Errorf("empty default should be omitted, not emitted")
	}
	if strings.Join(targets["filesystem_name_case"], ",") != "collection,filesystem" {
		t.Errorf("filesystem_name_case targets = %v, want collection+filesystem", targets["filesystem_name_case"])
	}
	if strings.Join(targets["markdown_requires_h1"], ",") != "collection" {
		t.Errorf("markdown_requires_h1 targets = %v, want collection", targets["markdown_requires_h1"])
	}
}

func TestCheckTypesShow_jsonObject(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "show", "object_number_range", "--json")
	if err != nil {
		t.Fatalf("check-types show object_number_range --json: %v", err)
	}

	var got struct {
		CheckType string   `json:"check_type"`
		Targets   []string `json:"targets"`
		Fields    []struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if got.CheckType != "object_number_range" {
		t.Errorf("got check type %q, want object_number_range", got.CheckType)
	}
	if strings.Join(got.Targets, ",") != "collection" {
		t.Errorf("targets = %v, want collection", got.Targets)
	}
	if len(got.Fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(got.Fields))
	}
}

func checkTypesByFamily(t *testing.T, family string) []string {
	t.Helper()
	stdout, _, err := runRoot(t, "check-types", "list", "--json")
	if err != nil {
		t.Fatalf("check-types list --json: %v", err)
	}
	var got []struct {
		CheckType string `json:"check_type"`
		Family    string `json:"family"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	var out []string
	for _, d := range got {
		if d.Family == family {
			out = append(out, d.CheckType)
		}
	}
	return out
}
