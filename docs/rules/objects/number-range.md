+++
title = "Number Range"
+++

## Rule ID

`kind: object_number_range`

## Purpose

Constrain a numeric field to a minimum and/or maximum value.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: object_number_range
        field: year
        min: 1900
        max: 2100
```
