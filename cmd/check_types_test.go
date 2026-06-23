package cmd_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/checks"
)

func TestCheckTypes_listsEveryTypeGroupedByFamily(t *testing.T) {
	// No project on disk: check-types reads the engine registry, not config.
	chdir(t, t.TempDir())

	stdout, _, err := runRoot(t, "check-types", "list")
	if err != nil {
		t.Fatalf("check-types list: %v", err)
	}

	for _, d := range checks.Descriptors() {
		if !strings.Contains(stdout, string(d.CheckType)) {
			t.Errorf("expected check type %q in output", d.CheckType)
		}
	}

	// Family titles appear in Families() order.
	last := -1
	for _, fam := range checks.Families() {
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

func TestCheckTypes_splitsRequiredAndOptionalKeys(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "list")
	if err != nil {
		t.Fatalf("check-types list: %v", err)
	}

	// object_number_range: field required, min/max optional.
	line := lineContaining(t, stdout, "object_number_range")
	if !strings.Contains(line, "field") {
		t.Errorf("expected required field on number_range line: %q", line)
	}
	if !strings.Contains(line, "min") || !strings.Contains(line, "max") {
		t.Errorf("expected optional min/max on number_range line: %q", line)
	}

	// A no-field check shows a dash placeholder on both sides.
	line = lineContaining(t, stdout, "markdown_single_h1")
	if strings.Count(line, "-") < 2 {
		t.Errorf("expected dashes for no-field check: %q", line)
	}
}

func TestCheckTypesShow_showsDetail(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "show", "object_required_field")
	if err != nil {
		t.Fatalf("check-types show object_required_field: %v", err)
	}
	for _, want := range []string{
		"object_required_field",      // check type id
		"Require that a frontmatter", // purpose
		"field",                      // key name
		"yes",                        // required column
		"checks:",                    // example body
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in detail output, got: %q", want, stdout)
		}
	}
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
	if !strings.Contains(stdout, "Markdown Body Text Check Types") {
		t.Errorf("expected Markdown Body Text Check Types heading, got: %q", stdout)
	}
	if strings.Contains(stdout, "Structured Object Check Types") || strings.Contains(stdout, "File System Check Types") {
		t.Errorf("expected only the markdown body text family, got: %q", stdout)
	}
	if !strings.Contains(stdout, "markdown_single_h1") {
		t.Errorf("expected a markdown check type, got: %q", stdout)
	}
	if strings.Contains(stdout, "object_required_field") {
		t.Errorf("did not expect an object check type, got: %q", stdout)
	}
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
	if len(got) != len(familyCheckTypes("fileSystem")) {
		t.Fatalf("got %d fileSystem descriptors, want %d", len(got), len(familyCheckTypes("fileSystem")))
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
	// Breadcrumb + family intro give the docs-traversal context.
	if !strings.Contains(stdout, "Structured Object Check Types › Field Enum") {
		t.Errorf("expected breadcrumb header, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Structured-object check types validate structured frontmatter") {
		t.Errorf("expected family intro, got: %q", stdout)
	}
	// Siblings list points at the rest of the family.
	if !strings.Contains(stdout, "object_required_field") {
		t.Errorf("expected a sibling check type, got: %q", stdout)
	}
}

func TestCheckTypesShow_noFieldTypeStatesSo(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "show", "markdown_single_h1")
	if err != nil {
		t.Fatalf("check-types show markdown_single_h1: %v", err)
	}
	if !strings.Contains(stdout, "no configuration keys") {
		t.Errorf("expected no-keys note, got: %q", stdout)
	}
}

// familyCheckTypes returns the registered check types in a family, for test
// expectations.
func familyCheckTypes(family string) []string {
	var out []string
	for _, d := range checks.Descriptors() {
		if d.Family == family {
			out = append(out, string(d.CheckType))
		}
	}
	return out
}

func TestCheckTypesList_jsonArrayCoversEveryDescriptor(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "list", "--json")
	if err != nil {
		t.Fatalf("check-types list --json: %v", err)
	}

	var got []struct {
		CheckType string `json:"check_type"`
		Family    string `json:"family"`
		Fields    []struct {
			Name string `json:"name"`
		} `json:"fields"`
		ConfigExample string `json:"config_example"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got) != len(checks.Descriptors()) {
		t.Fatalf("got %d descriptors, want %d", len(got), len(checks.Descriptors()))
	}
	for i, d := range checks.Descriptors() {
		if got[i].CheckType != string(d.CheckType) {
			t.Errorf("entry %d: got check type %q, want %q", i, got[i].CheckType, d.CheckType)
		}
		if got[i].ConfigExample == "" {
			t.Errorf("entry %d (%s): empty config_example", i, d.CheckType)
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
}

func TestCheckTypesShow_jsonObject(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "check-types", "show", "object_number_range", "--json")
	if err != nil {
		t.Fatalf("check-types show object_number_range --json: %v", err)
	}

	var got struct {
		CheckType string `json:"check_type"`
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
	if len(got.Fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(got.Fields))
	}
}

// lineContaining returns the first line of s that contains sub, failing if none.
func lineContaining(t *testing.T, s, sub string) string {
	t.Helper()
	for _, ln := range strings.Split(s, "\n") {
		if strings.Contains(ln, sub) {
			return ln
		}
	}
	t.Fatalf("no line containing %q in:\n%s", sub, s)
	return ""
}
