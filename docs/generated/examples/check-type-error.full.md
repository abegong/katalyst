Here `year` is a string, not an integer. An inline `object_field_type` check fails the item, points at the offending field with a JSON pointer (`/year`) and a `path:line` prefix, and exits 1.

### Input

`notes/dune.md`

```markdown
---
title: Dune
year: "not a number"
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
    checks:
      - kind: object_field_type
        field: year
        type: integer
```

### Command

```console
$ katalyst check notes/dune
<project>/notes/dune.md:3: /year: field "year" must be type "integer"
exit status 1
```

