The `markdown_title_matches_h1` check ties a frontmatter field to the document's first H1. When they disagree, `check` reports the mismatch and exits 1.

## Input

`.katalyst/storage/local.yaml`

```yaml
type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_title_matches_h1
        field: title
```

`notes/dune.md`

```markdown
---
title: Dune
---
# Children of Dune
```

## Command

```console
$ katalyst check notes/dune
<project>/notes/dune.md:4: /title: "Dune" does not match first H1 "Children of Dune"
exit status 1
```

