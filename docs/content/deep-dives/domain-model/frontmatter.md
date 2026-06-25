+++
title = "Frontmatter"
weight = 60
+++

# Frontmatter

How Katalyst parses a markdown file's frontmatter and the in-memory document
that produces. The codec lives in `internal/codec/markdownbodytext`; it turns a
markdown file into structured metadata plus body bytes that checks, inspectors,
and [`fix`]({{< relref "fix.md" >}}) can share.

## The markdown document

The unit of work is a file on disk with two optional regions:

- A **frontmatter** block at the very top of the file, in one of three formats
  detected by the opening fence:

  | Format | Fence | Example openers |
  |--------|-------|-----------------|
  | YAML   | `---` | Jekyll, Obsidian, Hugo |
  | TOML   | `+++` | Hugo, Obsidian, Jekyll |
  | JSON   | `{` ... `}` | Hugo |

  These are the three formats Hugo, Obsidian, and Jekyll emit. Whatever the
  source format, the parsed `Meta` is a plain `map[string]any`, so checks and
  inspectors never branch on format. `Document.Format` records the detected
  syntax so `fix` can re-emit a file in its own format rather than rewriting,
  say, TOML as YAML.
- A **body**, everything after the closing fence.

A document may have no frontmatter. Structured-object checks that need metadata
cannot run against it, but body-text checks can still evaluate the body.

When parsed, a markdown document becomes a `markdownbodytext.Document`:

| Field            | Meaning |
|------------------|---------|
| `HasFrontmatter` | Did the file open with a recognized fence? |
| `Format`         | Detected syntax: `KindYAML`, `KindTOML`, or `KindJSON` |
| `Meta`           | Parsed frontmatter, normalized to `map[string]any` |
| `Body`           | Bytes after the closing fence |
| `Lines`          | JSON-pointer-path to 1-indexed source line |

The `Lines` index is what makes error messages locatable. It accounts for the
opening fence offset, so `Lines["/title"] = 2` means the `title` key is on line
2 of the original file.

**Line tracking is full for YAML only.** For TOML and JSON, `Lines` is empty
today; checks degrade gracefully (they emit the error without a line number).
Richer line tracking for the other formats is a planned follow-up.

## Invariants

1. **Checks and inspectors read one metadata shape.** YAML, TOML, and JSON all
   parse to `map[string]any`.
2. **Line numbers are file-relative and 1-indexed.** The opening fence is line
   1, so the first key is typically line 2. (Populated for YAML today; see the
   line-tracking note above.)
3. **Format detection is preserved for writers.** Readers expose normalized
   metadata, but retain the original frontmatter syntax so a writer can emit
   the same format.

## See also

- [Fix]({{< relref "fix.md" >}}) for the canonical rewrite policy that consumes
  parsed documents.
- `go doc ./internal/codec/markdownbodytext` for the code-level codec contract.
