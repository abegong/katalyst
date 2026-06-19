# Spec — separate the docs Hugo module from the Go module

> **Status: planning.** A plan exists
> ([`docs-module-separation-plan.md`](docs-module-separation-plan.md));
> implementation not yet started. Independent of the v0 CLI rebuild.

## Problem

The Hugo Book docs theme (`github.com/alex-shpak/hugo-book`) is a **Hugo
module**, not a Go package — no `.go` file imports it. It is declared in:

- `hugo.yaml` → `theme:` and `module.imports`
- the application's `go.mod` / `go.sum` (as a plain require)

CI's `Tidy check` runs `go mod tidy` then `git diff --exit-code go.mod
go.sum`. Because nothing in Go code imports the theme, `go mod tidy`
**strips it every time**, so any commit that contains the theme in
`go.mod` fails the check. (This kept `main` red for weeks.)

The current mitigation — commit the tidied `go.mod` without the theme —
is a bandaid: `make docs-deps` re-runs `hugo mod get -u`, which re-adds the
theme locally, reintroducing the diff and the trap.

## Design

Give the Hugo site its **own module** so the theme is tracked where it
belongs and never touches the application's `go.mod`.

The constraint that forces the shape: Hugo runs `go` from the **site root**
(the directory holding `hugo.yaml`), so a Hugo module's `go.mod` must live
at the site root. Two `go.mod` files cannot share a directory, so the Hugo
site root must move out of the repo root into a subdirectory. Content
already lives under `docs/`, so `docs/` is the natural new site root.

### Target layout

```
katalyst/
  go.mod                 # application module — Hugo-free, tidy-clean
  go.sum
  Makefile               # docs targets run Hugo with -s docs
  docs/
    go.mod               # Hugo module — requires hugo-book
    go.sum
    hugo.yaml            # moved from repo root
    content/             # the markdown currently at docs/*.md
      _index.md
      getting-started.md
      rules/ ...
```

### Architecture impact

- **Two modules, one repo.** The root Go module builds the CLI; the
  `docs/` Hugo module builds the site. They share no dependency graph.
- The root `Tidy check` becomes permanently stable — `go mod tidy` at the
  root is a no-op because the only thing that ever made it dirty (the
  theme) is gone for good.
- No application code, package layout, or CLI behavior changes. This is
  pure build/module hygiene.

### Key decisions

- Hugo site root → `docs/`; the markdown currently at `docs/*.md` moves
  under `docs/content/` (Hugo's default `contentDir`).
- Docs module path: `github.com/katabase-ai/katalyst/docs` (any valid
  module path works; this one is conventional and self-documenting).
- `baseURL` and published-site structure stay identical; only the *source*
  tree moves.

## Open questions

1. **Content restructure vs. module mounts.** Recommended: physically move
   `docs/*.md` → `docs/content/` and let Hugo use its default `contentDir`.
   Alternative: keep content flat and use Hugo `module.mounts` to mount it
   as content. The move is simpler and more conventional; mounts avoid a
   large file rename but add config indirection. *Leaning: move.*
2. **Internal doc links.** After the move, audit relative links and any
   `relref`/`ref` shortcodes that assume the old paths. Mostly mechanical;
   flagged so it isn't forgotten.
3. **Pages deploy.** There is no committed Pages/deploy workflow today
   (docs build locally via `make docs-build`). If one is added later it
   must run inside `docs/`; out of scope for this change.

## Rejected alternatives

- **Weaken the CI guard** (drop `git diff --exit-code`, or special-case
  `hugo-book`): removes the protection that catches a forgotten
  `go mod tidy` on real Go deps — it's what caught the `doublestar`
  promotion. A different bandaid, not a fix.
- **`tools.go` blank import** to pin the theme: impossible — the theme has
  no importable Go package, so `import _ "...hugo-book"` won't compile.

## Out of scope

- Changing the docs theme, site content, or `baseURL`.
- The application CLI (purely build/module hygiene).
- Adding a Pages/CI deploy workflow for the docs.
