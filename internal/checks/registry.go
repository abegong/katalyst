package checks

import "github.com/katabase-ai/katalyst/internal/config"

// This file is the single source of truth for rule-reference documentation.
// Every check kind dispatched in config.normalizeCheck must have a matching
// Descriptor here; registry_test.go enforces that parity, and cmd/gendocs
// renders docs/reference/rules/ from these descriptors. A new check cannot
// ship undocumented.

// Field describes one configuration key accepted by a check. The json tags
// are the published wire contract for `katalyst rules list --json`; keep them
// stable and snake_case (matching the config keys they describe) even if the
// Go field names change.
type Field struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
	Desc     string `json:"desc"`
}

// Descriptor is the machine-readable record for one check kind. Its json tags
// are the wire contract for `katalyst rules list --json`; see Field.
type Descriptor struct {
	// Kind is the value used as `kind:` in katalyst.yaml.
	Kind config.CheckKind `json:"kind"`
	// Family groups the check on the docs site: "objects", "markdown", or
	// "filesystem". It is also the subdirectory under reference/rules/.
	Family string `json:"family"`
	// Slug is the page basename under the family directory.
	Slug string `json:"slug"`
	// Title is the human-readable page title.
	Title string `json:"title"`
	// Summary is a one-line statement of what the check enforces.
	Summary string `json:"summary"`
	// Fields documents the check's configuration keys, if any. The rules
	// command normalizes a nil slice to [] so consumers never see null.
	Fields []Field `json:"fields"`
	// ConfigExample is a complete katalyst.yaml snippet (YAML, no fence)
	// showing the check in a collection.
	ConfigExample string `json:"config_example"`
}

// Family identifies the three rule families and their intro copy. Order is
// significant: it fixes the section ordering in generated output.
type Family struct {
	ID    string
	Title string
	Intro string
}

// Families returns the rule families in display order.
func Families() []Family {
	return []Family{
		{
			ID:    "objects",
			Title: "Object Rules",
			Intro: "Object rules validate structured frontmatter fields using schema-backed checks.",
		},
		{
			ID:    "markdown",
			Title: "Markdown Rules",
			Intro: "Markdown rules validate relationships between frontmatter metadata and markdown body content.",
		},
		{
			ID:    "filesystem",
			Title: "Filesystem Rules",
			Intro: "Filesystem rules validate filename and path conventions for markdown items.",
		},
	}
}

// Descriptors returns every check kind in display order. The order is
// authored, not sorted, so generated output is deterministic.
func Descriptors() []Descriptor {
	return []Descriptor{
		// --- object family ---
		{
			Kind:    config.CheckObject,
			Family:  "objects",
			Slug:    "object",
			Title:   "Object Validation",
			Summary: "Validate frontmatter metadata against a named JSON Schema from `schemas:`.",
			Fields: []Field{
				{Name: "schema", Required: true, Desc: "Name of an entry in `schemas:`."},
			},
			ConfigExample: `schemas:
  book: ./schemas/book.json
collections:
  notes:
    path: notes
    checks:
      - kind: object
        schema: book`,
		},
		{
			Kind:    config.CheckObjectRequiredField,
			Family:  "objects",
			Slug:    "required-field",
			Title:   "Required Field",
			Summary: "Require that a frontmatter field exists.",
			Fields: []Field{
				{Name: "field", Required: true, Desc: "Frontmatter key that must be present."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_required_field
        field: year`,
		},
		{
			Kind:    config.CheckObjectFieldType,
			Family:  "objects",
			Slug:    "field-type",
			Title:   "Field Type",
			Summary: "Require that a frontmatter field has a specific type.",
			Fields: []Field{
				{Name: "field", Required: true, Desc: "Frontmatter key to check."},
				{Name: "type", Required: true, Desc: "One of `string`, `boolean`, `array`, `object`, `number`, `integer`."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_field_type
        field: year
        type: integer`,
		},
		{
			Kind:    config.CheckObjectFieldEnum,
			Family:  "objects",
			Slug:    "field-enum",
			Title:   "Field Enum",
			Summary: "Require that a field is one of a fixed set of values.",
			Fields: []Field{
				{Name: "field", Required: true, Desc: "Frontmatter key to check."},
				{Name: "values", Required: true, Desc: "Allowed values."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_field_enum
        field: status
        values: [draft, published, archived]`,
		},
		{
			Kind:    config.CheckObjectNumberRange,
			Family:  "objects",
			Slug:    "number-range",
			Title:   "Number Range",
			Summary: "Constrain a numeric field to a minimum and/or maximum value.",
			Fields: []Field{
				{Name: "field", Required: true, Desc: "Frontmatter key to check."},
				{Name: "min", Required: false, Desc: "Inclusive lower bound. At least one of `min`/`max` is required."},
				{Name: "max", Required: false, Desc: "Inclusive upper bound. At least one of `min`/`max` is required."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_number_range
        field: year
        min: 1900
        max: 2100`,
		},
		{
			Kind:    config.CheckObjectStringLength,
			Family:  "objects",
			Slug:    "string-length",
			Title:   "String Length",
			Summary: "Constrain the minimum and/or maximum length of a string field.",
			Fields: []Field{
				{Name: "field", Required: true, Desc: "Frontmatter key to check."},
				{Name: "min_length", Required: false, Desc: "Minimum length. At least one of `min_length`/`max_length` is required."},
				{Name: "max_length", Required: false, Desc: "Maximum length. At least one of `min_length`/`max_length` is required."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: object_string_length
        field: title
        min_length: 3
        max_length: 120`,
		},
		// --- markdown family ---
		{
			Kind:    config.CheckMarkdownTitleMatchesH1,
			Family:  "markdown",
			Slug:    "title-matches-h1",
			Title:   "Title Matches H1",
			Summary: "Require a frontmatter field to match the first H1 heading in the body.",
			Fields: []Field{
				{Name: "field", Required: false, Default: "title", Desc: "Frontmatter key compared to the first H1."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_title_matches_h1
        field: title`,
		},
		{
			Kind:    config.CheckMarkdownRequiresH1,
			Family:  "markdown",
			Slug:    "requires-h1",
			Title:   "Requires H1",
			Summary: "Require at least one H1 heading in the markdown body.",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1`,
		},
		{
			Kind:    config.CheckMarkdownSingleH1,
			Family:  "markdown",
			Slug:    "single-h1",
			Title:   "Single H1",
			Summary: "Require that the markdown body contains at most one H1 heading.",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_single_h1`,
		},
		{
			Kind:    config.CheckMarkdownNoHeadingLevelJumps,
			Family:  "markdown",
			Slug:    "no-heading-level-jumps",
			Title:   "No Heading Level Jumps",
			Summary: "Disallow jumps larger than one heading level (for example `H1 -> H3`).",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_no_heading_level_jumps`,
		},
		{
			Kind:    config.CheckMarkdownRequiredSection,
			Family:  "markdown",
			Slug:    "required-section",
			Title:   "Required Section",
			Summary: "Require that a heading with specific text exists somewhere in the body.",
			Fields: []Field{
				{Name: "heading", Required: true, Desc: "Heading text that must appear."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_required_section
        heading: Summary`,
		},
		{
			Kind:    config.CheckMarkdownCodeFenceHasLanguage,
			Family:  "markdown",
			Slug:    "code-fence-language-required",
			Title:   "Code Fence Language Required",
			Summary: "Require that opening fenced code blocks include a language tag.",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_code_fence_language_required`,
		},
		// --- filesystem family ---
		{
			Kind:    config.CheckFilesystemFilenameMatchesSlug,
			Family:  "filesystem",
			Slug:    "filename-matches-slug",
			Title:   "Filename Matches Slug",
			Summary: "Require a frontmatter field to match the markdown file basename.",
			Fields: []Field{
				{Name: "field", Required: false, Default: "slug", Desc: "Frontmatter key compared to the basename."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_filename_matches_slug
        field: slug`,
		},
		{
			Kind:    config.CheckFilesystemExtensionIn,
			Family:  "filesystem",
			Slug:    "extension-in",
			Title:   "Extension In",
			Summary: "Allow only specific file extensions.",
			Fields: []Field{
				{Name: "values", Required: true, Desc: "Allowed extensions, including the leading dot."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    pattern: "*"
    checks:
      - kind: filesystem_extension_in
        values: [.md, .markdown]`,
		},
		{
			Kind:    config.CheckFilesystemFilenameKebabCase,
			Family:  "filesystem",
			Slug:    "filename-kebab-case",
			Title:   "Filename Kebab Case",
			Summary: "Require lowercase kebab-case filenames (without extension).",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_filename_kebab_case`,
		},
		{
			Kind:    config.CheckFilesystemNoSpacesInPath,
			Family:  "filesystem",
			Slug:    "no-spaces-in-path",
			Title:   "No Spaces In Path",
			Summary: "Disallow spaces anywhere in the file path.",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_no_spaces_in_path`,
		},
		{
			Kind:    config.CheckFilesystemParentDirIn,
			Family:  "filesystem",
			Slug:    "parent-dir-in",
			Title:   "Parent Directory In",
			Summary: "Require that the file's parent directory name is in an allowed set.",
			Fields: []Field{
				{Name: "values", Required: true, Desc: "Allowed parent directory names."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_parent_dir_in
        values: [books, people]`,
		},
		{
			Kind:    config.CheckFilesystemFilenamePrefix,
			Family:  "filesystem",
			Slug:    "filename-prefix",
			Title:   "Filename Prefix",
			Summary: "Require that the filename starts with a specific prefix.",
			Fields: []Field{
				{Name: "value", Required: true, Desc: "Required filename prefix."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_filename_prefix
        value: book-`,
		},
	}
}
