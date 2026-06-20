+++
title = "Katalyst Documentation"
+++

# Katalyst

Katalyst is an experimental framework for content validation in AI-native
systems. Today the implementation validates markdown frontmatter against
JSON Schema and a registry of structural checks, driven by a CLI. That is a
starting slice, not the full intended scope — over time Katalyst aims to
support richer validation across structured and unstructured content,
multiple storage backends, and migrations between them.

These docs get you oriented first, then go deep.

## Get oriented

1. **[Why Katalyst]({{< relref "why-katalyst.md" >}})** — the problem it solves
   and where it's headed.
2. **[Getting started]({{< relref "getting-started.md" >}})** — build the CLI,
   scaffold a project, and run your first checks.
3. **[Core concepts]({{< relref "core-concepts.md" >}})** — the
   backend-agnostic model the tool is built on.

## Use Katalyst

For people validating content with the tool.

- **[How-to guides]({{< relref "how-to/_index.md" >}})** — task-oriented
  recipes: [configure checks]({{< relref "how-to/configure-rules.md" >}}),
  [add a schema]({{< relref "how-to/add-a-schema.md" >}}),
  [validate in CI]({{< relref "how-to/validate-in-ci.md" >}}).
- **[Reference]({{< relref "reference/_index.md" >}})** — the
  [configuration]({{< relref "reference/configuration.md" >}}) surface, the
  generated [rule reference]({{< relref "reference/rules/_index.md" >}}),
  and the [glossary]({{< relref "reference/glossary.md" >}}).

## Go deeper

For people working on Katalyst, and anyone who wants the bigger picture.

- **[Deep dives]({{< relref "deep-dives/_index.md" >}})** — the cross-cutting
  *why*: connectors, progressive operations, and design rationale that no single
  package owns. Subsystem rationale lives next to the code in the per-package
  `README.md` files under `internal/`.
- **[Contributing]({{< relref "contributing/_index.md" >}})** — how we
  [document]({{< relref "contributing/how-we-document.md" >}}) and
  [plan]({{< relref "contributing/how-we-plan.md" >}}) changes.
