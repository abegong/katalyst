+++
title = "File metadata"
weight = 40
+++

# File metadata

File metadata is the item view derived from an item's filesystem reference. It
does not parse item content; it reads names, extensions, parent directories,
path segments, and path depth from where the item lives.

## Terms

| Term | Meaning |
|---|---|
| **File metadata** | Attributes derived from an item's path or filesystem reference. |
| **Filename** | The basename of the item path. |
| **Extension** | The suffix used to classify the file's format, such as `.md` or `.txt`. |
| **Parent directory** | The directory immediately containing the item. |
| **Path depth** | The number of directory levels between the collection root and the item. |

## Model

File metadata belongs to the item because the item's reference can carry
meaning: a file's name may need to match a field, an extension may need to be
allowed, or a collection may require one index file per directory.

This view backs file-system check types. It also feeds source inspectors such
as `file_tree` and `document_shape`, where file names and paths help profile a
base before or after collections are configured.

Unlike [Markdown body text]({{< relref "markdown-body-text.md" >}}), file
metadata is not a codec. It is derived from the reference the base already uses
to address the item.

## Invariants

1. **File metadata is derived from references.** It does not require reading or
   parsing the item body.
2. **Path targets are explicit.** Checks choose the path slice they inspect:
   filename, filename with extension, parent directory, or path segments.
3. **It is still an item view.** Checks and inspectors can reason about path
   attributes alongside structured fields and body text.

## See also

- [File system check types]({{< relref "../check-types/file-system/_index.md" >}})
- [File tree inspector]({{< relref "../inspectors/source/file-tree.md" >}})
- [Document shape inspector]({{< relref "../inspectors/source/document-shape.md" >}})

