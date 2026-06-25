+++
title = "Domain model deep-dive template"
weight = 25
draft = true
+++

<!--
TEMPLATE, copy this file into docs/content/deep-dives/domain-model/ and fill it in.

This page IS: an explanation of one Katalyst domain-model concept, command
surface, or subsystem. It defines the terms, explains the model, records the
design rationale, and names the invariants that should stay true.

This page is NOT: a reference page for every config key, a how-to recipe, or a
generated catalog. Link to the relevant reference page for precise syntax and
to Go docs for package-level implementation detail.

Vision, strategy, and progression essays do not need to use this structure.
-->

# <Concept>

One or two short paragraphs that define the concept, name what owns it, and
explain what it connects to. Link to the reference page when the reader needs
precise syntax rather than design rationale.

## Terms

| Term | Meaning |
|---|---|
| **<Term>** | Definition in user-facing vocabulary. Mention code identifiers only when they clarify the seam. |

## Model

Explain how the concept works structurally. Prefer one coherent model section
over several scattered "why" sections. Use diagrams or tables when they make
relationships easier to scan.

## Lifecycle

Use this section only when the page describes a command or process. Describe
the ordered flow from input to output, including where errors accumulate and
where state changes happen.

## Design rationale

**Decision name.** Explain why the system works this way, including the
trade-off. When this choice replaced an earlier approach, record that history
here rather than in a separate decision log.

## Invariants

1. **Invariant name.** The rule that must stay true.
2. **Invariant name.** The rule that must stay true.

## Extension points

Use this section only when there is a code seam, planned backend, or future
expansion path worth naming.

## See also

- The reference page for precise syntax.
- Related domain-model pages.
- `go doc ./internal/<package>` for the code-level contract.
