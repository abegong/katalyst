+++
title = "Domain model"
weight = 30
bookCollapseSection = true
+++

# Domain model

This page introduces core concepts in the Katalyst domain model and how they relate to each other.

<img
  src="../../images/domain-model-core-concepts.png"
  alt="Domain model diagram showing project containing storage, collection, item, and attribute, with checks and inspectors operating on the data model."
  style="display:block; margin:1rem auto; width:auto; max-width:600px; height:auto;"
/>

A **project** is the whole workspace Katalyst operates over: a configured
  root that binds one or more storage backends into named collections. Its
  configuration is the **config**; in Katalyst today, that config is the
  `.katalyst/` directory. An empty selector addresses the whole project.

**Data**

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

**Operations**

An **operation** is something a backend lets you do with data: read, list,
  aggregate, write, and eventually query. Which operations a backend supports,
  and what structural commitments those operations require, is the subject of
  [progressive operations]({{< relref "../progressive-operations.md" >}}).

- A **check** asserts a condition on an item, an attribute, or a whole
  collection and reports a violation when the condition fails. See
  [Checks]({{< relref "checks.md" >}}).
- An **inspector** is the descriptive dual of a check: it measures a
  distribution and returns evidence, never a verdict. See
  [Inspectors]({{< relref "inspectors.md" >}}).
