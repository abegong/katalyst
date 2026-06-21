+++
title = "Getting Started"
weight = 20
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
katalyst check
```

`katalyst init` prepares the current directory as a katalyst project:

- `.katalyst/config.yaml` — commented project settings
- `.katalyst/schemas/` — one schema per file (empty to start)
- `.katalyst/collections/` — one collection per file (empty to start)

It writes no example content. Add a schema under `.katalyst/schemas/` and a
collection under `.katalyst/collections/`, then run `katalyst check`.

Next:
- [Configuration]({{< relref "reference/configuration.md" >}})
- [Check types reference]({{< relref "reference/check-types/_index.md" >}})
