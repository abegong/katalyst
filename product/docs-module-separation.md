# Spec — separate the docs Hugo module from the Go module

> **Status: proposed.** Pick up on a dedicated branch. Independent of the
> v0 CLI rebuild. Goal: end the recurring CI red caused by the Hugo theme
> living in the application's `go.mod`.

## Problem

The Hugo Book docs theme (`github.com/alex-shpak/hugo-book`) is a **Hugo
module**, not a Go package — no `.go` file imports it. It is declared in:

- `hugo.yaml` → `theme:` and `module.imports`
- the application's `go.mod` / `go.sum` (as a plain require)

CI's `Tidy check` runs `go mod tidy` then `git diff --exit-code go.mod
go.sum`. Because nothing in Go code imports the theme, `go mod tidy`
**strips it every time**, so any commit that contains the theme in
`go.mod` fails the check. (This kept `main` red for weeks.)

The current mitigation (commit the tidied `go.mod` without the theme) is a
bandaid: `make docs-deps` re-runs `hugo mod get -u`, which re-adds the
theme locally, reintroducing the diff and the trap.

## Goal

The application's `go.mod` stays `go mod tidy`-clean **and** the docs theme
is permanently, correctly tracked — so neither `make docs-build` nor CI
ever produces an unexpected `go.mod` diff.

## Approach: give the docs their own module

Move the Hugo site into its own directory with its own `go.mod`, so the
theme is tracked there and never touches the application module.

```
katalyst/
  go.mod                 # application module — no Hugo deps
  docs/
    go.mod               # Hugo module: requires hugo-book
    go.sum
    hugo.yaml            # moved here (theme + module.imports)
    content/ ...         # the markdown that is `docs/` today
```

### Work items

1. **Create `docs/go.mod`** (`module github.com/katabase-ai/katalyst/docs`
   or a docs-only module path Hugo is happy with) and `go.sum`. Run
   `hugo mod get -u github.com/alex-shpak/hugo-book` from `docs/` so the
   theme is required there.
2. **Move `hugo.yaml`** into `docs/` and adjust `contentDir`/paths so the
   site root is `docs/`. Confirm `module.imports` still resolves the theme.
3. **Strip Hugo from the app module:** remove `hugo-book` from the root
   `go.mod`/`go.sum` (already done by the bandaid); confirm `go mod tidy`
   at the repo root is a no-op.
4. **Update the Makefile** `docs-deps`/`docs-serve`/`docs-build` targets to
   run Hugo with `docs/` as the working directory / source (e.g. `hugo -s
   docs` or `cd docs && hugo`).
5. **Update `ci.yml` if docs get a CI job:** any docs build/tidy step must
   run inside `docs/`. The root `Tidy check` is unchanged and now stable.
6. **Fix doc links / `baseURL`** as needed after the move; verify
   `make docs-build` produces the same site.

### Acceptance criteria

- [ ] `go mod tidy` at the repo root produces **no** diff (CI `Tidy check`
      green without special-casing).
- [ ] `make docs-build` succeeds and re-runs do **not** modify the root
      `go.mod`/`go.sum`.
- [ ] The theme is tracked in `docs/go.mod` (a fresh `hugo mod get` there
      is a no-op on a clean tree).
- [ ] GitHub Pages output (`baseURL`, content) is unchanged.

## Alternatives considered (and rejected)

- **Weaken the CI guard** (drop `git diff --exit-code`, or special-case
  `hugo-book`): removes the protection that catches a forgotten
  `go mod tidy` on real Go deps. A bandaid, not a fix.
- **`tools.go` blank import** to pin the theme: impossible — the theme has
  no importable Go package, so `import _ "...hugo-book"` won't compile.

## Out of scope

- Changing the docs theme or site content.
- The application CLI (this is purely build/module hygiene).
