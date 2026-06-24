// Package markdownbodytext parses and encodes markdown body text with optional
// structured frontmatter.
//
// A frontmatter block is the metadata document at the very top of a markdown
// file. Katalyst recognizes the three formats used by Hugo, Obsidian, and
// Jekyll, detected by the opening fence:
//
//	---            +++            {
//	title: Dune    title = "Dune"   "title": "Dune",
//	year: 1965     year = 1965      "year": 1965
//	---            +++            }
//	# Body         # Body         # Body
//	(YAML)         (TOML)         (JSON)
//
// The parsed representation is format-agnostic: regardless of the source
// format, Parse returns a Document whose Meta is a map[string]any, so checks
// and inspectors never need to know which format a file used. The detected
// format is recorded in Document.Format so Encode can round-trip a file back
// into its own syntax rather than rewriting, say, TOML as YAML.
//
// Source-line tracking (Document.Lines) is currently full for YAML only; for
// TOML and JSON the map is empty. Error messages degrade gracefully when a line
// is unknown. Richer line tracking for the other formats is a planned follow-up.
package markdownbodytext
