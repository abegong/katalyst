+++
title = "Core concepts"
weight = 20
+++

# Core concepts

> **Status: work in progress.** This is a deliberately abstract sketch,
> captured as concepts are introduced. It is not about `katalyst`
> specifically — katalyst is one instantiation among many. Expect breaking
> revisions until the concepts settle.

## Goal

We tend to think about data in one of two modes:

- **Structured** — tables with column definitions, schemas, types,
  queries. Databases, warehouses, document stores.
- **Unstructured** — files, blobs, free-form text. Filesystems, object
  stores, "data lakes."

Tools, vocabulary, and mental models usually pick a side. A SQL DBA
and a markdown-vault note-taker have almost no shared language for
their work, even though the operations they want to perform —
*describe the shape of this data, find data that matches, change it
safely, check that constraints hold* — are essentially the same.

This document collects concepts that **bridge structured and
unstructured data**: terms general enough to describe a Postgres
table, a directory of markdown files, and a MongoDB collection with
the same vocabulary, so that abstractions built on top can bridge
them too.

## Concepts and Examples

### Data interface

A **data interface** is a formal protocol for accessing and interacting with data.

Examples:

- A filesystem (files on a disk).
- A SQLite database.
- A filesystem *and* a SQLite database in combination — a data
  interface can be heterogeneous.
- A structured backend such as DuckDB, MongoDB, or Postgres.
- An interface in front of any of those (read-only views, query APIs,
  federated readers all count; *storing* data is not a requirement —
  only *defining and exposing* it).

### Item

An **item** is a unit of data that you can interact with via a data interface.

Examples:

- A markdown file in a filesystem.
- A row in a relational database table.
- A document in a MongoDB collection.

### Collection

A **collection** is a group of items within a data interface sharing similar structure, attributes, provenance, etc.

Examples:

- Similarly formatted files within a directory.
- A table in a relational DB.
- A collection in MongoDB.


### Attribute

An **attribute** is a named characteristic or field of items within a collection.

Examples:

- A column definition (name + type) in a relational database table.
- A field within a MongoDB document.
- A key inside a markdown file's frontmatter.
- A document metadata — including things like its
  name, path, or how it's stored.

### Operation

An **operation** is something a data interface lets you do with its
data: read, write, query, transform, traverse. Every operation has a
defined **scope** (single item, single collection, across collections)
and a set of **structural requirements** the data interface must
satisfy for the operation to be available.

Examples:

- **Read** an item by identifier.
- **List** the items in a collection.
- **Write**, **delete**, **create** — the obvious mutations.
- **Search** items by raw content (substring or semantic similarity).
- **Query** items by structured attribute values.
- **Diff**
- **Aggregate** across items in a collection.

### Check

A **check** asserts a condition on items or their attributes. A **check type**
is a reusable definition of such a condition; a **check instance** is one check
type configured for a collection.

Examples:

- Type validation — an attribute must be a certain type.
- A foreign key constraint — an attribute must reference an item in another collection.
- Uniqueness — no two items in the collection share a value for this attribute.
- API-level validation — an attribute must satisfy a business rule (e.g. a valid email domain).

## Examples

Here are a few more examples, to help ground these concepts.

| System                  | Data interface         | Collection                  | Item                | Attribute                 |
|-------------------------|------------------------|-----------------------------|---------------------|---------------------------|
| Postgres                | The database           | A table                     | A row               | A column                  |
| MongoDB                 | The database           | A collection                | A document          | A field                   |
| A directory of CSVs     | The directory          | A CSV file                  | A row               | A column                  |
| A REST API              | The API                | A resource type             | A resource          | A response field          |
| An S3 bucket of JSON    | The bucket             | A key prefix                | An object           | A JSON key                |


## Implications

- The difference between "structured" and "unstructured" data largely comes down to *which operations are supported*.
- Structured data supports a wide range of operations; unstructured data is limited to a narrow set.
- Normalized relational databases sit at the strong end of "structured"--they have long been proven to support a general and powerful set of operations: joins across collections, filters on any column, aggregations, etc.
- SQL engines enforce a variety of upfront checks: columns, typed fields, NOT NULL, uniqueness, foreign keys.
- These checks are required in order to guarantee that the system can provide its catalog of operations.
- In other words, schemas and structuredness are a means to an end. They're about enforcing checks in order to provide a catalog of operations.

For how these general concepts are instantiated in katalyst specifically — the
concrete `Document`, `Schema`, `Collection`, and `Check` types and the
invariants between them — see the [domain model]({{< relref "domain-model.md" >}}).
For how they translate to today's code, see the per-package `README.md` files
under `internal/` (notably `internal/config`, `internal/frontmatter`, and
`internal/checks`).
