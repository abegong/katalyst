// Package examples is the registry of worked examples: small input corpora
// paired with a `katalyst` command. Each example is run by both a test (which
// snapshots its rendered output, gating behavior) and cmd/gendocs (which renders
// it into the docs). Because the documentation is produced from the same command
// invocations the tests gate, the published examples cannot drift from real
// behavior. See issue #30.
package examples

// File is one file in an example's input corpus, at a project-relative path.
type File struct {
	Path    string
	Content string
}

// Example is a worked example: an input corpus plus a `katalyst` command.
// Running it (see Run) yields deterministic output used both to gate behavior
// and to generate documentation.
type Example struct {
	// ID is the slug shared by the generated catalog page, the generated output
	// snippet (docs/generated/examples/<ID>.txt), the {{< katalyst-example >}}
	// shortcode argument, and the golden fixture (testdata/<ID>.md).
	ID string
	// Title is the catalog page title. Sentence case, to satisfy the docs'
	// own object_sentence_case check when the page is validated by katalyst.
	Title string
	// Summary is a one-line description for the catalog page and its index.
	Summary string
	// Doc is a short narrative shown on the catalog page, explaining what the
	// example demonstrates.
	Doc string
	// Files is the input corpus, including the .katalyst/ project files.
	Files []File
	// Args is the command line after "katalyst" (e.g. {"check", "notes/dune"}).
	Args []string
	// ResultFiles are corpus files whose post-run content the example shows
	// (the "after" of a fix). Empty for read-only commands.
	ResultFiles []string
	// Weight orders the example within the generated catalog section.
	Weight int
}

// bookSchema requires title+year; the object check examples bind it.
const bookSchema = `type: object
required: [title, year]
properties:
  title: { type: string }
  year:  { type: integer }
`

// notesStorage declares a single `notes` collection bound to the book schema.
const notesStorage = `type: filesystem
root: .
collections:
  notes:
    path: notes
    schema: book
`

// wikiBookSchema requires title+author+status with a status enum; the
// collection-layer inspect example binds it so the project loads.
const wikiBookSchema = `type: object
required: [title, author, status]
properties:
  title:  { type: string }
  author: { type: string }
  status: { enum: [read, reading, to-read] }
`

// wikiStorage binds a `books` collection over the wiki/ tree.
const wikiStorage = `type: filesystem
root: .
collections:
  books:
    path: wiki
    schema: book
`

// wikiCorpus is the shared book corpus the two inspect examples profile: four
// kebab-named books with title/author/status and a Review section (one cluster),
// plus a single spaced, author-less outlier.
var wikiCorpus = []File{
	{Path: "wiki/dune.md", Content: "---\ntitle: Dune\nauthor: Frank Herbert\nstatus: read\n---\n# Dune\n\n## Review\nA landmark of the genre.\n"},
	{Path: "wiki/neuromancer.md", Content: "---\ntitle: Neuromancer\nauthor: William Gibson\nstatus: reading\n---\n# Neuromancer\n\n## Review\n"},
	{Path: "wiki/foundation.md", Content: "---\ntitle: Foundation\nauthor: Isaac Asimov\nstatus: to-read\n---\n# Foundation\n\n## Review\n"},
	{Path: "wiki/snow-crash.md", Content: "---\ntitle: Snow Crash\nauthor: Neal Stephenson\nstatus: read\n---\n# Snow Crash\n\n## Review\n"},
	{Path: "wiki/Dune Messiah.md", Content: "---\ntitle: Dune Messiah\nstatus: read\n---\n# Dune Messiah\n"},
}

// withWikiProject prepends the .katalyst project files to the wiki corpus.
func withWikiProject() []File {
	out := []File{
		{Path: ".katalyst/schemas/book.yaml", Content: wikiBookSchema},
		{Path: ".katalyst/storage/local.yaml", Content: wikiStorage},
	}
	return append(out, wikiCorpus...)
}

// All returns the worked-example registry, in catalog order.
func All() []Example {
	return []Example{
		{
			ID:      "check-valid-item",
			Title:   "Check a valid item",
			Summary: "An item that satisfies its schema passes and reports OK.",
			Doc:     "The `notes` collection binds the `book` schema, which requires `title` and an integer `year`. This item satisfies both, so `check` exits 0 and prints OK.",
			Weight:  10,
			Files: []File{
				{Path: ".katalyst/schemas/book.yaml", Content: bookSchema},
				{Path: ".katalyst/storage/local.yaml", Content: notesStorage},
				{Path: "notes/dune.md", Content: "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n"},
			},
			Args: []string{"check", "notes/dune"},
		},
		{
			ID:      "check-type-error",
			Title:   "Report a type error with a pointer",
			Summary: "A field of the wrong type fails with a JSON-pointer diagnostic and exit 1.",
			Doc:     "Here `year` is a string, not an integer. `check` fails the item, points at the offending field with a JSON pointer (`/year`) and a `path:line` prefix, and exits 1.",
			Weight:  20,
			Files: []File{
				{Path: ".katalyst/schemas/book.yaml", Content: bookSchema},
				{Path: ".katalyst/storage/local.yaml", Content: notesStorage},
				{Path: "notes/dune.md", Content: "---\ntitle: Dune\nyear: \"not a number\"\n---\n# Dune\n"},
			},
			Args: []string{"check", "notes/dune"},
		},
		{
			ID:      "check-title-h1-mismatch",
			Title:   "Catch a title that does not match its H1",
			Summary: "A markdown check fails when the frontmatter title and first H1 disagree.",
			Doc:     "The `markdown_title_matches_h1` check ties a frontmatter field to the document's first H1. When they disagree, `check` reports the mismatch and exits 1.",
			Weight:  30,
			Files: []File{
				{Path: ".katalyst/storage/local.yaml", Content: "type: filesystem\nroot: .\ncollections:\n  notes:\n    path: notes\n    checks:\n      - kind: markdown_title_matches_h1\n        field: title\n"},
				{Path: "notes/dune.md", Content: "---\ntitle: Dune\n---\n# Children of Dune\n"},
			},
			Args: []string{"check", "notes/dune"},
		},
		{
			ID:          "fix-normalize-frontmatter",
			Title:       "Normalize frontmatter without touching the body",
			Summary:     "fix canonicalizes frontmatter (sorted keys) and leaves the body verbatim.",
			Doc:         "`fix` rewrites frontmatter into a canonical form (here, sorting the keys) while leaving the markdown body byte-for-byte unchanged. It is idempotent and never injects missing keys.",
			Weight:      40,
			ResultFiles: []string{"notes/doc.md"},
			Files: []File{
				{Path: ".katalyst/storage/local.yaml", Content: "type: filesystem\nroot: .\ncollections:\n  notes:\n    path: notes\n    checks:\n      - kind: markdown_requires_h1\n"},
				{Path: "notes/doc.md", Content: "---\nzebra: 1\napple: 2\n---\n# Body\nverbatim\n"},
			},
			Args: []string{"fix", "notes/doc"},
		},
		{
			ID:          "fix-text-forbids",
			Title:       "Rewrite only the matched text",
			Summary:     "A text_forbids fix template rewrites the violation and nothing else.",
			Doc:         "The `text_forbids` check forbids a trailing period on the first line; its `fix` template strips it. Only the matched text changes; the later `keep this.` line is untouched.",
			Weight:      50,
			ResultFiles: []string{"notes/doc.md"},
			Files: []File{
				{Path: ".katalyst/storage/local.yaml", Content: "type: filesystem\nroot: .\ncollections:\n  notes:\n    path: notes\n    checks:\n      - kind: text_forbids\n        target: first-line\n        pattern: '\\.(\\s*)$'\n        fix: '$1'\n"},
				{Path: "notes/doc.md", Content: "---\nt: 1\n---\n# Title.\nkeep this.\n"},
			},
			Args: []string{"fix", "notes/doc"},
		},
		{
			ID:      "inspect-source-shape",
			Title:   "Cluster a raw directory by shape",
			Summary: "The raw-source document_shape inspector groups files into candidate collections.",
			Doc:     "Pointed at a bare directory (no project), `inspect` runs the raw-source inspectors. `document_shape` clusters files by a composite fingerprint, so a shared convention shows up as one class and the stragglers as outliers.",
			Weight:  60,
			Files:   wikiCorpus,
			Args:    []string{"inspect", "./wiki", "--inspector", "document_shape"},
		},
		{
			ID:      "inspect-collection-fields",
			Title:   "Profile a collection's fields",
			Summary: "The collection-layer object_fields inspector is a data dictionary over frontmatter.",
			Doc:     "Once a collection is configured, `inspect <name>` runs the collection inspectors. `object_fields` reports, per field, how often it appears over `n`, its observed types, and its value cardinality: the evidence behind `required`/optional and `enum` decisions.",
			Weight:  70,
			Files:   withWikiProject(),
			Args:    []string{"inspect", "books", "--inspector", "object_fields", "-v"},
		},
	}
}
