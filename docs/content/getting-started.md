+++
title = "Getting started"
weight = 20
+++

> [!NOTE]
> **Primary onboarding path: delegate setup to your agent.** Copy this into your
> agent client:
>
> ```text
> You are setting up Katalyst for this workspace.
>
> 1. Install the latest Katalyst CLI from the release binary for this machine:
>    curl -fsSL https://raw.githubusercontent.com/abegong/katalyst/main/scripts/install.sh | sh
>
> 2. Verify the CLI works:
>    katalyst --version
>
> 3. Download the Katalyst agent skills:
>    katalyst skills install
>
> 4. Tell me where the `.skill` files were written and what I need to do, if
>    anything, to activate them in this client.
> ```
>
> This is the intended onboarding path. The agent installs the CLI from a
> prebuilt release binary, downloads the current skill bundles, and reports any
> client-specific activation step. Manual installs are fallback paths.

## Install

The agent-delegated flow above is the recommended way to get started. If you only
want the CLI, install the latest release binary:

```bash
curl -fsSL https://raw.githubusercontent.com/abegong/katalyst/main/scripts/install.sh | sh
```

Or install with Go (1.25+) if you already have a Go toolchain:

```bash
go install github.com/abegong/katalyst@latest
```

The release includes macOS, Linux, and Windows archives for common Intel and ARM
machines. The archive names follow this pattern:

```text
katalyst_<version>_<os>_<arch>.tar.gz
katalyst_<version>_windows_<arch>.zip
```

Build from source only if you are working on Katalyst itself:

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
