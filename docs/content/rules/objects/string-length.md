+++
title = "String Length"
+++

## Rule ID

`kind: object_string_length`

## Purpose

Constrain minimum and/or maximum length of a string field.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: object_string_length
        field: title
        min_length: 3
        max_length: 120
```
