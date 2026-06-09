+++
title = "Getting Started"
+++

## Install

Install from source:

```bash
git clone https://github.com/katabase-ai/katalyst
cd katalyst
make build
```

## Quickstart

```bash
mkdir my-notes && cd my-notes
katalyst init
katalyst validate notes/example.md
```

`katalyst init` scaffolds:

- `katalyst.yaml`
- `schemas/book.json`
- `notes/example.md`

Next:
- [Configuration]({{< relref "configuration.md" >}})
- [Rules Reference]({{< relref "rules/_index.md" >}})
