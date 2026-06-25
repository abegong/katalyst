With no target, `check` validates every collection. One item is missing its H1, so the run reports the violation and exits 1; the `exit status 1` line is what fails the CI step.

## Input

`notes/intro.md`

```markdown
---
title: Intro
---
# Intro
```

`notes/draft.md`

```markdown
---
title: Draft
---
No heading here.
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
$ katalyst check
<project>/notes/intro.md: OK
<project>/notes/draft.md: /: missing H1 heading in markdown body
exit status 1
```

