+++
title = "Getting started"
weight = 20
+++

## Install

Install from source:

```bash
git clone https://github.com/abegong/katalyst
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

- `.katalyst/config.yaml`, commented project settings
- `.katalyst/schemas/`, one schema per file (empty to start)
- `.katalyst/bases/local.yaml`, the default base (the local
  filesystem), where you declare collections

It writes no example content. Add a schema under `.katalyst/schemas/` and
declare a collection inside `.katalyst/bases/local.yaml`, then run
`katalyst check`.

Next:
- [Configuration]({{< relref "reference/configuration.md" >}})
- [Check types reference]({{< relref "reference/check-types/_index.md" >}})
