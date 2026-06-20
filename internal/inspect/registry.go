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
// checks.Descriptor.
type Descriptor struct {
	// Name is the inspector's identifier, used by --inspector and as the
	// "inspector" field in evidence.
	Name string
	// Family groups the inspector; one of Families().
	Family string
	// Summary is a one-line statement of what the inspector reports.
	Summary string
}

// Descriptors returns every inspector in display order. The order is authored,
// not sorted, so generated output is deterministic.
func Descriptors() []Descriptor {
	return []Descriptor{
		{
			Name:    "walk_parse",
			Family:  "structural",
			Summary: "Count files and report how many parse and carry frontmatter.",
		},
		{
			Name:    "frontmatter_shape",
			Family:  "structural",
			Summary: "Group files by their frontmatter key-set and report observed field types.",
		},
		{
			Name:    "object_field_frequency",
			Family:  "object",
			Summary: "Report, per frontmatter key, how many files contain it.",
		},
		{
			Name:    "object_field_types",
			Family:  "object",
			Summary: "Report, per key, the histogram of observed value types.",
		},
		{
			Name:    "object_field_values",
			Family:  "object",
			Summary: "Report, per key, value cardinality and a small value set when it looks like an enum.",
		},
		{
			Name:    "object_field_numeric_range",
			Family:  "object",
			Summary: "Report the observed min and max of numeric fields.",
		},
		{
			Name:    "object_field_string_length",
			Family:  "object",
			Summary: "Report the observed min and max length of string fields.",
		},
		{
			Name:    "markdown_heading_shape",
			Family:  "markdown",
			Summary: "Report single-H1, H1-matches-title, and heading-level-jump rates.",
		},
		{
			Name:    "markdown_sections",
			Family:  "markdown",
			Summary: "Report recurring section headings and how many files contain each.",
		},
		{
			Name:    "markdown_code_fences",
			Family:  "markdown",
			Summary: "Report how many fenced code blocks open and how many carry a language tag.",
		},
		{
			Name:    "filesystem_naming",
			Family:  "filesystem",
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
