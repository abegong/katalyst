Here `year` is a string, not an integer. `check` fails the item, points at the offending field with a JSON pointer (`/year`) and a `path:line` prefix, and exits 1.

### Input

`.katalyst/schemas/book.yaml`

```yaml
type: object
required: [title, year]
properties:
  title: { type: string }
  year:  { type: integer }
```

`.katalyst/storage/local.yaml`

```yaml
type: filesystem
root: .
collections:
  notes:
    path: notes
    schema: book
```

`notes/dune.md`

```markdown
---
title: Dune
year: "not a number"
---
# Dune
```

### Command

```console
$ katalyst check notes/dune
<project>/notes/dune.md:3: /year: got string, want integer
exit status 1
```

