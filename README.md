# katalyst

Katalyst is a **content consistency layer**: schemas and checks that live with
your content and keep its structure steady as it grows. It's built for the
content AI agents manage — memory stores, wikis, knowledge bases. Today it
validates markdown frontmatter against JSON Schema and a registry of structural
checks, driven by a CLI. Inspired by [JSON Schema][js] and the
[MongoDB validation API][mv].

For the problems it solves, see [Why Katalyst](docs/content/why-katalyst.md);
for where it's headed, [Vision and scope](docs/content/deep-dives/vision.md).

[js]: https://json-schema.org/
[mv]: https://www.mongodb.com/docs/manual/core/schema-validation/

> **Status:** v0. The command surface is `init`, `check`, `fix`,
> `collection list/get`, `item list/get/add/update/delete`, and
> `schema list/show`.

## Install

```
go install github.com/katabase-ai/katalyst@latest
```

Or from source:

```
git clone https://github.com/katabase-ai/katalyst
cd katalyst
make build  # produces ./bin/katalyst
```

## Quickstart

```bash
mkdir my-notes && cd my-notes
katalyst init                  # prepares a .katalyst/ project directory
katalyst check                 # check every item in the project
```

The config is picked up automatically: every command discovers the nearest
`.katalyst/` directory walking up from the working directory, then resolves
**selectors** against the collections it declares.

## Documentation

Full docs are at the [Katalyst documentation site][docs] (source under
`docs/content/`):

- [Getting started](docs/content/getting-started.md) — install, scaffold a
  project, run your first checks.
- [How-to guides](docs/content/how-to/) — configure checks, add a schema,
  validate in CI.
- [Reference](docs/content/reference/) — selectors, the `.katalyst/`
  configuration surface, the command surface, and the generated rule reference.
- [Deep dives](docs/content/deep-dives/) — the vision, core concepts, and design
  rationale.

Contributing conventions — tests, fixtures, code style, and project layout —
live in [`AGENTS.md`](AGENTS.md).

[docs]: https://katabase-ai.github.io/katalyst/
