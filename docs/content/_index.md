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

These docs are organized along two tracks.

## Use Katalyst

For people validating content with the tool.

1. **[Getting started]({{< relref "getting-started.md" >}})** — build the CLI,
   scaffold a project, and run your first checks.
2. **[How-to guides]({{< relref "how-to/_index.md" >}})** — task-oriented
   recipes: [configure checks]({{< relref "how-to/configure-rules.md" >}}),
   [add a schema]({{< relref "how-to/add-a-schema.md" >}}),
   [validate in CI]({{< relref "how-to/validate-in-ci.md" >}}).
3. **[Reference]({{< relref "reference/_index.md" >}})** — the
   [configuration]({{< relref "reference/configuration.md" >}}) surface, the
   generated [rule reference]({{< relref "reference/rules/_index.md" >}}),
   and the [glossary]({{< relref "reference/glossary.md" >}}).

## Contribute

For people working on Katalyst, and anyone who wants the bigger picture.

- **[Explanation]({{< relref "explanation/_index.md" >}})** — the *why*: the
  [manifesto]({{< relref "explanation/manifesto.md" >}}), the
  [general model]({{< relref "explanation/general-model.md" >}}), the
  [domain model]({{< relref "explanation/domain-model.md" >}}), and the
  design rationale behind each command.
- **[Contributing]({{< relref "contributing/_index.md" >}})** — how we
  [document]({{< relref "contributing/how-we-document.md" >}}) and
  [plan]({{< relref "contributing/how-we-plan.md" >}}), and the
  [roadmap]({{< relref "contributing/roadmap.md" >}}).
