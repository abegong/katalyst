+++
title = "Plain text"
weight = 20
+++

# Plain text

Plain text is the body content read as raw text. It ignores markdown structure
and treats the selected span as text to match with regular expressions or
literal denylist entries.

## Terms

| Term | Meaning |
|---|---|
| **Plain text** | The body surface interpreted as raw text. |
| **Body** | The content being searched. For markdown files, this excludes frontmatter. |
| **Span** | The slice of body text a text check evaluates: the whole body, each line, the first line, or matched lines. |
| **Target** | The configured span selector for a text check. |

## Model

Plain-text checks run against body text, not structured metadata and not
markdown syntax trees. For markdown items, the body comes from the
[Markdown body text]({{< relref "markdown-body-text.md" >}}) view after
frontmatter has been separated. For plain-text items, the whole item body is the
text surface.

This view backs the `text_requires`, `text_forbids`, and `text_denylist` check
types. Those checks answer content questions such as "must contain this
pattern", "must not contain this pattern", or "must not contain any of these
literal strings."

## Invariants

1. **Frontmatter is outside the body.** Text checks over markdown files do not
   match metadata unless a check explicitly inspects raw frontmatter elsewhere.
2. **Markdown structure is not parsed.** Headings, links, and code fences are
   just text to this view.
3. **The configured span controls matching.** A check may evaluate the whole
   body or smaller slices such as individual lines.

## See also

- [Plain text check types]({{< relref "../check-types/plain-text/_index.md" >}})
- [Markdown body text]({{< relref "markdown-body-text.md" >}})
- [Configs]({{< relref "../configs/_index.md#text-rules" >}}) for text
  rule configuration.
