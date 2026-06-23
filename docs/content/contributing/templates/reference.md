+++
title = "Reference page template"
weight = 10
draft = true
+++

<!--
TEMPLATE, copy this file into docs/reference/ and fill it in.

This page IS: an information-oriented description of one thing, a config
key, a command, a check type, a term. It is looked up, not read top to
bottom. It is accurate, complete, and austere.

This page is NOT: a tutorial (no "first, then, next"), a how-to (no
task-driven steps), or an explanation (no rationale, no "why"). Link out to
those instead of absorbing them.

Keep prose minimal. Prefer tables and lists. State defaults explicitly.
-->

# <Thing being described>

One-sentence statement of what this is.

## Synopsis

```text
<the canonical form: config snippet, command signature, etc.>
```

## Fields / options

| Name | Required | Default | Meaning |
|---|---|---|---|
| `<name>` | yes/no | `<default>` | <what it does> |

## Behavior

Factual description of what happens. Edge cases and error conditions belong
here, stated plainly.

## See also

- Related reference page
- The relevant explanation page for the *why* (link with `relref` once you
  copy this into `docs/reference/`)
