The `books` collection binds the `book` schema (`title` plus an integer `year`). `dune.md` satisfies the schema and reports OK; `foundation.md` omits `year`, so `check` reports the missing required property and exits 1.

### Input

`notes/books/dune.md`

```markdown
---
title: Dune
year: 1965
---
# Dune
```

`notes/books/foundation.md`

```markdown
---
title: Foundation
---
# Foundation
```

`.katalyst/bases/my_directory.yaml`

```yaml
type: filesystem
root: .
collections:
  books:
    path: notes/books
    schema: book
```

`.katalyst/schemas/book.yaml`

```yaml
$schema: https://json-schema.org/draft/2020-12/schema
title: book
type: object
required: [title, year]
properties:
  title: { type: string, minLength: 1 }
  year:  { type: integer, minimum: 0 }
```

### Command

```console
$ katalyst check books
<project>/notes/books/dune.md: OK
<project>/notes/books/foundation.md: /: missing property 'year'
exit status 1
```

