+++
title = "Code Fence Language Required"
+++

## Rule ID

`kind: markdown_code_fence_language_required`

## Purpose

Require that opening fenced code blocks include a language tag.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: markdown_code_fence_language_required
```
