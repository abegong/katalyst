+++
title = "No Heading Level Jumps"
+++

## Rule ID

`kind: markdown_no_heading_level_jumps`

## Purpose

Disallow jumps larger than one heading level (for example `H1 -> H3`).

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: markdown_no_heading_level_jumps
```
