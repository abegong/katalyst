+++
title = "Item views"
weight = 35
bookCollapseSection = true
+++

# Item views

Item views are the data surfaces Katalyst exposes for each item. Checks,
inspectors, and `fix` do not all need the same representation: one check may
read structured metadata, another may scan body text, and another may inspect
the item's path. Naming those surfaces keeps the reference precise without
turning every representation into a codec.

Today, Katalyst exposes four item views:

| View | Meaning |
|---|---|
| [Markdown body text]({{< relref "markdown-body-text.md" >}}) | A parsed markdown document with optional frontmatter metadata, body bytes, source format, and source-line lookup. |
| [Plain text]({{< relref "plain-text.md" >}}) | Body content read as raw text, independent of markdown structure. |
| [Structured object]({{< relref "structured-object.md" >}}) | Metadata normalized to a `map[string]any`, used by object and schema-backed checks. |
| [File metadata]({{< relref "file-metadata.md" >}}) | Filename, extension, parent directory, path depth, and other attributes derived from the item's reference. |

Only Markdown body text is backed by a dedicated codec package today. The other
views are projections over parsed item data or derived item references.

