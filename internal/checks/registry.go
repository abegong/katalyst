package checks

import "github.com/abegong/katalyst/internal/config"

// This file is the single source of truth for the check-type reference
// documentation. Every check type dispatched in config.normalizeCheck must
// have a matching Descriptor here; registry_test.go enforces that parity, and
// cmd/gendocs renders docs/reference/check-types/ from these descriptors. A new
// check type cannot ship undocumented.

// Field describes one configuration key accepted by a check type. The json tags
// are the wire contract for `katalyst check-types list --json`; keep them
// snake_case (matching the config keys they describe) even if the Go field
// names change.
type Field struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
	Desc     string `json:"desc"`
}

// Descriptor is the machine-readable record for one check type. Its json tags
// are the wire contract for `katalyst check-types list --json`; see Field.
type Descriptor struct {
	// CheckType is the value used as `kind:` in a collection's checks.
	CheckType config.CheckType `json:"check_type"`
	// Family groups the check type on the docs site: "objects", "markdown", or
	// "filesystem". It is also the subdirectory under reference/check-types/.
	Family string `json:"family"`
	// Slug is the page basename under the family directory.
	Slug string `json:"slug"`
	// Title is the human-readable page title.
	Title string `json:"title"`
	// Summary is a one-line statement of what the check type enforces.
	Summary string `json:"summary"`
	// Fields documents the check type's configuration keys, if any. The
	// check-types command normalizes a nil slice to [] so consumers never see null.
	Fields []Field `json:"fields"`
	// ConfigExample is a complete config snippet (YAML, no fence)
	// showing the check in a collection.
	ConfigExample string `json:"config_example"`
	// Scope is "collection" for checks that run once per collection over all
	// its items; empty means an ordinary per-item check.
	Scope string `json:"scope,omitempty"`
}

// Family identifies the three check-type families and their intro copy. Order
// is significant: it fixes the section ordering in generated output.
type Family struct {
	ID    string
	Title string
	Intro string
}

// Families returns the check-type families in display order.
func Families() []Family {
	return []Family{
		{
			ID:    "objects",
			Title: "Object Check Types",
			Intro: "Object check types validate structured frontmatter fields using schema-backed checks.",
		},
		{
			ID:    "markdown",
			Title: "Markdown Check Types",
			Intro: "Markdown check types validate relationships between frontmatter metadata and markdown body content.",
		},
		{
			ID:    "filesystem",
			Title: "Filesystem Check Types",
			Intro: "Filesystem check types validate filename and path conventions for markdown items.",
		},
		{
			ID:    "text",
			Title: "Text Check Types",
			Intro: "Text check types validate body content as raw text, independent of markdown structure. They apply to plain-text items as well as markdown bodies.",
		},
	}
}

// Descriptors returns every check type in display order. The order is
// authored, not sorted, so generated output is deterministic.
func Descriptors() []Descriptor {
	return []Descriptor{
		// --- object family ---
		{
			CheckType: config.CheckObject,
			Family:    "objects",
			Slug:      "object",
			Title:     "Object Validation",
			Summary:   "Validate frontmatter metadata against a named JSON Schema from `schemas:`.",
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
			CheckType: config.CheckObjectRequiredField,
			Family:    "objects",
			Slug:      "required-field",
			Title:     "Required Field",
			Summary:   "Require that a frontmatter field exists.",
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
			CheckType: config.CheckObjectFieldType,
			Family:    "objects",
			Slug:      "field-type",
			Title:     "Field Type",
			Summary:   "Require that a frontmatter field has a specific type.",
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
			CheckType: config.CheckObjectFieldEnum,
			Family:    "objects",
			Slug:      "field-enum",
			Title:     "Field Enum",
			Summary:   "Require that a field is one of a fixed set of values.",
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
			CheckType: config.CheckObjectNumberRange,
			Family:    "objects",
			Slug:      "number-range",
			Title:     "Number Range",
			Summary:   "Constrain a numeric field to a minimum and/or maximum value.",
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
			CheckType: config.CheckObjectStringLength,
			Family:    "objects",
			Slug:      "string-length",
			Title:     "String Length",
			Summary:   "Constrain the minimum and/or maximum length of a string field.",
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
			CheckType: config.CheckMarkdownTitleMatchesH1,
			Family:    "markdown",
			Slug:      "title-matches-h1",
			Title:     "Title Matches H1",
			Summary:   "Require a frontmatter field to match the first H1 heading in the body.",
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
			CheckType: config.CheckMarkdownRequiresH1,
			Family:    "markdown",
			Slug:      "requires-h1",
			Title:     "Requires H1",
			Summary:   "Require at least one H1 heading in the markdown body.",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1`,
		},
		{
			CheckType: config.CheckMarkdownSingleH1,
			Family:    "markdown",
			Slug:      "single-h1",
			Title:     "Single H1",
			Summary:   "Require that the markdown body contains at most one H1 heading.",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_single_h1`,
		},
		{
			CheckType: config.CheckMarkdownNoHeadingLevelJumps,
			Family:    "markdown",
			Slug:      "no-heading-level-jumps",
			Title:     "No Heading Level Jumps",
			Summary:   "Disallow jumps larger than one heading level (for example `H1 -> H3`).",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_no_heading_level_jumps`,
		},
		{
			CheckType: config.CheckMarkdownRequiredSection,
			Family:    "markdown",
			Slug:      "required-section",
			Title:     "Required Section",
			Summary:   "Require that a heading with specific text exists somewhere in the body.",
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
			CheckType: config.CheckMarkdownCodeFenceHasLanguage,
			Family:    "markdown",
			Slug:      "code-fence-language-required",
			Title:     "Code Fence Language Required",
			Summary:   "Require that opening fenced code blocks include a language tag.",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: markdown_code_fence_language_required`,
		},
		// --- filesystem family ---
		{
			CheckType: config.CheckFilesystemNameCase,
			Family:    "filesystem",
			Slug:      "name-case",
			Title:     "Name Case",
			Summary:   "Require a name (or path segments) to follow a case style.",
			Fields: []Field{
				{Name: "style", Required: true, Desc: "One of `kebab`, `snake`, `screaming-snake`, `camel`, `pascal`, `point`, `lower`."},
				{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, `parent-dir`, or `path-segments`."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_case
        style: kebab`,
		},
		{
			CheckType: config.CheckFilesystemNameMatchesField,
			Family:    "filesystem",
			Slug:      "name-matches-field",
			Title:     "Name Matches Field",
			Summary:   "Require a name to equal a frontmatter field, optionally slugified.",
			Fields: []Field{
				{Name: "field", Required: false, Default: "slug", Desc: "Frontmatter key compared to the name."},
				{Name: "transform", Required: false, Default: "none", Desc: "`none` or `slugify` (applied to the field value before comparison)."},
				{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, or `parent-dir`."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_matches_field
        field: slug`,
		},
		{
			CheckType: config.CheckFilesystemNameAffix,
			Family:    "filesystem",
			Slug:      "name-affix",
			Title:     "Name Affix",
			Summary:   "Require a name to start with a prefix and/or end with a suffix.",
			Fields: []Field{
				{Name: "prefix", Required: false, Desc: "Required name prefix (at least one of prefix/suffix)."},
				{Name: "suffix", Required: false, Desc: "Required name suffix (at least one of prefix/suffix)."},
				{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, or `parent-dir`."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_affix
        prefix: book-`,
		},
		{
			CheckType: config.CheckFilesystemPathCharset,
			Family:    "filesystem",
			Slug:      "path-charset",
			Title:     "Path Charset",
			Summary:   "Constrain the characters allowed in the item's path.",
			Fields: []Field{
				{Name: "deny", Required: false, Desc: "Forbidden substrings (e.g. a space). Use `deny` or `allow`, not both."},
				{Name: "allow", Required: false, Desc: "The only permitted characters; the path separator is always allowed."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_path_charset
        deny: [" "]`,
		},
		{
			CheckType: config.CheckFilesystemNameRegex,
			Family:    "filesystem",
			Slug:      "name-regex",
			Title:     "Name Regex",
			Summary:   "Require a name to match a regular expression (anchored).",
			Fields: []Field{
				{Name: "pattern", Required: true, Desc: "Regular expression; matched anchored (`^pattern$`)."},
				{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, `parent-dir`, or `path-segments`."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_regex
        pattern: '[0-9]{4}-[a-z-]+'`,
		},
		{
			CheckType: config.CheckFilesystemNameLength,
			Family:    "filesystem",
			Slug:      "name-length",
			Title:     "Name Length",
			Summary:   "Bound the character length of a name.",
			Fields: []Field{
				{Name: "min", Required: false, Desc: "Minimum length (at least one of min/max)."},
				{Name: "max", Required: false, Desc: "Maximum length (at least one of min/max)."},
				{Name: "target", Required: false, Default: "filename", Desc: "What to test: `filename`, `filename-ext`, `parent-dir`, or `path-segments`."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_name_length
        max: 80`,
		},
		{
			CheckType: config.CheckFilesystemPathDepth,
			Family:    "filesystem",
			Slug:      "path-depth",
			Title:     "Path Depth",
			Summary:   "Bound directory nesting relative to the collection root.",
			Fields: []Field{
				{Name: "min", Required: false, Desc: "Minimum depth (at least one of min/max)."},
				{Name: "max", Required: false, Desc: "Maximum depth; `0` means a flat collection (at least one of min/max)."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_path_depth
        max: 0`,
		},
		{
			CheckType: config.CheckFilesystemParentDirMatchesFld,
			Family:    "filesystem",
			Slug:      "parent-dir-matches-field",
			Title:     "Parent Directory Matches Field",
			Summary:   "Require the parent directory name to equal a frontmatter field.",
			Fields: []Field{
				{Name: "field", Required: true, Desc: "Frontmatter key compared to the parent directory name."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_parent_dir_matches_field
        field: category`,
		},
		{
			CheckType: config.CheckFilesystemReferencedFiles,
			Family:    "filesystem",
			Slug:      "referenced-files-exist",
			Title:     "Referenced Files Exist",
			Summary:   "Require path-valued frontmatter fields to resolve to real files.",
			Fields: []Field{
				{Name: "fields", Required: true, Desc: "Frontmatter keys holding a path (string) or list of paths, resolved relative to the item."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_referenced_files_exist
        fields: [cover, attachments]`,
		},
		{
			CheckType: config.CheckFilesystemUniqueFilename,
			Family:    "filesystem",
			Slug:      "unique-filename",
			Title:     "Unique Filename",
			Summary:   "Require that no two items in the collection share a basename.",
			Scope:     "collection",
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_unique_filename`,
		},
		{
			CheckType: config.CheckFilesystemUniqueField,
			Family:    "filesystem",
			Slug:      "unique-field",
			Title:     "Unique Field",
			Summary:   "Require that no two items share a value for a frontmatter field.",
			Scope:     "collection",
			Fields: []Field{
				{Name: "field", Required: true, Desc: "Frontmatter key whose value must be unique across the collection."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_unique_field
        field: slug`,
		},
		{
			CheckType: config.CheckFilesystemIndexFileRequired,
			Family:    "filesystem",
			Slug:      "index-file-required",
			Title:     "Index File Required",
			Summary:   "Require that every directory containing items has an index file.",
			Scope:     "collection",
			Fields: []Field{
				{Name: "name", Required: false, Default: "_index.md", Desc: "Index filename that must be present in each item directory."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: filesystem_index_file_required`,
		},
		{
			CheckType: config.CheckFilesystemExtensionIn,
			Family:    "filesystem",
			Slug:      "extension-in",
			Title:     "Extension In",
			Summary:   "Allow only specific file extensions.",
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
			CheckType: config.CheckFilesystemParentDirIn,
			Family:    "filesystem",
			Slug:      "parent-dir-in",
			Title:     "Parent Directory In",
			Summary:   "Require that the file's parent directory name is in an allowed set.",
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
		// --- text family ---
		{
			CheckType: config.CheckTextRequires,
			Family:    "text",
			Slug:      "requires",
			Title:     "Requires",
			Summary:   "Require a regular expression to appear in the body text.",
			Fields: []Field{
				{Name: "pattern", Required: true, Desc: "Go regular expression, matched unanchored (appears somewhere in the span)."},
				{Name: "target", Required: false, Default: "body", Desc: "Span selector: body, line, first-line, or matched-lines."},
				{Name: "select", Required: false, Desc: "Line-filter regex; required for and only valid with target matched-lines."},
				{Name: "match", Required: false, Default: "any", Desc: "For multi-span targets: any (at least one span matches) or all (every span matches)."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: text_requires
        pattern: Sources`,
		},
		{
			CheckType: config.CheckTextForbids,
			Family:    "text",
			Slug:      "forbids",
			Title:     "Forbids",
			Summary:   "Forbid a regular expression from appearing in the body text.",
			Fields: []Field{
				{Name: "pattern", Required: true, Desc: "Go regular expression, matched unanchored."},
				{Name: "target", Required: false, Default: "body", Desc: "Span selector: body, line, first-line, or matched-lines."},
				{Name: "select", Required: false, Desc: "Line-filter regex; required for and only valid with target matched-lines."},
				{Name: "fix", Required: false, Desc: "Optional replacement template (regexp capture syntax) applied to the matched text by the fix command."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: text_forbids
        target: line
        pattern: '\bTODO\b'`,
		},
		{
			CheckType: config.CheckTextDenylist,
			Family:    "text",
			Slug:      "denylist",
			Title:     "Denylist",
			Summary:   "Forbid any of a list of literal substrings in the body text.",
			Fields: []Field{
				{Name: "values", Required: true, Desc: "Literal substrings to forbid; regex metacharacters are inert."},
				{Name: "target", Required: false, Default: "body", Desc: "Span selector: body, line, first-line, or matched-lines."},
				{Name: "select", Required: false, Desc: "Line-filter regex; required for and only valid with target matched-lines."},
			},
			ConfigExample: `collections:
  notes:
    path: notes
    checks:
      - kind: text_denylist
        values: [TODO, FIXME, XXX]`,
		},
	}
}
