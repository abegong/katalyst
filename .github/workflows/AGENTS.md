# CI/CD workflows

Canonical description of how this repo builds, validates, previews, and
publishes — especially the **docs site**, whose publish path has been
confused before. The workflow files and `netlify.toml` point here instead of
each restating the setup, so there is one source of truth and nothing to drift.

## The docs site: three separate paths

The docs site has three jobs, owned by three different things. Keeping them
straight is the whole point of this file.

| Path | Owner | Trigger | What it does |
|---|---|---|---|
| **Publish** (production) | `deploy-docs.yml` | push to `main`, manual | Builds the site and deploys it to GitHub Pages. |
| **Preview** (per-PR) | `netlify.toml` (Netlify GitHub App) | every PR | Builds a throwaway Deploy Preview and posts the URL. **Not a GitHub Action.** |
| **Validate** (build/lint gate) | `docs` job in `ci.yml` | PRs + push to `main` | Builds the docs to catch broken refs and runs `katalyst check` (dogfood). **Never deploys.** |

The production site is **one** site (GitHub Pages). Per-PR previews need one
URL *per PR*, which a single Pages site can't provide — that is the only reason
previews live on Netlify, not because Netlify publishes production.

## The load-bearing invariant (read before touching `deploy-docs.yml`)

**GitHub Pages → Settings → Pages → Source is set to "GitHub Actions."**

Therefore the production deploy must go **through the Pages Actions pipeline** —
`actions/upload-pages-artifact` + `actions/deploy-pages` — which is what
`deploy-docs.yml` does. Consequences:

- There is **no `gh-pages` branch**. Do not reintroduce one; with this Source,
  a branch is never served.
- Do **not** swap back to a branch-push action (e.g.
  `JamesIves/github-pages-deploy-action`). It would run green and publish to a
  branch nothing reads. That is exactly the regression this setup replaced.
- If you ever change the Source back to "Deploy from a branch," this workflow
  must change with it (and you'd want a `.nojekyll` at the published root).

## Failure mode to recognize

**A green `deploy-docs` run does not prove the live site updated.** If the
published docs lag `main`, suspect a mismatch between the Pages **Source**
setting and the deploy **mechanism** first (Source = "GitHub Actions" but a
workflow pushing to a branch, or vice versa) — before chasing the Hugo build.
This is the silent divergence the current setup is designed to prevent.

## Shared build

All three paths build the same way, so output matches across them:

- `make docs-build` → `hugo --minify` (the local hugo when extended). The
  `deploy-docs` and `ci` jobs call it; Netlify runs the equivalent
  `hugo --gc --minify` from `netlify.toml`.
- Hugo is pinned to **0.163.3** everywhere (both workflows and `netlify.toml`);
  bump them together.
- The check-type/inspector **reference pages are generated** by `cmd/gendocs`
  (`make docs-gen`) and guarded by `make docs-gen-check` in the `ci` `test`
  job — so a stale generated page fails CI rather than reaching the site.

## Other workflows

- **`release.yml`** — on a `v*` tag, GoReleaser builds cross-platform binaries
  and publishes a GitHub Release. Unrelated to docs.

---

For *where documentation content lives* (Diátaxis tree, generated reference,
templates, style), see
[`docs/content/contributing/how-we-document.md`](../../docs/content/contributing/how-we-document.md).
This file is only about the **build/deploy machinery**.
