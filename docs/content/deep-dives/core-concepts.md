+++
title = "Core concepts"
weight = 30
+++

# Core concepts

> **Status: work in progress.** A deliberately abstract sketch. These concepts
> are not about `katalyst` specifically; katalyst is one instantiation among
> many. Expect revisions until they settle.

The vocabulary katalyst reasons in, general enough to describe a Postgres table,
a directory of markdown files, and a MongoDB collection the same way, so the
abstractions built on top bridge them too. Each term's canonical definition
lives in the [glossary]({{< relref "../reference/glossary.md" >}}); this page
introduces the concepts and how they fit. For the katalyst-specific
instantiation, see the [domain model]({{< relref "domain-model.md" >}}).

## The concepts

- **Storage** is a backend that holds data: a filesystem, a SQLite database, a
  Postgres instance, an S3 bucket. Katalyst's realization is the
  [storage layer]({{< relref "storage.md" >}}).
- **Collection** is a group of items sharing structure: a directory of similar
  files, a relational table, a Mongo collection. See
  [collections]({{< relref "collections.md" >}}).
- **Item** is one unit of data in a collection: a markdown file, a table row, a
  Mongo document.
- **Attribute** is a named characteristic of an item: a column, a frontmatter
  key, a response field, even its name or path. A key in a structured object
  specifically is a **field**.
- **Operation** is something storage lets you do with its data: read, list,
  query, aggregate, write. Which operations a backend supports is the subject of
  [progressive operations]({{< relref "progressive-operations.md" >}}).
- **Check** asserts a condition on an item or its attributes and reports a
  violation when it fails. See [checks]({{< relref "checks.md" >}}).
- **Inspector** is the descriptive dual of a check: it measures a distribution
  and returns evidence, never a verdict. See
  [inspectors]({{< relref "inspectors.md" >}}).

## The same vocabulary across backends

| System               | Storage       | Collection      | Item       | Attribute        |
|----------------------|---------------|-----------------|------------|------------------|
| Postgres             | The database  | A table         | A row      | A column         |
| MongoDB              | The database  | A collection    | A document | A field          |
| A directory of CSVs  | The directory | A CSV file      | A row      | A column         |
| A REST API           | The API       | A resource type | A resource | A response field |
| An S3 bucket of JSON | The bucket    | A key prefix    | An object  | A JSON key       |

An operation defined once in this vocabulary, check an attribute, aggregate over
a collection, applies to every backend that supports it. Which operations a
backend supports, and the structural commitments each demands, is the
[progressive operations]({{< relref "progressive-operations.md" >}}) story.
