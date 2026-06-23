+++
title = "How to contribute"
weight = 5
+++

# How to contribute

Katalyst is an experimental, in-the-open project. Issues, feedback, and pull
requests are all welcome. This page covers getting set up and shipping a
change; for the project's planning and documentation process, see
[How we plan]({{< relref "how-we-plan.md" >}}) and
[How we document]({{< relref "how-we-document.md" >}}).

## Ground rules

- Be respectful. This project follows a
  [Code of Conduct]({{< relref "code-of-conduct.md" >}}).
- Open an issue to discuss substantial changes before investing in a large PR.
- Keep changes focused; smaller PRs are easier to review and merge.

## Where feedback goes

Put feedback in public artifacts so it stays searchable and actionable:

- Open an issue for bugs, feature requests, confusing docs, and questions.
- Open a PR when you have a concrete change ready to review.
- Use the [Security policy]({{< relref "security.md" >}}) for sensitive reports.

## Development setup

Katalyst is a Go (1.25+) CLI. Common tasks are driven by the `Makefile`:

```sh
make build   # build ./bin/katalyst
make test    # go test ./...
make vet     # go vet ./...
make fmt     # gofmt -w .
```

## Before you push

Mirror what CI runs so failures surface locally, not on the PR:

```sh
go mod tidy && git diff --exit-code go.mod go.sum
make vet
go test -race -count=1 ./...
make build
make docs-gen-check   # generated reference and mirrored docs are current
make docs-build       # docs site builds with no broken refs
```

## Working on the docs

The docs site is a Hugo project under `docs/` (its own `go.mod`). Preview it
locally with:

```sh
make docs-serve
```

Some pages are **generated** by `cmd/gendocs` and must not be edited by hand:

- the check-type reference under `docs/content/reference/check-types`,
- the inspector reference under `docs/content/reference/inspectors`, and
- the [Code of Conduct]({{< relref "code-of-conduct.md" >}}) and
  [Security policy]({{< relref "security.md" >}}) pages, mirrored from the
  repo-root files.

Run `make docs-gen` after changing a check type, an inspector, or a root
governance file, and commit the result. CI's `make docs-gen-check` fails if any
generated page is stale.

## Opening a pull request

- Ensure the commands under [Before you push](#before-you-push) pass locally.
- Update or add documentation for user-facing changes.
- Reference any related issue in the PR description.
