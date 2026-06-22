+++
title = "Profile an existing wiki by hand"
weight = 5
+++

# Profile an existing wiki by hand

You have a directory of markdown — a vault, a docs tree, a knowledge base —
and you want a Katalyst schema for it. Rather than guess the conventions,
`inspect` measures them. This guide turns an existing corpus into a draft
schema **by reading the evidence yourself**. To hand that judgment to an agent
instead, see [Profile an existing wiki with an
agent]({{< relref "profile-an-existing-wiki-with-an-agent.md" >}}).

`inspect` is read-only and needs no `.katalyst/` project. It reports
**evidence** — counts and distributions — never recommendations. Reading the
evidence and deciding the schema is your call.

## 1. Inspect the directory

Point `inspect` at the directory:

```bash
katalyst inspect ./wiki
```

You get a Markdown report grouped by family. The fields that matter most:

```
### object_field_frequency (n=142)
- title:  { present: 142 }
- author: { present: 141 }
- status: { present: 142 }
- isbn:   { present: 17 }

### object_field_values (n=142)
- status: { cardinality: 3, values: { read: 80, reading: 12, to-read: 50 } }
```

For a machine-readable form an agent can parse, add `--json`; to save the
report, use `-o report.md`.

## 2. Read the evidence

Each inspector answers one question. Translate its counts into schema
decisions yourself — the threshold is your judgment, not the tool's:

| Inspector | What it tells you | A reasonable reading |
|---|---|---|
| `object_field_frequency` | how often each field appears | present in nearly every file → `required`; sometimes → optional |
| `object_field_values` | distinct values of a field | a small, stable set → an `enum` |
| `object_field_types` | observed types per field | one consistent type → a `type` constraint; mixed → a field to clean up first |
| `object_field_numeric_range` / `string_length` | observed bounds | a `min`/`max` or length constraint |
| `markdown_heading_shape` | single-H1, H1-matches-title, level jumps | `markdown_single_h1`, `markdown_title_matches_h1` |
| `markdown_sections` | recurring section headings | a `markdown_required_section` |
| `filesystem_naming` | casing, spaces, extensions | `filesystem_name_case` (`style: kebab`), `filesystem_path_charset` (`deny: [" "]`) |

The denominator `n` is always reported, so you decide what "nearly every file"
means. The outliers — the 17-of-142 `isbn`, the three filenames with spaces —
are exactly the files a schema will flag.

## 3. Draft a schema from the evidence

`inspect` does not write anything; you author the `.katalyst/` files (or have
an agent draft them). A schema for the evidence above:

```yaml
# .katalyst/schemas/book.yaml
type: object
required: [title, author, status]
properties:
  title:  { type: string }
  author: { type: string }
  status: { enum: [read, reading, to-read] }
```

```yaml
# .katalyst/collections/books.yaml
path: wiki
schema: book
checks:
  - kind: markdown_single_h1
  - kind: filesystem_name_case
    style: kebab
```

See [Add a schema]({{< relref "add-a-schema.md" >}}) for the binding details.

## 4. Check and iterate

Run `check` against the draft:

```bash
katalyst check books
```

The files that already follow the conventions pass; the outliers the evidence
flagged light up as violations. From there you tighten the schema, relax a
field to optional, or fix the stray files — then re-run. That loop — *inspect →
draft → check → fix the holdouts* — is the whole onboarding.

## See also

- [Profile an existing wiki with an agent]({{< relref "profile-an-existing-wiki-with-an-agent.md" >}}) — the same loop, driven by an agent.
- [Inspectors reference]({{< relref "../reference/inspectors/_index.md" >}}) — every inspector and what it reports.
- [Add a schema]({{< relref "add-a-schema.md" >}}) — bind the draft to a collection.
