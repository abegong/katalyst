`fix --check` is the read-only formatting gate: it lists items whose frontmatter is not canonical and exits 1, without modifying any file. Here `messy.md` has unsorted keys, so it is reported; `tidy.md` is already canonical and passes.

### Input

`notes/tidy.md`

```markdown
---
title: Tidy
---
# Tidy
```

`notes/messy.md`

```markdown
---
title: Messy
author: Ada
---
# Messy
```

`.katalyst/storage/my_directory.yaml`

```yaml
type: filesystem
root: .
collections:
  notes:
    path: notes
    checks:
      - kind: markdown_requires_h1
```

### Command

```console
$ katalyst fix --check
<project>/notes/messy.md
exit status 1
```

