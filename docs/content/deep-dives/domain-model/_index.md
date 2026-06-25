+++
title = "Domain model"
weight = 30
bookCollapseSection = true
+++

# Domain model

Katalyst reasons in a small vocabulary that is general enough to describe a
Postgres table, a directory of markdown files, a MongoDB collection, and a
hosted API response the same way. That shared vocabulary is what lets checks,
inspectors, selectors, and future backends fit the same model instead of
becoming one-off adapters.

This page introduces the concepts and how they fit. Each term's canonical
definition lives in the [glossary]({{< relref "../../reference/glossary.md" >}}).

## The concepts

- A **project** is the whole workspace katalyst operates over: a configured
  root that binds one or more storage backends into named collections. Its
  configuration is the **config**; in katalyst today, that config is the
  `.katalyst/` directory. An empty selector addresses the whole project.
- **Storage** is a backend that holds data: a filesystem, a SQLite database, a
  Postgres instance, an S3 bucket, or another store. Katalyst's implementation
  is the [storage layer]({{< relref "storage.md" >}}), where a storage instance
  maps backend-native references into the domain model.
- A **collection** is a group of items that share structure: a directory of
  similar files, a relational table, a Mongo collection, or a family of API
  resources. Collections are the unit that owns checks and that users address
  by name. See [Collections]({{< relref "collections.md" >}}).
- An **item** is one unit of data in a collection: a markdown file, a table row,
  a Mongo document, or one API resource.
- An **attribute** is a named characteristic of an item: a column, a
  frontmatter key, a response field, its filename, its path, or another
  backend-derived property. A key in a structured object specifically is a
  **field**.
- An **operation** is something a backend lets you do with data: read, list,
  aggregate, write, and eventually query. Which operations a backend supports,
  and what structural commitments those operations require, is the subject of
  [progressive operations]({{< relref "../progressive-operations.md" >}}).
- A **check** asserts a condition on an item, an attribute, or a whole
  collection and reports a violation when the condition fails. See
  [Checks]({{< relref "checks.md" >}}).
- An **inspector** is the descriptive dual of a check: it measures a
  distribution and returns evidence, never a verdict. See
  [Inspectors]({{< relref "inspectors.md" >}}).

## How the concepts fit

The hierarchy is intentionally small:

![Domain model diagram showing project containing storage, collection, item, and attribute, with checks and inspectors operating on the data model.](../../images/domain-model-core-concepts.png)

Storage locates data, collections group it, items are the units commands act
on, and attributes are the named things checks and inspectors can read.
Operations describe what the backend can do with those units. Checks and
inspectors sit on top: checks enforce a rule; inspectors measure the same
surface without enforcing anything.

That separation is why katalyst can start with markdown files but leave room
for richer stores. The check engine does not need to know whether an item came
from a file or a row if the storage layer can present the item, its attributes,
and the operations available on them.
