`fix` rewrites frontmatter into a canonical form (here, sorting the keys) while leaving the markdown body byte-for-byte unchanged. It is idempotent and never injects missing keys.

## Input

`notes/doc.md`

```markdown
---
zebra: 1
apple: 2
---
# Body
verbatim
```

`.katalyst/bases/my_directory.yaml`

```yaml
type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
```

## Command

```console
$ katalyst fix notes/doc
<project>/notes/doc.md
```

## Result

`notes/doc.md` after `katalyst fix notes/doc`:

```markdown
---
apple: 2
zebra: 1
---
# Body
verbatim
```

