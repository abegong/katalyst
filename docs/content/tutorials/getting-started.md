+++
title = "Getting started"
weight = 10
+++

# Getting started

This lesson takes you from an empty directory to a validated collection of
markdown notes. By the end you will have scaffolded a project, run the
checker on a clean file, watched it catch a real error, and tidied
frontmatter with `fix`. It assumes only a working Go toolchain.

## 1. Build the CLI

Install from source:

```bash
git clone https://github.com/katabase-ai/katalyst
cd katalyst
make build
```

That produces `bin/katalyst`. Put it on your `PATH`, or call it by path. The
rest of this lesson writes `katalyst`.

## 2. Scaffold a project

Make a fresh directory and scaffold it:

```bash
mkdir my-notes && cd my-notes
katalyst init
```

`init` creates three files and refuses to overwrite anything that already
exists:

```text
created katalyst.yaml
created schemas/book.json
created notes/example.md
```

- `katalyst.yaml` — one collection, `notes`, backed by the `notes/`
  directory and the `book` schema, plus two structural checks.
- `schemas/book.json` — a small JSON Schema requiring `title` and `year`.
- `notes/example.md` — a valid example note.

## 3. Check the project

Run the checker over everything:

```bash
katalyst check
```

The scaffold is valid, so every item reports `OK`:

```text
notes/example.md: OK
```

`check` with no selector walks every collection. You can narrow it: `katalyst
check notes` checks one collection, and `katalyst check notes/example` checks
a single item.

## 4. Watch it catch an error

Open `notes/example.md` and change `year` to a string:

```markdown
---
slug: example
tags:
  - example
title: Example
year: "two thousand"
---
# Example
```

Check again:

```bash
katalyst check notes/example
```

This time the `book` schema rejects the value, and the report points at the
exact line:

```text
notes/example.md:6: /year: got string, want integer
```

Restore `year: 2026` and `check` returns to `OK`.

## 5. Tidy with `fix`

`katalyst fix` rewrites frontmatter into a canonical form — keys sorted, one
trailing newline, body untouched. Add some messy spacing or reorder the keys
in `notes/example.md`, then preview what would change without writing:

```bash
katalyst fix --check
```

It prints the path of any item that is not already canonical and exits
non-zero — the form you would run in CI. Drop `--check` to apply the fix in
place.

## Next steps

- [Configure checks for a collection]({{< relref "../how-to/configure-rules.md" >}})
- [Add a schema]({{< relref "../how-to/add-a-schema.md" >}})
- [Validate in CI]({{< relref "../how-to/validate-in-ci.md" >}})
- [Configuration reference]({{< relref "../reference/configuration.md" >}})
  and the [rule reference]({{< relref "../reference/rules/_index.md" >}})
