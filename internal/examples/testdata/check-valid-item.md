The `notes` collection binds the `book` schema, which requires `title` and an integer `year`. This item satisfies both, so `check` exits 0 and prints OK.

## Input

`notes/dune.md`

```markdown
---
title: Dune
year: 1965
---
# Dune
```

`.katalyst/bases/my_directory.yaml`

```yaml
type: filesystem
root: .
collections:
  notes:
    path: notes
    schema: book
```

`.katalyst/schemas/book.yaml`

```yaml
type: object
required: [title, year]
properties:
  title: { type: string }
  year:  { type: integer }
```

## Command

```console
$ katalyst check notes/dune
<project>/notes/dune.md: OK
```

