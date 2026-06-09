+++
title = "No Spaces In Path"
+++

## Rule ID

`kind: filesystem_no_spaces_in_path`

## Purpose

Disallow spaces anywhere in the file path.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: filesystem_no_spaces_in_path
```
