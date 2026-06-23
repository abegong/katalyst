+++
title = "Getting started"
weight = 20
+++

> [!WARNING]
> ⚠️ **Katalyst is in its earliest days.** I'm actively building it in the open, which means things are incomplete, rough in places, and likely to change without notice. APIs, commands, config formats, and concepts can all break between commits. Please don't rely on it for anything important yet, but I'd genuinely love your feedback as it takes shape.

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
- `.katalyst/storage/local.yaml`, the default storage instance (the local
  filesystem), where you declare collections

It writes no example content. Add a schema under `.katalyst/schemas/` and
declare a collection inside `.katalyst/storage/local.yaml`, then run
`katalyst check`.

Next:
- [Configuration]({{< relref "reference/configuration.md" >}})
- [Check types reference]({{< relref "reference/check-types/_index.md" >}})
