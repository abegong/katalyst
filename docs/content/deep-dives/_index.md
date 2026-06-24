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
owns: how [checks work]({{< relref "checks.md" >}}) and the libraries that run them,
how the [storage layer]({{< relref "storage.md" >}}) maps stores onto the model,
and how operations grow richer as a backend's capabilities increase. For the
short version, start with [Welcome]({{< relref "../welcome.md" >}}).

These pages carry the **behavioral *why*** - any rationale a user can observe -
and each subsystem's **architecture**: how it is built, its entities, and the
design decisions behind it. A package's `AGENTS.md` holds only its code
conventions and a pointer back to its page here.
