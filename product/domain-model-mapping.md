# Domain model mapping

A bridge between the general data-systems model in
[`general-model.md`](general-model.md) and the concrete
model in [`domain-model.md`](domain-model.md). Katabridge is one
instantiation of the general model — many others are valid
(databases, document stores, REST APIs); this document is only about
how katabridge today maps onto the general vocabulary.

When either source document changes, update this mapping in the same
commit. If a general concept can't be mapped cleanly onto current
katabridge, that's information — note it explicitly rather than
forcing a translation.

## Mapping

| Future concept | Today, in katabridge |
|----------------|----------------------|
| Data interface | The filesystem — specifically, the subtree rooted at the directory containing `katabridge.yaml`. Katabridge is single-backend today; nothing in the code knows about "data interface" as an abstraction. |
| Item           | A markdown document (frontmatter + body). |
| Collection     | The set of files matched by a `Rule`'s glob, sharing a schema. |
| Attribute      | A key in a document's frontmatter — *and* arguably the file's path, name, and existence of a body. The current code only validates frontmatter keys; the other "attributes of the document itself" are not yet first-class. |
| "Shared attributes" of a collection | The JSON Schema that all items in the collection must satisfy. |
