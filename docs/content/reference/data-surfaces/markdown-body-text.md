+++
title = "Markdown body text"
weight = 10
+++

# Markdown body text

Markdown body text is the data surface produced from a markdown-like file with
optional structured frontmatter. The codec lives in
`internal/codec/markdownbodytext`; it turns bytes on disk into structured
metadata plus body bytes that checks, inspectors, and
[`fix`]({{< relref "../../deep-dives/domain-model/fix.md" >}}) can share.

## Terms

| Term | Meaning |
|---|---|
| **Markdown body text** | The parsed markdown file-form exposed as a data surface: optional structured frontmatter, body bytes, source format, and source-line lookup. |
| **Frontmatter** | The structured metadata block at the top of a markdown file, in YAML, TOML, or JSON. |
| **Body** | Everything after the closing frontmatter fence. If there is no frontmatter, the whole file is the body. |
| **Document** | The in-memory representation returned by `markdownbodytext.Parse`. |
| **Metadata** | The parsed frontmatter shape, normalized to `map[string]any`. |
| **Source line map** | A JSON-pointer-path to 1-indexed source line lookup used for locatable violations. |

## Model

The unit of work is a file on disk with two possible regions:

| Region | Meaning |
|---|---|
| Frontmatter | An optional structured block at the very top of the file. |
| Body | Everything after the closing frontmatter fence, or the whole file when no frontmatter is present. |

Katalyst recognizes the three frontmatter formats emitted by Hugo, Obsidian,
and Jekyll:

| Format | Fence | Example sources |
|---|---|---|
| YAML | `---` | Jekyll, Obsidian, Hugo |
| TOML | `+++` | Hugo, Obsidian, Jekyll |
| JSON | `{` ... `}` | Hugo |

Whatever the source format, parsed metadata has the same shape:
`map[string]any`. Checks and inspectors can read fields without branching on
YAML, TOML, or JSON. `Document.Format` records the detected syntax so writers
can re-emit a file in its own format rather than rewriting TOML as YAML.

When parsed, a markdown document becomes a `markdownbodytext.Document`:

| Field | Meaning |
|---|---|
| `HasFrontmatter` | Did the file open with a recognized frontmatter fence? |
| `Format` | Detected syntax: `KindYAML`, `KindTOML`, or `KindJSON`. |
| `Meta` | Parsed frontmatter, normalized to `map[string]any`. |
| `Body` | Bytes after the closing fence, or the entire file when there is no frontmatter. |
| `BodyLine` | The 1-indexed source line where the body begins. |
| `Lines` | JSON-pointer-path to 1-indexed source line. |
| `Frontmatter` | Raw frontmatter bytes, used by text search and diagnostics. |

The `Lines` index is what makes structured-object violations locatable. It
accounts for the opening fence offset, so `Lines["/title"] = 2` means the
`title` key is on line 2 of the original file.

Line tracking is full for YAML only. For TOML and JSON, `Lines` is empty today;
checks degrade gracefully by emitting the violation without a line number.

## Invariants

1. **Readers see one metadata shape.** YAML, TOML, and JSON all parse to
   `map[string]any`.
2. **Body bytes remain the body view.** The body is available to markdown and
   plain-text checks without requiring callers to understand the frontmatter
   syntax.
3. **Format detection is preserved for writers.** Readers expose normalized
   metadata but retain the original syntax so `fix` can emit the same format.
4. **Line numbers are file-relative and 1-indexed.** The opening fence is line
   1, so the first metadata key is typically line 2 when line data is available.

## See also

- [Markdown body text check types]({{< relref "../check-types/markdown-body-text/_index.md" >}})
- [Plain text]({{< relref "plain-text.md" >}}), the raw body-text surface over
  the same body bytes.
- [Fix]({{< relref "../../deep-dives/domain-model/fix.md" >}}), which consumes
  parsed documents when it rewrites frontmatter.
- `go doc ./internal/codec/markdownbodytext` for the code-level codec contract.
