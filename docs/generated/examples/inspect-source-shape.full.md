Pointed at a bare directory (no project), `inspect` runs the raw base inspectors. `document_shape` clusters files by a composite fingerprint, so a shared convention shows up as one class and the stragglers as outliers.

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

### Command

```console
$ katalyst inspect ./wiki --inspector file_content_shape --select "ext = \".md\""
# Inspection report: ./wiki

## Structural

### file_content_shape (n=5)

_Profile selected files by text, tabular, and tree content structure._

----------------------------------------
selection:
  expression    : ext = ".md"
  files         : 5
  directories   : 1
  readable      : 5
  unsupported   : 0
  parse failures: 0

----------------------------------------
file types:
  TYPE  FILES
  .md   5

----------------------------------------
coherence:
  status: coherent

----------------------------------------
common structure:
  - 5/5 Markdown files have an H1
  - 4/5 Markdown files have frontmatter key author
  - 5/5 Markdown files have frontmatter key status
  - 5/5 Markdown files have frontmatter key title
  - 4/5 Markdown files have section Review

----------------------------------------
variation:
  - frontmatter key author appears in 4/5 Markdown files

----------------------------------------
text:
  files  : 5
  with H1: 5
  frontmatter keys:
  KEY     FILES
  status  5
  title   5
  author  4

----------------------------------------
tabular:
  no CSV files selected

----------------------------------------
tree:
  no JSON files selected

----------------------------------------
read/parse issues:
  none
```
