+++
title = "Getting started"
weight = 20
+++

## Install

Install the latest release with Go (1.25+):

```bash
go install github.com/abegong/katalyst@latest
```

Or download a prebuilt binary from the
[latest GitHub Release](https://github.com/abegong/katalyst/releases/latest).
Releases include macOS, Linux, and Windows archives for common Intel and ARM
machines.

Build from source only if you are working on katalyst itself:

```bash
git clone https://github.com/abegong/katalyst
cd katalyst
make build
```

## Agent setup

Katalyst also ships task skills for agents. Download the `.skill` files from the
[latest GitHub Release](https://github.com/abegong/katalyst/releases/latest) and
install them with your agent client.

Each skill includes a shared bootstrap script. If an agent needs the CLI and
`katalyst` is not already on `PATH`, run the bundled script from inside the
installed skill:

```bash
./bootstrap.sh
```

The script is idempotent. It reuses an existing `katalyst` command when one is
available; otherwise it detects the current OS and architecture, downloads the
matching archive from the latest GitHub Release, and installs it into
`~/.local/bin`. Set `KATALYST_INSTALL_DIR` before running the script to install
somewhere else. If no matching archive is available, the script falls back to
`go install github.com/abegong/katalyst@latest`.

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
