+++
title = "Field Type"
+++

## Rule ID

`kind: object_field_type`

## Purpose

Require that a frontmatter field has a specific type.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: object_field_type
        field: year
        type: integer
```

Supported `type` values:
- `string`
- `boolean`
- `array`
- `object`
- `number`
- `integer`
