+++
title = "Domain model"
weight = 30
bookCollapseSection = true
+++

# Domain model

This page introduces core concepts in the Katalyst domain model and how they relate to each other.

## Bases

The most central concept in Katalyst is a **base**: a storage system that holds **content** (data) and supports a specific set of **operations**. Katalyst is compatible with several different types of backend: filesystems, key-value stores, relational databases, etc.

An **operation** is something a base lets you do with data: read, list,
aggregate, write, and eventually query. Which operations a base supports,
and what structural commitments those operations require, is the subject of
[progressive operations]({{< relref "../progressive-operations.md" >}}).

In addition to natively-supported operations for various backends, Katalyst provides two very useful kinds of operation.

- A **check** makes an assertion about content and reports a violation if the condition fails. See
  [Checks]({{< relref "checks.md" >}}).
- An **inspector** is the descriptive dual of a check: it gathers and reports the state of content. See
  [Inspectors]({{< relref "inspectors.md" >}}).

<img
  src="../../images/domain-model-core-concepts.png"
  alt="Domain model diagram showing project containing base, collection, item, and attribute, with checks and inspectors operating on the data model."
  class="diagram--domain-model"
/>

## Raw vs collection-configured bases

When configuring a base, the most important division is between **raw content** and **collectionized content**. A base configured only for raw content supports only a limited set of operations: checks, inspections and a small set of fixes. Most operations that require writes are not permitted, because the system would not have the context necessary to guarantee that the new content is correct.

When a base is configured with **collections**, it can guarantee correctness and consistency for more operations. Check and inspect operations can be more specific and context-aware. Far more write operations are available, since the system now has more context to enable correctness and consistency.

Within a given base, collection configs do not replace raw configs. Instead, they stack on top. Similarly, operations that require a collection stack on top of those available when the base was only configured for raw access to content.

## Projects

A **project** is the whole workspace Katalyst operates over: a configured root that includes one or more bases, plus some additional metadata.
