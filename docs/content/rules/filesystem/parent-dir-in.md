+++
title = "Parent Directory In"
+++

## Rule ID

`kind: filesystem_parent_dir_in`

## Purpose

Require that the file's parent directory name is in an allowed set.

## Configuration

```yaml
rules:
  - paths: "**/*.md"
    checks:
      - kind: filesystem_parent_dir_in
        values: [books, people]
```
