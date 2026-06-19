+++
title = "Required Section"
+++

## Rule ID

`kind: markdown_required_section`

## Purpose

Require that a heading with specific text exists somewhere in the body.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: markdown_required_section
        heading: Summary
```
