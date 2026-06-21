# Skill distribution

> **Status: planning.** A family of user-facing katalyst skills — one per
> lifecycle stage (catalog, define, enforce, reshape) — committed under
> `skills/` and versioned with the CLI. Two delivery channels: a `.skill`
> artifact per skill attached to GitHub Releases for Git-free install today,
> and marketplace plugins wrapping the same folders later. `make skills`
> packages them; the tag-triggered release workflow builds the binaries and
> uploads every skill alongside them.

## Overview

Katalyst is a content-consistency layer that spans a lifecycle: catalog the
content you have, define its language and structure, enforce that structure
day to day, and reshape it as needs change (see
`docs/content/why-katalyst.md`). A
*skill* — a `SKILL.md`, its references, and a bootstrap — teaches a Claude/Cowork
agent to drive the CLI for one of those jobs. These skills need a maintenance
home that versions them with the CLI, and a way for users who never touch Git
to install them. Those are separable: **where a skill is versioned** and **how
users obtain it** do not have to be the same place. This spec commits the skills
to the repo as the single source of truth and ships each as a downloadable
release artifact, so committing them never implies users need repo access.

## Scope

This spec covers **katalyst's own release cycle** — how skills and binaries are
sourced, packaged, and published. It does **not** cover the user's deployment
cycle: how an adopter wires katalyst into their environment (a linter on every
agent write, a gate in their CI). That is skill *content* — what the **enforce**
skill teaches — and is authored with that skill, not here.

## Value

Committing the skills alongside the CLI is what keeps them in lockstep — a check
rename and the skill update that documents it land in one PR, reviewed together.
But "committed to the repo" reads as "clone the repo to get it," which is wrong
for the audience: these users run a Claude client, not `git`. Publishing a
`.skill` per skill to GitHub Releases breaks that false coupling. Maintainers
get versioned-with-the-tool authoring; users get a download link and a Settings
panel, and can install only the lifecycle stage they need. One source feeds both
channels, so the committed-vs-plugin decision only ever changes install UX —
never whether repo access is required.

## Current State

- **No skills in the repo.** There is no `skills/` directory; the user-facing
  skills live outside this tree today. (Distinct from `.cursor/skills/`, which
  holds *contributor* agent skills — `write-spec`, `write-docs` — mirrored into
  `.claude/skills/` and `.codex/skills/` for people working *on* katalyst. The
  new skills are *product* artifacts for people *using* katalyst.)
- **No release pipeline.** `.github/workflows/ci.yml` defines one `test` job on
  push to `main` and on PRs: tidy-check, vet, race tests, `make build`,
  `docs-gen-check`. There is no tag trigger, no cross-platform build, and
  nothing uploads release assets.
- **One local binary.** `make build` runs `go build -o bin/$(BINARY) .` — a
  single host-platform binary. `README.md` documents install via `go install
  github.com/katabase-ai/katalyst@latest` or `make build` from source. There is
  no packaging or multi-platform target.
- **How-to guides are separate, human docs.** `docs/content/how-to/` holds
  task recipes for human readers. Skills are independent of them (see Design):
  an agent gets everything it needs from the installed skill plus the CLI, with
  no dependency on the docs site.
- **Skill symlinks have a precedent.** `scripts/setup-claude-code.sh` (via
  `sync_skill_links_from_cursor` in `scripts/agent-link-utils.sh`) symlinks each
  `.cursor/skills/*` into `.claude/skills/`. `.gitignore` excludes
  `.claude/skills/` and `.codex/skills/`, so those mirrors stay uncommitted.
  This is the model the local-dev symlink reuses.

## Design

A family of skills, one committed source each; two delivery channels fed from
the same folders.

### The skill family

One skill per lifecycle stage, mirroring the "Why Katalyst" feature set:

| Skill | Lifecycle stage | What it teaches the agent to do |
|---|---|---|
| **catalog** | Catalog | Take stock of existing content, map the main concepts, get oriented. |
| **define** | Define | Two steps: **identify collections** (the object types the knowledge base has repeatable instances of), then **define their schemas** (the properties and invariants of items within a collection). |
| **enforce** | Enforce | Run the day-to-day loop — `check`, `fix`, `item` — to keep content in good shape on every write. |
| **reshape** | Reshape | Navigate change: add or change checks, restructure content, change the storage layer. |

Each is a separate job with a different agent posture, so each is a separate
skill — independently discoverable and installable. The set is additive:
packaging and the release workflow take whatever skills exist under `skills/`,
so stages can land one at a time without reworking the pipeline.

The **define** stage carries two distinct steps (identify collections, then
define schemas). Whether they are one `define` skill with two phases or two
discrete skills (`identify-collections`, `define-schemas`) is Q4.

### Source of truth: `skills/{name}/`

Each skill lives at `skills/{name}/`, committed and versioned with the CLI:

```
skills/
  enforce/
    SKILL.md          # at the directory root — the .skill entrypoint
    references/        # supporting reference material the skill loads
  define/
    SKILL.md
    references/
  …
  bootstrap.…          # shared CLI provisioning, reused by every skill
```

Maintainers edit them here. A change to the CLI surface and the skill text that
documents it land in the same PR, so a skill never drifts from the tool. This is
a **new top-level `skills/` directory**, deliberately separate from
`.cursor/skills/`: that tree is contributor tooling synced to local agents, this
tree is shipped product. Naming them apart keeps the two audiences from
colliding in one folder.

### Independent of the how-to guides

Skills are **self-contained**; they do not reference the how-to guides under
`docs/content/how-to/`. This diverges from treating the two as one procedural
source, on purpose: an agent must function without reaching the docs site at
runtime, and the two audiences (an agent driving the CLI vs. a human reading
recipes) diverge in what they need. The how-to guides and the skills may cover
the same tasks, but neither is generated from or depends on the other.

### Channel 1 (now): `.skill` on GitHub Releases

A `.skill` is a zip of a skill directory with `SKILL.md` at its root. `make
skills` produces one `.skill` per skill under `skills/` (e.g. `enforce.skill`,
`define.skill`); the release workflow attaches them to the GitHub Release for
each tag, beside the CLI binaries. Users download the skill(s) they want from
the releases page and install through the client's **Settings → Capabilities →
"Save skill"**. No clone, no Git.

### Channel 2 (later): marketplace plugins

When one-click install is wanted, wrap the **same** `skills/{name}/` folders in
marketplace plugins users add by URL. The plugin contains the folder verbatim,
so this is additive — no rework of the source, and the `.skill` downloads keep
working alongside it. Whether the family ships as one bundle plugin or one
plugin per skill, plus timing and ownership, is Q3.

### Packaging: `make skills`

A `skills` target in the `Makefile` zips each `skills/{name}/` to
`{name}.skill` with `SKILL.md` at the archive root (not nested under a
`{name}/` prefix — the client expects `SKILL.md` at the top). It is the single
packaging entry point, reused by the release job so local and CI packaging are
identical. A `make skill SKILL=enforce` form packages one. `make clean` removes
the `.skill` artifacts alongside `bin/`.

### Release cycle

A tag-triggered GitHub Actions job (e.g. `on: push: tags: ['v*']`), separate
from the existing `test` job in `ci.yml`:

1. Build the cross-platform CLI binaries — the current `go build -o bin/katalyst .`
   produces only the host platform, so this introduces the GOOS/GOARCH matrix.
2. Run `make skills` to package every skill under `skills/`.
3. Upload the binaries and all `.skill` files as assets on the Release for that
   tag, in one workflow.

Per-PR CI is unchanged: it builds and tests the CLI. The skills are plain files
versioned in the same repo, so the existing review of a PR is what keeps a skill
current with the CLI it drives — no separate gate.

### Local dev: symlink, uncommitted

A `make` target symlinks each `skills/{name}/` into `.claude/skills/` so they
auto-load in a working copy, following the `sync_skill_links_from_cursor`
pattern already in `scripts/`. `.gitignore` excludes `.claude/skills/`
specifically (not all of `.claude/`), so the symlinks stay uncommitted.

### Binary provisioning: the shared bootstrap

The skills' bootstrap installs or locates the CLI. Once binaries are published
as release assets, the bootstrap should prefer fetching the matching binary from
the Release (or `go install`) over embedding binaries in each `.skill`:
embedding bloats every download and risks a skill whose bundled binary lags the
Release. One shared bootstrap serves the whole family. The embed-vs-fetch
decision is Q1.

### Version coupling

A downloaded skill should line up with the CLI it documents. Stamp each
published skill with the CLI version, or declare a compatibility range the
bootstrap checks. Exact policy is Q2.

## Open Questions

1. **Embed binaries in the `.skill`, or fetch from the Release?** Lean: fetch
   (or `go install`) to keep each `.skill` small and the binary matched to the
   Release. Confirm the bootstrap can reliably reach the Release in the target
   client environments; if not, a bundled fallback may be unavoidable.
2. **Skill ↔ CLI version coupling policy.** Pin each skill to one CLI version,
   or declare a supported range? Where is the version stamped (a key in
   `SKILL.md`, a file in `references/`, the artifact name), and does the
   bootstrap enforce it or only warn on mismatch?
3. **Marketplace shape, timing, and ownership.** One bundle plugin for the
   family or one plugin per skill? When does Channel 2 land, and who owns the
   plugin repo/listing? Additive, so it does not block Channel 1.
4. **Is `define` one skill or two?** One skill with two phases (identify
   collections, then define schemas), or two discrete skills. Affects only the
   `skills/` layout and the artifact list, not the pipeline.

## Rejected alternatives

- **One mega-skill for the whole lifecycle.** Buries three of the four jobs
  behind whichever the `SKILL.md` leads with, and forces users to install
  catalog/reshape machinery to get day-to-day enforcement. A per-stage family
  matches the "tools and skills" framing in Why Katalyst and lets users install
  only what they need.
- **Keep the skills outside the repo (own repo or gists).** Decouples each skill
  from the CLI it documents; a check rename and its skill update would land in
  separate PRs with no shared review, which is exactly the drift this design
  prevents.
- **Generate skills from the how-to guides (one procedural source).** Couples
  agent behavior to human docs and to the docs site being reachable at runtime;
  the two audiences diverge. Skills stay self-contained instead.
- **Ship only marketplace plugins, skip the `.skill` download.** Defers all
  distribution behind plugin infrastructure and ownership that isn't settled
  (Q3). The `.skill` download works today against plain GitHub Releases and the
  client's existing "Save skill" flow.
- **Commit the skills under `.cursor/skills/` with the contributor skills.**
  Conflates two audiences in one tree and pulls shipped artifacts into the local
  agent-sync machinery. A separate top-level `skills/` keeps product and
  contributor tooling distinct.

## Test checklist (what the build contract asserts)

- [ ] `make skills` produces one `{name}.skill` per directory under `skills/`,
      each with `SKILL.md` at the archive root.
- [ ] `make skill SKILL=<name>` packages a single skill.
- [ ] `make clean` removes the `.skill` artifacts.
- [ ] The local-dev target symlinks each `skills/{name}/` into `.claude/skills/`,
      and the symlinks are git-ignored.
- [ ] On a tag, the release workflow uploads the cross-platform binaries and
      every `.skill` as assets on that Release.
- [ ] A `.skill` downloaded from a Release installs via the client's "Save skill"
      flow with no repo clone.
