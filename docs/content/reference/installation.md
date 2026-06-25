+++
title = "Installation"
weight = 15
+++

# Installation

Install Katalyst with a release binary, Go, or a local source build.

## Release binary (recommended)

Use the install script to fetch the latest release for your OS and architecture:

```bash
curl -fsSL https://raw.githubusercontent.com/abegong/katalyst/main/scripts/install.sh | sh
```

## Go install

If you already have Go (1.25+), install directly from the module:

```bash
go install github.com/abegong/katalyst@latest
```

## Release archive names

Published archives follow these patterns:

```text
katalyst_<version>_<os>_<arch>.tar.gz
katalyst_<version>_windows_<arch>.zip
```

## Build from source

Build from source when you are developing Katalyst itself:

```bash
git clone https://github.com/abegong/katalyst
cd katalyst
make build
```

For contributor workflows and pre-push checks, see
[How to contribute]({{< relref "../contributing/how-to-contribute.md" >}}).
