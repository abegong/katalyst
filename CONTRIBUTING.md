# Contributing to Katalyst

Thanks for your interest in Katalyst! It's an experimental, in-the-open
project, and feedback, issues, and pull requests are all welcome.

## Ground rules

- Be respectful. This project follows a [Code of Conduct](CODE_OF_CONDUCT.md).
- Open an issue to discuss substantial changes before investing in a large PR.
- Keep changes focused; smaller PRs are easier to review and merge.

## How we plan and document

Katalyst has a deliberate process for planning and documenting changes. Before
opening a non-trivial PR, please skim:

- [How we plan](https://abegong.github.io/katalyst/contributing/how-we-plan/) —
  specs and plans precede implementation.
- [How we document](https://abegong.github.io/katalyst/contributing/how-we-document/) —
  the docs follow the [Diátaxis](https://diataxis.fr) framework.

## Development setup

Katalyst is a Go (1.25+) CLI. Common tasks are driven by the `Makefile`:

```sh
make build   # build ./bin/katalyst
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
```

Before pushing, mirror what CI runs:

```sh
go mod tidy && git diff --exit-code go.mod go.sum
make vet
go test -race -count=1 ./...
make docs-gen-check   # regenerated check-type reference is current
make docs-build       # docs site builds with no broken refs
```

## Documentation

The docs site is a Hugo project under `docs/` (its own `go.mod`). Preview it
locally with:

```sh
make docs-serve
```

The check-type reference under `docs/content/reference/check-types` is
generated from the checks registry — run `make docs-gen` after adding or
changing a check type, and commit the result.

## Pull requests

- Ensure the commands above pass locally.
- Update or add documentation for user-facing changes.
- Reference any related issue in the PR description.
