package cmd_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/katabase-ai/katalyst/internal/checks"
)

func TestRules_listsEveryKindGroupedByFamily(t *testing.T) {
	// No project on disk: rules reads the engine registry, not config.
	chdir(t, t.TempDir())

	stdout, _, err := runRoot(t, "rules", "list")
	if err != nil {
		t.Fatalf("rules list: %v", err)
	}

	for _, d := range checks.Descriptors() {
		if !strings.Contains(stdout, string(d.Kind)) {
			t.Errorf("expected kind %q in output", d.Kind)
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

func TestRules_splitsRequiredAndOptionalKeys(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "list")
	if err != nil {
		t.Fatalf("rules list: %v", err)
	}

	// object_number_range: field required, min/max optional.
	line := lineContaining(t, stdout, "object_number_range")
	if !strings.Contains(line, "field") {
		t.Errorf("expected required field on number_range line: %q", line)
	}
	if !strings.Contains(line, "min") || !strings.Contains(line, "max") {
		t.Errorf("expected optional min/max on number_range line: %q", line)
	}

	// A no-field check shows an em dash on both sides.
	line = lineContaining(t, stdout, "markdown_single_h1")
	if strings.Count(line, "—") < 2 {
		t.Errorf("expected dashes for no-field check: %q", line)
	}
}

func TestRulesShow_showsDetail(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "show", "object_required_field")
	if err != nil {
		t.Fatalf("rules show object_required_field: %v", err)
	}
	for _, want := range []string{
		"object_required_field",      // kind id
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

func TestRulesShow_unknown_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "rules", "show", "no_such_kind")
	if err == nil {
		t.Fatalf("expected error for unknown kind")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestRulesList_familyFiltersList(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "list", "--family", "markdown")
	if err != nil {
		t.Fatalf("rules list --family markdown: %v", err)
	}
	if !strings.Contains(stdout, "Markdown Rules") {
		t.Errorf("expected Markdown Rules heading, got: %q", stdout)
	}
	if strings.Contains(stdout, "Object Rules") || strings.Contains(stdout, "Filesystem Rules") {
		t.Errorf("expected only the markdown family, got: %q", stdout)
	}
	if !strings.Contains(stdout, "markdown_single_h1") {
		t.Errorf("expected a markdown kind, got: %q", stdout)
	}
	if strings.Contains(stdout, "object_required_field") {
		t.Errorf("did not expect an object kind, got: %q", stdout)
	}
}

func TestRulesList_unknownFamily_exit2(t *testing.T) {
	chdir(t, t.TempDir())
	_, _, err := runRoot(t, "rules", "list", "--family", "nope")
	if err == nil {
		t.Fatalf("expected error for unknown family")
	}
	var coded interface{ Code() int }
	if !errors.As(err, &coded) || coded.Code() != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestRulesList_familyJSONFiltersToFamily(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "list", "--family", "filesystem", "--json")
	if err != nil {
		t.Fatalf("rules list --family filesystem --json: %v", err)
	}
	var got []struct {
		Kind   string `json:"kind"`
		Family string `json:"family"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(got) != len(familyKinds("filesystem")) {
		t.Fatalf("got %d filesystem descriptors, want %d", len(got), len(familyKinds("filesystem")))
	}
	for _, d := range got {
		if d.Family != "filesystem" {
			t.Errorf("got non-filesystem family %q", d.Family)
		}
	}
}

// TestRules_bare_printsHelpNotList pins the grammar rule: a resource noun
// invoked bare prints help and never silently lists (see cmd/AGENTS.md). It
// must show its sub-verbs and not the catalog a `list` would print.
func TestRules_bare_printsHelpNotList(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules")
	if err != nil {
		t.Fatalf("rules: %v", err)
	}
	for _, want := range []string{"Usage:", "list", "show"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected help to mention %q, got: %q", want, stdout)
		}
	}
	// The catalog must not leak: bare `rules` is not an action.
	if strings.Contains(stdout, "object_required_field") {
		t.Errorf("bare rules listed kinds instead of printing help: %q", stdout)
	}
}

func TestRulesShow_showsFamilyContextAndSiblings(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "show", "object_field_enum")
	if err != nil {
		t.Fatalf("rules show object_field_enum: %v", err)
	}
	// Breadcrumb + family intro give the docs-traversal context.
	if !strings.Contains(stdout, "Object Rules › Field Enum") {
		t.Errorf("expected breadcrumb header, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Object rules validate structured frontmatter") {
		t.Errorf("expected family intro, got: %q", stdout)
	}
	// Siblings list points at the rest of the family.
	if !strings.Contains(stdout, "object_required_field") {
		t.Errorf("expected a sibling kind, got: %q", stdout)
	}
}

func TestRulesShow_noFieldKindStatesSo(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "show", "markdown_single_h1")
	if err != nil {
		t.Fatalf("rules show markdown_single_h1: %v", err)
	}
	if !strings.Contains(stdout, "no configuration keys") {
		t.Errorf("expected no-keys note, got: %q", stdout)
	}
}

// familyKinds returns the registered kinds in a family, for test expectations.
func familyKinds(family string) []string {
	var out []string
	for _, d := range checks.Descriptors() {
		if d.Family == family {
			out = append(out, string(d.Kind))
		}
	}
	return out
}

func TestRulesList_jsonArrayCoversEveryDescriptor(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "list", "--json")
	if err != nil {
		t.Fatalf("rules list --json: %v", err)
	}

	var got []struct {
		Kind   string `json:"kind"`
		Family string `json:"family"`
		Fields []struct {
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
		if got[i].Kind != string(d.Kind) {
			t.Errorf("entry %d: got kind %q, want %q", i, got[i].Kind, d.Kind)
		}
		if got[i].ConfigExample == "" {
			t.Errorf("entry %d (%s): empty config_example", i, d.Kind)
		}
	}

	// Wire-shape guarantees: snake_case keys, no null fields, no empty default.
	if !strings.Contains(stdout, `"config_example"`) {
		t.Errorf("expected snake_case config_example key")
	}
	if strings.Contains(stdout, `"fields": null`) {
		t.Errorf("a no-field check emitted null instead of []")
	}
	if !strings.Contains(stdout, `"fields": []`) {
		t.Errorf("expected at least one no-field check to emit []")
	}
	if strings.Contains(stdout, `"default": ""`) {
		t.Errorf("empty default should be omitted, not emitted")
	}
}

func TestRulesShow_jsonObject(t *testing.T) {
	chdir(t, t.TempDir())
	stdout, _, err := runRoot(t, "rules", "show", "object_number_range", "--json")
	if err != nil {
		t.Fatalf("rules show object_number_range --json: %v", err)
	}

	var got struct {
		Kind   string `json:"kind"`
		Fields []struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if got.Kind != "object_number_range" {
		t.Errorf("got kind %q, want object_number_range", got.Kind)
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
