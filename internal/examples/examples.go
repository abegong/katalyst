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
	// ID is the slug shared by the two generated snippets
	// (docs/generated/examples/<ID>.txt for {{< katalyst-example >}} and
	// <ID>.full.md for {{< katalyst-example-full >}}), the shortcode arguments
	// that embed them into prose, and the golden fixture (testdata/<ID>.md).
	// Each example is embedded into the reference, how-to, or deep-dive page
	// that owns the feature it demonstrates; there is no standalone catalog.
	ID string
	// Title is a short, sentence-case label for the example. It is not rendered
	// into the embedded snippet (the host page supplies its own heading); it
	// documents the example here and labels its test subtest.
	Title string
	// Summary is a one-line description of what the example demonstrates.
	Summary string
	// Doc is the short narrative rendered at the top of the embedded worked
	// example, explaining what it demonstrates.
	Doc string
	// Files is the input corpus, including the .katalyst/ project files.
	Files []File
	// Args is the command line after "katalyst" (e.g. {"check", "notes/dune"}).
	Args []string
	// ResultFiles are corpus files whose post-run content the example shows
	// (the "after" of a fix). Empty for read-only commands.
	ResultFiles []string
	// Weight is a stable ordering hint for the registry. It no longer drives a
	// catalog page; each example is embedded into the page that owns its feature.
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

// postsRulesStorage is the `posts` collection from the configure-rules how-to:
// the three structural/markdown/filesystem checks that guide attaches.
const postsRulesStorage = `type: filesystem
root: .
collections:
  posts:
    path: content/posts
    checks:
      - kind: markdown_requires_h1
      - kind: markdown_title_matches_h1
        field: title
      - kind: filesystem_name_case
        style: kebab
`

// bookConstrainedSchema is the YAML form of the add-a-schema how-to's book
// schema: a JSON Schema (draft 2020-12) written in YAML, which the default
// schema discovery picks up with no extra config. It mirrors the page's
// required keys, types, and ranges.
const bookConstrainedSchema = `$schema: https://json-schema.org/draft/2020-12/schema
title: book
type: object
required: [title, year]
properties:
  title: { type: string, minLength: 1 }
  year:  { type: integer, minimum: 0 }
`

// booksAtNotesStorage binds the `book` schema to a `books` collection at
// notes/books, matching the add-a-schema how-to.
const booksAtNotesStorage = `type: filesystem
root: .
collections:
  books:
    path: notes/books
    schema: book
`

// ciStorage is the small project the validate-in-ci how-to gates: a `notes`
// collection that only requires an H1, so the failing item fails on structure
// alone and the canonical-frontmatter gate is easy to read.
const ciStorage = `type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
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
		{
			ID:      "check-schema-missing-field",
			Title:   "Check items against a bound schema",
			Summary: "A collection bound to the book schema passes a complete item and fails one missing a required field.",
			Doc:     "The `books` collection binds the `book` schema (`title` plus an integer `year`). `dune.md` satisfies the schema and reports OK; `foundation.md` omits `year`, so `check` reports the missing required property and exits 1.",
			Weight:  80,
			Files: []File{
				{Path: ".katalyst/schemas/book.yaml", Content: bookConstrainedSchema},
				{Path: ".katalyst/storage/local.yaml", Content: booksAtNotesStorage},
				{Path: "notes/books/dune.md", Content: "---\ntitle: Dune\nyear: 1965\n---\n# Dune\n"},
				{Path: "notes/books/foundation.md", Content: "---\ntitle: Foundation\n---\n# Foundation\n"},
			},
			Args: []string{"check", "books"},
		},
		{
			ID:      "check-collection-rules",
			Title:   "Check a collection's attached rules",
			Summary: "A collection with markdown and filesystem checks passes a conforming item and flags a mis-named, mismatched one.",
			Doc:     "The `posts` collection attaches three checks: an H1 must exist, the frontmatter `title` must match that H1, and the filename must be kebab-case. `hello-world.md` satisfies all three; `Bad_Title.md` violates the casing rule and the title/H1 match, so `check` reports both and exits 1.",
			Weight:  90,
			Files: []File{
				{Path: ".katalyst/storage/local.yaml", Content: postsRulesStorage},
				{Path: "content/posts/hello-world.md", Content: "---\ntitle: Hello world\n---\n# Hello world\n"},
				{Path: "content/posts/Bad_Title.md", Content: "---\ntitle: Bad title\n---\n# A different heading\n"},
			},
			Args: []string{"check", "posts"},
		},
		{
			ID:      "ci-check-fails",
			Title:   "Gate CI on validation",
			Summary: "A whole-project check exits 1 when any item has a violation, the signal CI gates on.",
			Doc:     "With no target, `check` validates every collection. One item is missing its H1, so the run reports the violation and exits 1; the `exit status 1` line is what fails the CI step.",
			Weight:  100,
			Files: []File{
				{Path: ".katalyst/storage/local.yaml", Content: ciStorage},
				{Path: "notes/intro.md", Content: "---\ntitle: Intro\n---\n# Intro\n"},
				{Path: "notes/draft.md", Content: "---\ntitle: Draft\n---\nNo heading here.\n"},
			},
			Args: []string{"check"},
		},
		{
			ID:      "ci-fix-check",
			Title:   "Gate CI on canonical frontmatter",
			Summary: "fix --check writes nothing and exits 1 when frontmatter is not canonical.",
			Doc:     "`fix --check` is the read-only formatting gate: it lists items whose frontmatter is not canonical and exits 1, without modifying any file. Here `messy.md` has unsorted keys, so it is reported; `tidy.md` is already canonical and passes.",
			Weight:  110,
			Files: []File{
				{Path: ".katalyst/storage/local.yaml", Content: ciStorage},
				{Path: "notes/tidy.md", Content: "---\ntitle: Tidy\n---\n# Tidy\n"},
				{Path: "notes/messy.md", Content: "---\ntitle: Messy\nauthor: Ada\n---\n# Messy\n"},
			},
			Args: []string{"fix", "--check"},
		},
	}
}
