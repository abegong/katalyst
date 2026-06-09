+++
title = "Extension In"
+++

## Rule ID

`kind: filesystem_extension_in`

## Purpose

Allow only specific file extensions.

## Configuration

```yaml
rules:
  - paths: "notes/**/*"
    checks:
      - kind: filesystem_extension_in
        values: [.md, .markdown]
```
