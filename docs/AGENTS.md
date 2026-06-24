# docs

The Hugo documentation site. User and contributor pages live under
`content/`; generated CLI/example output lives under `generated/`; Hugo
configuration and theme module state live in this directory, separate from the
root Go module.

Architecture and rationale for the docs contract live in
[`content/contributing/how-we-document.md`](content/contributing/how-we-document.md)
and the root [`AGENTS.md`](../AGENTS.md). This file keeps only local
conventions.

## Conventions

- Keep the Hugo module isolated here. Do not add the docs theme or Hugo module
  dependencies to the root `go.mod`; use `make docs-deps` when the docs module
  needs dependency work.
- Hand-authored pages belong under `content/`. Reference pages generated from
  registries should be regenerated through the existing commands instead of
  edited in place.
- Katalyst dogfoods this tree through the repo-root `.katalyst/` project. After
  changing frontmatter or generated docs, run `make build && ./bin/katalyst
  check` so the page schema contract is checked the same way CI checks it.
- Keep documentation task-oriented and literal. Put long design rationale in
  `content/deep-dives/`; keep reference pages stable and command-shaped.
