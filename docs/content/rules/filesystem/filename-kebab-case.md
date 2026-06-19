+++
title = "Filename Kebab Case"
+++

## Rule ID

`kind: filesystem_filename_kebab_case`

## Purpose

Require lowercase kebab-case filenames (without extension).

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: filesystem_filename_kebab_case
```
