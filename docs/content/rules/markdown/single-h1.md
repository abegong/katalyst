+++
title = "Single H1"
+++

## Rule ID

`kind: markdown_single_h1`

## Purpose

Require that the markdown body contains at most one H1 heading.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: markdown_single_h1
```
