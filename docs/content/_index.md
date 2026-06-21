+++
title = "Katalyst Documentation"
+++

# Katalyst

Katalyst is a **content consistency layer**: schemas and checks that live with
your content and keep its structure steady as it grows — built for the content
AI agents manage. [Why Katalyst]({{< relref "why-katalyst.md" >}}) makes the
case; these docs get you oriented, then go deep.

## Get oriented

1. **[Why Katalyst]({{< relref "why-katalyst.md" >}})** — the problems it
   solves.
2. **[Getting started]({{< relref "getting-started.md" >}})** — build the CLI,
   scaffold a project, and run your first checks.

## Use Katalyst

For people validating content with the tool.

- **[How-to guides]({{< relref "how-to/_index.md" >}})** — task-oriented
  recipes: [configure checks]({{< relref "how-to/configure-rules.md" >}}),
  [add a schema]({{< relref "how-to/add-a-schema.md" >}}),
  [validate in CI]({{< relref "how-to/validate-in-ci.md" >}}).
- **[Reference]({{< relref "reference/_index.md" >}})** — the
  [configuration]({{< relref "reference/configuration.md" >}}) surface, the
  generated [check types reference]({{< relref "reference/check-types/_index.md" >}}),
  and the [glossary]({{< relref "reference/glossary.md" >}}).

## Go deeper

For people working on Katalyst, and anyone who wants the bigger picture.

- **[Deep dives]({{< relref "deep-dives/_index.md" >}})** — the *why*: the
  [vision and scope]({{< relref "deep-dives/vision.md" >}}), the [core
  concepts]({{< relref "deep-dives/core-concepts.md" >}}), connectors,
  progressive operations, and design rationale that no single package owns.
  Subsystem rationale lives next to the code in the per-package `README.md`
  files under `internal/`.
- **[Contributing]({{< relref "contributing/_index.md" >}})** — how we
  [document]({{< relref "contributing/how-we-document.md" >}}) and
  [plan]({{< relref "contributing/how-we-plan.md" >}}) changes.
