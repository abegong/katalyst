# Plan — separate the docs Hugo module from the Go module

> **Status: implementing.** Implements
> [`docs-module-separation.md`](docs-module-separation.md). Phases 1–4 done;
> Phase 5 verification in progress.

Branch: `claude/docs-module-separation` (off `main`).

## Strategy

Move the Hugo site root from the repo root into `docs/`, give it its own
`go.mod`, and remove every trace of the Hugo theme from the application's
`go.mod`/`go.sum`. The root Go module ends up Hugo-free and permanently
`go mod tidy`-clean. No application code changes.

The risk to manage is the **content move** (`docs/*.md` → `docs/content/`):
it touches every doc file and any internal links between them. Do it as one
mechanical phase, verified by a successful `hugo build`.

## Phases

### Phase 1 — Stand up the docs module (no content move yet)

1. Create `docs/go.mod` with module path
   `github.com/abegong/katalyst/docs` (`go mod init` in `docs/`, or
   `hugo mod init`).
2. Move `hugo.yaml` from the repo root to `docs/hugo.yaml`.
3. From `docs/`, run `hugo mod get -u github.com/alex-shpak/hugo-book` so
   the theme is required in `docs/go.mod` / `docs/go.sum`.
4. Adjust `hugo.yaml`: with `docs/` as the site root, `contentDir` becomes
   `content` (default). Leave `baseURL`, theme, params unchanged.

### Phase 2 — Move the content under the new site root

1. `git mv` the current `docs/*.md` and `docs/rules/` tree into
   `docs/content/` (preserving the section layout, incl. `_index.md`).
2. Audit internal links and `ref`/`relref` shortcodes for paths broken by
   the move; fix them.
3. Confirm `_index.md` section structure still produces the same nav.

### Phase 3 — Purge Hugo from the application module

1. Ensure the root `go.mod` / `go.sum` contain **no** `hugo-book` entry
   (already true after the Option-1 tidy on the v0 branch; re-verify on
   this branch's base).
2. `go mod tidy` at the repo root must be a no-op.

### Phase 4 — Rewire the build

1. Update the Makefile docs targets to run Hugo with `docs/` as the source
   root: `hugo -s docs ...` (or `cd docs && hugo ...`). Keep `docs-deps`
   pointing the `hugo mod get` at the `docs/` module.
2. Update `README.md` "Documentation site" section and any `AGENTS.md`
   layout note to reflect the new `docs/` module + `docs/content/` source.

### Phase 5 — Verify and graduate

1. Run the verification checklist below.
2. Per [`how-we-plan.md`](../how-we-plan.md): on **done**, graduate durable
   content — record the two-module layout decision in
   [`decisions.md`](../decisions.md) (new D-number), update the `product/`
   architecture/layout notes, then retire this spec + plan.

## Verification checklist

These are the assertions that prove the change; they stand in for the
"failing tests" of a code change.

- [x] `go mod tidy` at the repo root produces **no** diff
      (`git diff --exit-code go.mod go.sum` clean) — i.e. CI `Tidy check`
      passes without special-casing.
- [x] Repo-root `go.mod`/`go.sum` contain no `alex-shpak/hugo-book`.
- [x] `docs/go.mod` requires `alex-shpak/hugo-book`; `go mod verify` in
      `docs/` passes (theme has no transitive Go deps).
- [x] Docs build succeeds **and** leaves the repo-root `go.mod`/`go.sum`
      unmodified afterward (verified with `hugo -s docs --minify`; root
      `git diff` clean post-build).
- [x] Build re-runs are idempotent (no tree changes; `docs/go.*` stable).
- [x] Built site renders (39 pages, 31 HTML; theme SCSS compiled;
      `getting-started/` and `rules/objects/object/` present).
- [x] `make all` (vet + test + build) still green.
- [ ] CI green on the PR (all four `ci.yml` steps) — pending push.

## Deviations

- **Phase 3 was a no-op.** This branch's base (`main`) already had the
  theme stripped from the root `go.mod` (the earlier bandaid) and
  `doublestar` as a direct require, so no root-module change was needed —
  only confirming `go mod tidy` is clean.
- **`docs/go.sum` generated via `go mod download`**, not `hugo mod get`:
  no `hugo` binary is installed in this environment. Equivalent result for
  a theme with no transitive Go deps; `hugo mod get` would produce the same
  two `go.sum` lines.
- **Build verified with `go run … hugo@latest -tags extended`** (Hugo
  v0.163.3 extended) since there's no local Hugo. Pre-existing, unrelated
  deprecation warning surfaced: `languageCode` → `locale` (Hugo v0.158+);
  left for a separate docs touch-up.
