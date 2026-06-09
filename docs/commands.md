+++
title = "Commands"
+++

## Validate

```bash
katalyst validate [paths...]
```

Validate markdown frontmatter against a resolved schema.

## Format

```bash
katalyst fmt [paths...]
katalyst fmt --check [paths...]
```

Normalize frontmatter formatting.

## Schema

```bash
katalyst schema list
katalyst schema show <name>
```

Inspect configured schemas.

## CRUD

```bash
katalyst create <path> [key=value ...]
katalyst read <path>
katalyst update <path> key=value [key=value...]
katalyst delete <path> [path...]
```

Create, read, update, and delete markdown items with frontmatter.

## Init

```bash
katalyst init [--dir <path>]
```

Scaffold a starter project layout.
