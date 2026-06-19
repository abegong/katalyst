+++
title = "Filename Prefix"
+++

## Rule ID

`kind: filesystem_filename_prefix`

## Purpose

Require that the filename starts with a specific prefix.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: filesystem_filename_prefix
        value: book-
```
