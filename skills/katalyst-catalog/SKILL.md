---
name: katalyst-catalog
description: >-
  Take stock of an existing body of content with katalyst: profile what's there,
  surface its recurring kinds of item (candidate collections), and get oriented
  before declaring any structure. Use when a user points at an existing wiki,
  docs tree, or notes corpus and wants to understand it, "see what's in here", or
  prepare to onboard it into katalyst. Precedes katalyst-identify-collections.
---

# Catalog existing content

Before declaring any structure, take stock of what the content already is.
Katalyst's `inspect` reads a corpus and reports descriptive evidence — counts and
distributions — so you can see the recurring shapes without guessing or scanning
every file by hand. The goal of this stage is orientation: what is here, and what
are the candidate collections.

If the CLI is missing, run `./bootstrap.sh` first.

## Profile the corpus

Point `inspect` at the content to profile it:

```bash
katalyst inspect <path>
```

`inspect` returns **evidence**, never a verdict — frontmatter fields and how
often they appear, body structure, file naming and layout. Read it to answer:

- What recurring **kinds of item** are there? Clusters of files that share
  frontmatter fields and body shape are candidate collections.
- Which fields are near-universal (likely required) versus sparse (likely
  optional or accidental)?
- How is the content laid out — directories, naming conventions, extensions?

## Map candidate collections

From the evidence, sketch the collections the content is made of: the repeatable
object types it has many instances of (e.g. "meeting notes", "people", "API
endpoints"). You are not declaring anything yet — just naming the groups and the
fields each seems to carry. That map is the input to the next stage.

## Next

Hand the candidate collections to **katalyst-identify-collections**, which turns
this informal map into the collections katalyst will track. Defer declaring
schemas and checks until then — cataloging is about understanding what exists,
not yet constraining it.
