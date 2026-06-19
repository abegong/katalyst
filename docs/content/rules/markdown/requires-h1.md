+++
title = "Requires H1"
+++

## Rule ID

`kind: markdown_requires_h1`

## Purpose

Require at least one H1 heading in the markdown body.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: markdown_requires_h1
```
