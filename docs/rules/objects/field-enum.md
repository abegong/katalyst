+++
title = "Field Enum"
+++

## Rule ID

`kind: object_field_enum`

## Purpose

Require that a string field is one of a fixed set of values.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: object_field_enum
        field: status
        values: [draft, published, archived]
```
