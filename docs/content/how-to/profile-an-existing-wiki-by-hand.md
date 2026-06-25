+++
title = "Profile an existing wiki by hand"
weight = 5
+++

# Profile an existing wiki by hand

You have a directory of markdown: a vault, a docs tree, a knowledge base,
and you want a Katalyst schema for it. Rather than guess the conventions,
`inspect` measures them. This guide turns an existing corpus into a draft
schema **by reading the evidence yourself**. To hand that judgment to an agent
instead, see [Profile an existing wiki with an
agent]({{< relref "profile-an-existing-wiki-with-an-agent.md" >}}).

`inspect` reports **evidence**, counts and distributions, never
recommendations. Reading the evidence and deciding the schema is your call. It
runs in **two layers**: point it at a **directory** to profile a raw store
(no project needed), or at a configured **collection** to profile its items.
The onboarding loop uses both.

## 1. Survey the directory (source layer)

Point `inspect` at the directory. With no `.katalyst/` project it runs the
source inspectors:

```bash
katalyst inspect ./wiki
```

`file_tree` reports the file types and naming conventions per directory. Use it
to decide which directory or prefix you want to inspect more closely. Then run
`file_content_shape` over that explicit slice:

{{< katalyst-example "inspect-source-shape" >}}

This layer reports store and content facts, not candidate collections. Here the
Markdown files share enough structure that you can reasonably treat `./wiki` as
a single `books` collection and keep the file with the missing `author` in mind
as cleanup work.

## 2. Configure the collection

Point a collection at the directory so the field-level layer can run. Minimal
config:

```yaml
# .katalyst/bases/local.yaml
type: filesystem
root: .
collections:
  books:
    path: wiki
```

## 3. Inspect the collection (collection layer)

Now inspect the collection by name. Inside the project, `inspect` runs the
collection inspectors over its items:

```bash
katalyst inspect books
```

`object_fields` is a **data dictionary** over the items' frontmatter, per
field, presence over `n`, observed types, value cardinality, and the common
values when the set is small:

{{< katalyst-example "inspect-collection-fields" >}}

`markdown_body` reports the body conventions: single-H1 / H1-matches-title rates
and recurring section headings. For a machine-readable form, add `--json`; to
save the report, use `-o report.md`.

## 4. Read the evidence

Translate the counts into schema decisions yourself, the threshold is your
judgment, not the tool's:

| Evidence | What it tells you | A reasonable reading |
|---|---|---|
| `object_fields` `present` / `n` | how often a field appears | nearly every item → `required`; sometimes → optional |
| `object_fields` `values` | a small, stable value set | an `enum` |
| `object_fields` `types` | observed types per field | one consistent type → a `type` constraint; mixed → a field to clean up first |
| `markdown_body` heading shape | single-H1, H1-matches-title | `markdown_single_h1`, `markdown_title_matches_h1` |
| `markdown_body` sections | recurring section headings | a `markdown_required_section` |
| `file_tree` naming (step 1) | casing, spaces, extensions | `filesystem_name_case` (`style: kebab`), `filesystem_path_charset` (`deny: [" "]`) |
| `file_content_shape` common structure (step 1) | shared frontmatter keys and sections in the selected slice | confidence that the slice is coherent enough to configure as one collection |

The denominator `n` is always reported, so you decide what "nearly every item"
means. The one item missing `author`, which also has spaces in its name, is
exactly the kind of file a schema will flag.

## 5. Draft a schema and check

Add the schema and bind it to the collection:

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
# .katalyst/bases/local.yaml  (extend the collection from step 2)
type: filesystem
root: .
collections:
  books:
    path: wiki
    schema: book
    checks:
      - kind: markdown_single_h1
      - kind: filesystem_name_case
        style: kebab
```

See [Add a schema]({{< relref "add-a-schema.md" >}}) for the binding details.
Then run `check` against the draft:

```bash
katalyst check books
```

The files that already follow the conventions pass; the outliers the evidence
flagged light up as violations. From there you tighten the schema, relax a
field to optional, or fix the stray files, then re-run. That loop, *inspect →
draft → check → fix the holdouts*, is the whole onboarding.

## See also

- [Profile an existing wiki with an agent]({{< relref "profile-an-existing-wiki-with-an-agent.md" >}}): the same loop, driven by an agent.
- [Inspectors reference]({{< relref "../reference/inspectors/_index.md" >}}), every inspector and what it reports.
- [Add a schema]({{< relref "add-a-schema.md" >}}), bind the draft to a collection.
