+++
title = "Filename Matches Slug"
+++

## Rule ID

`kind: filesystem_filename_matches_slug`

## Purpose

Require a frontmatter field to match the markdown file basename.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: filesystem_filename_matches_slug
        field: slug
```

Fields:

- `kind` must be `filesystem_filename_matches_slug`
- `field` is optional; default is `slug`

## Behavior

For `notes/dune.md`, the expected basename is `dune`. If the configured field
value differs, validation fails.

## Example validation failure

Input file path: `notes/dune.md`

```markdown
---
slug: dune-messiah
title: Dune Messiah
---
# Dune Messiah
```

Command:

```bash
katalyst validate notes/dune.md
```

Output:

```text
notes/dune.md:2: /slug: slug "dune-messiah" must match filename "dune"
```
