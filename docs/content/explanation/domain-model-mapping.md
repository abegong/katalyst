+++
title = "Domain model mapping"
weight = 70
+++

# Domain model mapping

A bridge between the general data-systems model in
[General model]({{< relref "general-model.md" >}}) and the concrete
model in [Domain model]({{< relref "domain-model.md" >}}). Katalyst is one
instantiation of the general model — many others are valid
(databases, document stores, REST APIs); this document is only about
how katalyst today maps onto the general vocabulary.

When either source document changes, update this mapping in the same
commit. If a general concept can't be mapped cleanly onto current
katalyst, that's information — note it explicitly rather than
forcing a translation.

## Mapping

| Future concept | Today, in katalyst |
|----------------|----------------------|
| Data interface | The filesystem — specifically, the subtree rooted at the directory containing `.katalyst/`. Katalyst is single-backend today; nothing in the code knows about "data interface" as an abstraction. |
| Item           | A markdown document (frontmatter + body), addressed as `<collection>/<item>` where the item id is the filename stem. |
| Collection     | A named entry in `collections:` — a directory plus a filename `pattern` (default `*.md`) and the checks every item in it must pass. |
| Attribute      | A key in a document's frontmatter — *and* arguably the file's path, name, and existence of a body. The current code only validates frontmatter keys; the other "attributes of the document itself" are checked by the filesystem and markdown rule families but are not modeled as first-class attributes. |
| "Shared attributes" of a collection | The JSON Schema that all items in the collection must satisfy. |
