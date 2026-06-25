Once a collection is configured, `inspect <name>` runs the collection inspectors. `object_fields` reports, per field, how often it appears over `n`, its observed types, and its value cardinality: the evidence behind `required`/optional and `enum` decisions.

### Input

`wiki/dune.md`

```markdown
---
title: Dune
author: Frank Herbert
status: read
---
# Dune

## Review
A landmark of the genre.
```

`wiki/neuromancer.md`

```markdown
---
title: Neuromancer
author: William Gibson
status: reading
---
# Neuromancer

## Review
```

`wiki/foundation.md`

```markdown
---
title: Foundation
author: Isaac Asimov
status: to-read
---
# Foundation

## Review
```

`wiki/snow-crash.md`

```markdown
---
title: Snow Crash
author: Neal Stephenson
status: read
---
# Snow Crash

## Review
```

`wiki/Dune Messiah.md`

```markdown
---
title: Dune Messiah
status: read
---
# Dune Messiah
```

`.katalyst/bases/my_directory.yaml`

```yaml
type: filesystem
root: .
collections:
  books:
    path: wiki
    schema: book
```

`.katalyst/schemas/book.yaml`

```yaml
type: object
required: [title, author, status]
properties:
  title:  { type: string }
  author: { type: string }
  status: { enum: [read, reading, to-read] }
```

### Command

```console
$ katalyst inspect books --inspector object_fields -v
# Inspection report: books

## Object

### object_fields (n=5)

_A data dictionary over item frontmatter: per-field presence, types, cardinality, and common values._

- author:
  - cardinality: 4
  - present: 4
  - types:
    - string: 4
  - values:
    - Frank Herbert: 1
    - Isaac Asimov: 1
    - Neal Stephenson: 1
    - William Gibson: 1
- status:
  - cardinality: 3
  - present: 5
  - types:
    - string: 5
  - values:
    - read: 3
    - reading: 1
    - to-read: 1
- title:
  - cardinality: 5
  - present: 5
  - types:
    - string: 5
  - values:
    - Dune: 1
    - Dune Messiah: 1
    - Foundation: 1
    - Neuromancer: 1
    - Snow Crash: 1
```

