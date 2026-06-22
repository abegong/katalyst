package inspect

// This file is the single source of truth for the inspector set. Every
// inspector returned by All() must have a matching Descriptor here, and vice
// versa; registry_test.go enforces that parity. A new inspector cannot ship
// undocumented, mirroring the checks registry (internal/checks/registry.go).

// Family groups inspectors for display and documentation, mirroring the check
// families. Order is significant: it fixes section ordering in rendered output
// and generated docs.
type Family struct {
	ID    string
	Title string
	Intro string
}

// Families returns the inspector families in display order.
func Families() []Family {
	return []Family{
		{
			ID:    "structural",
			Title: "Structural",
			Intro: "Structural inspectors report corpus-level facts: how files parse and how their frontmatter is shaped.",
		},
		{
			ID:    "object",
			Title: "Object",
			Intro: "Object inspectors report the distribution of frontmatter fields — presence, types, values, ranges.",
		},
		{
			ID:    "markdown",
			Title: "Markdown",
			Intro: "Markdown inspectors report body conventions: headings, sections, and code fences.",
		},
		{
			ID:    "filesystem",
			Title: "Filesystem",
			Intro: "Filesystem inspectors report filename and path conventions across the corpus.",
		},
	}
}

// Descriptor is the machine-readable record for one inspector, mirroring
// checks.Descriptor. Its json tags are the wire contract for
// `katalyst inspectors list --json`; keep them snake_case.
type Descriptor struct {
	// Name is the inspector's identifier, used by --inspector and as the
	// "inspector" field in evidence.
	Name string `json:"name"`
	// Family groups the inspector; one of Families(). It is also the
	// subdirectory under reference/inspectors/.
	Family string `json:"family"`
	// Slug is the page basename under the family directory.
	Slug string `json:"slug"`
	// Title is the human-readable page title.
	Title string `json:"title"`
	// Summary is a one-line statement of what the inspector reports.
	Summary string `json:"summary"`
}

// Descriptors returns every inspector in display order. The order is authored,
// not sorted, so generated output is deterministic.
func Descriptors() []Descriptor {
	return []Descriptor{
		{
			Name:    "walk_parse",
			Family:  "structural",
			Slug:    "walk-parse",
			Title:   "Walk & Parse",
			Summary: "Count files and report how many parse and carry frontmatter.",
		},
		{
			Name:    "frontmatter_shape",
			Family:  "structural",
			Slug:    "frontmatter-shape",
			Title:   "Frontmatter Shape",
			Summary: "Group files by their frontmatter key-set and report observed field types.",
		},
		{
			Name:    "object_field_frequency",
			Family:  "object",
			Slug:    "field-frequency",
			Title:   "Field Frequency",
			Summary: "Report, per frontmatter key, how many files contain it.",
		},
		{
			Name:    "object_field_types",
			Family:  "object",
			Slug:    "field-types",
			Title:   "Field Types",
			Summary: "Report, per key, the histogram of observed value types.",
		},
		{
			Name:    "object_field_values",
			Family:  "object",
			Slug:    "field-values",
			Title:   "Field Values",
			Summary: "Report, per key, value cardinality and a small value set when it looks like an enum.",
		},
		{
			Name:    "object_field_numeric_range",
			Family:  "object",
			Slug:    "field-numeric-range",
			Title:   "Field Numeric Range",
			Summary: "Report the observed min and max of numeric fields.",
		},
		{
			Name:    "object_field_string_length",
			Family:  "object",
			Slug:    "field-string-length",
			Title:   "Field String Length",
			Summary: "Report the observed min and max length of string fields.",
		},
		{
			Name:    "markdown_heading_shape",
			Family:  "markdown",
			Slug:    "heading-shape",
			Title:   "Heading Shape",
			Summary: "Report single-H1, H1-matches-title, and heading-level-jump rates.",
		},
		{
			Name:    "markdown_sections",
			Family:  "markdown",
			Slug:    "sections",
			Title:   "Sections",
			Summary: "Report recurring section headings and how many files contain each.",
		},
		{
			Name:    "markdown_code_fences",
			Family:  "markdown",
			Slug:    "code-fences",
			Title:   "Code Fences",
			Summary: "Report how many fenced code blocks open and how many carry a language tag.",
		},
		{
			Name:    "filesystem_naming",
			Family:  "filesystem",
			Slug:    "naming",
			Title:   "Naming",
			Summary: "Report filename casing, spaces, extensions, and nesting depth.",
		},
	}
}

// All returns every inspector instance in display order.
func All() []Inspector {
	return []Inspector{
		WalkParse{},
		FrontmatterShape{},
		ObjectFieldFrequency{},
		ObjectFieldTypes{},
		ObjectFieldValues{},
		ObjectFieldNumericRange{},
		ObjectFieldStringLength{},
		MarkdownHeadingShape{},
		MarkdownSections{},
		MarkdownCodeFences{},
		FilesystemNaming{},
	}
}

// ByName returns the inspector with the given name.
func ByName(name string) (Inspector, bool) {
	for _, ins := range All() {
		if ins.Name() == name {
			return ins, true
		}
	}
	return nil, false
}

// Summary returns the one-line description of what an inspector's results mean,
// from its registry descriptor. The empty string if the name is unknown.
func Summary(name string) string {
	for _, d := range Descriptors() {
		if d.Name == name {
			return d.Summary
		}
	}
	return ""
}
