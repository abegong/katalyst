+++
title = "Title Matches H1"
+++

## Rule ID

`kind: markdown_title_matches_h1`

## Purpose

Require a frontmatter field to match the first H1 heading in the markdown body.

## Configuration

```yaml
rules:
  - paths: "notes/**/*.md"
    checks:
      - kind: markdown_title_matches_h1
        field: title
```

Fields:

- `kind` must be `markdown_title_matches_h1`
- `field` is optional; default is `title`

## Behavior

The rule reads the configured frontmatter field and compares it to the first
body heading that starts with `# `. A mismatch produces a validation error.

## Example validation failure

Input file:

```markdown
---
title: Dune
---
# Children of Dune
```

Command:

```bash
katalyst validate notes/dune.md
```

Output:

```text
notes/dune.md:4: /title: "Dune" does not match first H1 "Children of Dune"
```
