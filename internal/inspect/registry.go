package inspect

// This file is the single source of truth for the inspector set. Every
// inspector returned by SourceInspectors()/CollectionInspectors() must have a
// matching Descriptor here, and vice versa; registry_test.go enforces that
// parity per layer. A new inspector cannot ship undocumented, mirroring the
// checks registry (internal/checks/registry.go).

// Layer groups inspectors by the data they measure: a raw base (source) or a
// configured collection (collection). It is the primary grouping for display
// and docs. Order is significant.
type Layer struct {
	ID    string
	Title string
	Intro string
}

// Layers returns the inspector layers in display order.
func Layers() []Layer {
	return []Layer{
		{
			ID:    "source",
			Title: "Raw base inspectors",
			Intro: "Raw base inspectors profile a base directly, before any collection configuration: what files are present, how they parse, and how they are named.",
		},
		{
			ID:    "collection",
			Title: "Collection inspectors",
			Intro: "Collection inspectors profile a configured collection's items, probing them through the same substrate the checks use.",
		},
	}
}

// Family is a secondary grouping kept for continuity with the check families
// and for the inspect report's section ordering.
type Family struct {
	ID    string
	Title string
	Intro string
}

// Families returns the inspector families in display order.
func Families() []Family {
	return []Family{
		{ID: "structural", Title: "Structural", Intro: "Structural inspectors report corpus-level facts: how files parse and how their frontmatter is shaped."},
		{ID: "object", Title: "Object", Intro: "Object inspectors report the distribution of frontmatter fields: presence, types, values."},
		{ID: "markdown", Title: "Markdown", Intro: "Markdown inspectors report body conventions: headings and sections."},
		{ID: "filesystem", Title: "Filesystem", Intro: "Filesystem inspectors report filename and path conventions across the corpus."},
	}
}

// Descriptor is the machine-readable record for one inspector, mirroring
// checks.Descriptor. Its json tags are the wire contract for
// `katalyst inspectors list --json`; keep them snake_case.
type Descriptor struct {
	// Name is the inspector's identifier, used by --inspector and as the
	// "inspector" field in evidence.
	Name string `json:"name"`
	// Layer is the data the inspector measures: "source" or "collection".
	Layer string `json:"layer"`
	// Family groups the inspector within its layer; one of Families().
	Family string `json:"family"`
	// Slug is the page basename under the layer directory.
	Slug string `json:"slug"`
	// Title is the human-readable page title.
	Title string `json:"title"`
	// Summary is a one-line statement of what the inspector reports.
	Summary string `json:"summary"`
}

// Descriptors returns every inspector in display order (source layer first).
// The order is authored, not sorted, so generated output is deterministic.
func Descriptors() []Descriptor {
	return []Descriptor{
		{
			Name:    "file_tree",
			Layer:   "source",
			Family:  "filesystem",
			Slug:    "file-tree",
			Title:   "File tree",
			Summary: "Map files, directories, extensions, regions, and filename conventions, opening no files.",
		},
		{
			Name:    "file_content_shape",
			Layer:   "source",
			Family:  "structural",
			Slug:    "file-content-shape",
			Title:   "File content shape",
			Summary: "Profile selected files by text, tabular, and tree content structure.",
		},
		{
			Name:    "object_fields",
			Layer:   "collection",
			Family:  "object",
			Slug:    "object-fields",
			Title:   "Object fields",
			Summary: "A data dictionary over item frontmatter: per-field presence, types, cardinality, and common values.",
		},
		{
			Name:    "markdown_body",
			Layer:   "collection",
			Family:  "markdown",
			Slug:    "markdown-body",
			Title:   "Markdown body",
			Summary: "Body conventions across items: heading shape and recurring sections.",
		},
	}
}

// SourceInspectors returns every raw base inspector instance in display order.
func SourceInspectors() []SourceInspector {
	return []SourceInspector{
		FileTree{},
		FileContentShape{},
	}
}

// CollectionInspectors returns every collection inspector instance in display
// order.
func CollectionInspectors() []CollectionInspector {
	return []CollectionInspector{
		ObjectFields{},
		MarkdownBody{},
	}
}

// SourceByName returns the source inspector with the given name.
func SourceByName(name string) (SourceInspector, bool) {
	for _, ins := range SourceInspectors() {
		if ins.Name() == name {
			return ins, true
		}
	}
	return nil, false
}

// CollectionByName returns the collection inspector with the given name.
func CollectionByName(name string) (CollectionInspector, bool) {
	for _, ins := range CollectionInspectors() {
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
