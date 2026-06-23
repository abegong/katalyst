+++
title = "Deep dives"
weight = 50
bookCollapseSection = true
+++

# Deep dives

Understanding-oriented discussion of the *why* behind Katalyst, the
[vision and scope]({{< relref "vision.md" >}}), the [core
concepts]({{< relref "core-concepts.md" >}}) the tool is built on, the
[domain model]({{< relref "domain-model.md" >}}) that instantiates them in
katalyst, and the deeper design discussions that no single page or package
owns: how the [core commands are organized]({{< relref "command-organization.md" >}}),
how the [storage layer]({{< relref "storage.md" >}}) maps stores onto the model,
and how operations grow richer as a backend's capabilities increase. For the short version, start with [Why
Katalyst]({{< relref "../why-katalyst.md" >}}).

Rationale that is tied to a specific subsystem lives next to the code, in the
per-package `README.md` files under `internal/` (for example
`internal/config`, `internal/frontmatter`, and `internal/checks`). The pages
here cover the cross-cutting *why* that no single package owns.
