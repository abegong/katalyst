+++
title = "Required Field"
+++

## Rule ID

`kind: object_required_field`

## Purpose

Require that a frontmatter field exists.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: object_required_field
        field: year
```
