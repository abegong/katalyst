# Skill distribution

> **Status: planning.** A family of user-facing katalyst skills across the
> content lifecycle — **katalyst-overview** (orientation/router), **katalyst-catalog**, **define**
> (two cross-referencing skills, `katalyst-identify-collections` and `katalyst-define-schemas`),
> and a **katalyst-deploy** cluster (`katalyst-deploy` plus `katalyst-deploy-precommit-hook` and
> `katalyst-deploy-cli-gating`, setting up automatic enforcement) — committed under
> `skills/` and versioned with the CLI. Two Reshape-stage placeholders
> (`katalyst-migrate-schema`, `katalyst-migrate-storage`) are committed as stubs but excluded from
> release until they have content. In scope: package each shippable skill as a
> `.skill` and attach it to GitHub Releases alongside cross-platform binaries,
> which the skills' shared bootstrap fetches at install. `make skills` packages
> them; a tag-triggered release workflow uploads them. Out of scope: marketplace
> plugins (a later channel) and skill↔CLI version coupling.

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
sourced, packaged, and published through Channel 1 (the `.skill`-on-Releases
download). Out of scope:

- **The user's deployment cycle** — how an adopter wires katalyst into their
  environment (a pre-commit hook, a gate on directory access). That is skill
  *content* — what the **katalyst-deploy** cluster teaches — authored with those skills.
- **Channel 2, marketplace plugins** — recorded below as the future direction,
  but not built here.
- **Skill ↔ CLI version coupling** — the bootstrap tracks the latest Release;
  pinning a skill to a CLI version is a later concern.

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

Skills track the lifecycle stages in "Why Katalyst," plus an orientation skill
that spans them:

| Skill | Lifecycle stage | What it teaches the agent to do |
|---|---|---|
| **katalyst-overview** | Orientation (all) | What katalyst is, its model and vocabulary (collections, items, schemas, checks), and which skill to reach for. The front door and router; does no task work itself. |
| **katalyst-catalog** | Catalog | Take stock of existing content in a specific knowledge base, map the main concepts, get oriented. |
| **katalyst-identify-collections** | Define (1 of 2) | Identify the collections — the object types the knowledge base has repeatable instances of. Points to **katalyst-define-schemas** as the next step. |
| **katalyst-define-schemas** | Define (2 of 2) | Define each collection's schema — the properties and invariants of its items. Points back to **katalyst-identify-collections** as its prerequisite. |
| **katalyst-deploy** | Enforce | Set up automatic enforcement *once*. Knows **both** mechanisms, helps choose, and routes to the two specific skills below. |
| **katalyst-deploy-precommit-hook** | Enforce | Install a pre-commit hook that runs `katalyst check`, so violations are caught at commit time. |
| **katalyst-deploy-cli-gating** | Enforce | Gate write access to the content directory through the CLI, so writes are validated as they happen. |
| **katalyst-migrate-schema** | Reshape | *Placeholder — no content yet.* Migrate content when a collection's schema changes. |
| **katalyst-migrate-storage** | Reshape | *Placeholder — no content yet.* Migrate when the storage layer changes. |

Each is a separate job with a different agent posture, so each is a separate
skill — independently discoverable and installable. The set is additive:
packaging and the release workflow take whatever shippable skills exist under
`skills/`, so stages can land one at a time without reworking the pipeline.

**Naming convention: every skill is `katalyst-`-prefixed.** Channel 1 installs
each `.skill` individually into one flat namespace in the client, with no
enclosing folder or plugin to group them — so generic names (`overview`,
`catalog`, `deploy`) would collide with unrelated skills and give the agent's
selection a weaker signal. A uniform prefix disambiguates, clusters the family
in any sorted list, matches the "use **katalyst**" phrasing, and yields clear
artifact names (`katalyst-deploy.skill`). The prefix is the shipped identity:
it is the `name` in each `SKILL.md`, the `.skill` artifact name, **and** the
directory under `skills/`, kept 1:1 so there is no dir→name mapping to drift.
(Chosen over a `-with-katalyst` suffix, which would scatter the skills under
their action letter instead of grouping them.)

The define stage is **two discrete skills**, not one: `katalyst-identify-collections`
(name the object types) precedes `katalyst-define-schemas` (formalize each type's
fields and invariants). They **cross-reference** each other — `katalyst-identify-collections`
points forward, `katalyst-define-schemas` points back — so the two-step flow is explicit
without merging two jobs an agent invokes at different times into one skill.

### Orientation: the `katalyst-overview` skill

`katalyst-overview` is the family's front door. It carries katalyst's mental model and
vocabulary — collections, items, schemas, checks, the lifecycle — and routes an
agent to the right task skill for the goal at hand. It does no task work itself,
which keeps it distinct from `katalyst-catalog`: that one takes stock of a *specific*
knowledge base, while `katalyst-overview` explains katalyst-the-tool independent of any
repo. Broadly triggered and a candidate to install by default, it is how an
agent learns katalyst exists and which skill to load — the same "don't make the
agent guess" concern the deploy cluster addresses, met at the discovery layer.

### Enforcement is deployed, not invoked

The Enforce stage is a **cluster**, not a runbook the agent re-runs on every
write. Relying on an agent to *choose* to run `check`/`fix` each time is fragile
— the guardrail only holds when it is structural. `katalyst-deploy` is the umbrella
skill: it knows **both** enforcement mechanisms, helps pick between them, and
routes to the specific skill. `katalyst-deploy-precommit-hook` installs a pre-commit hook
that runs `katalyst check`; `katalyst-deploy-cli-gating` gates write access to the
content directory through the CLI. Either way enforcement is set up once and
then runs automatically, no matter which agent — or human — does the writing;
the day-to-day loop needs no skill of its own. The three cross-reference each
other, the same way the two `define` skills do. (How each mechanism is wired is
the skills' *content*, out of scope per [Scope](#scope).)

### Reshape: placeholders

The Reshape stage is two **placeholder** skills with no content yet:
`katalyst-migrate-schema` (content migration when a collection's schema changes) and
`katalyst-migrate-storage` (when the storage layer changes). They replace the single
`reshape` skill from the earlier draft. Committed to reserve the names and
capture intent, they are marked `status: placeholder` in their front matter and
**excluded from packaging/release** until they carry real content — so the
additive pipeline already knows about them without shipping empty skills.

### Source of truth: `skills/{name}/`

Each skill lives at `skills/{name}/`, committed and versioned with the CLI:

```
skills/
  katalyst-overview/
    SKILL.md          # at the directory root — the .skill entrypoint
    references/        # supporting reference material the skill loads
  katalyst-catalog/
    SKILL.md
    references/
  katalyst-identify-collections/
    SKILL.md
    references/
  katalyst-define-schemas/
    SKILL.md
    references/
  katalyst-deploy/
    SKILL.md
    references/
  katalyst-deploy-precommit-hook/
    SKILL.md
    references/
  katalyst-deploy-cli-gating/
    SKILL.md
    references/
  katalyst-migrate-schema/        # placeholder (status: placeholder) — not shipped yet
    SKILL.md
  katalyst-migrate-storage/       # placeholder — not shipped yet
    SKILL.md
  bootstrap.…          # shared CLI provisioning, reused by every shipped skill
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
skills` produces one `.skill` per skill under `skills/` (e.g. `katalyst-deploy.skill`,
`katalyst-catalog.skill`); the release workflow attaches them to the GitHub Release for
each tag, beside the CLI binaries. Users download the skill(s) they want from
the releases page and install through the client's **Settings → Capabilities →
"Save skill"**. No clone, no Git.

### Channel 2 (out of scope): marketplace plugins

Recorded as the future direction, not built here. When one-click install is
wanted, wrap the **same** `skills/{name}/` folders in marketplace plugins users
add by URL. The plugin contains the folder verbatim, so this is additive — no
rework of the source, and the `.skill` downloads keep working alongside it. The
single committed source is what makes Channel 2 a later add rather than a
rewrite; its shape (one bundle plugin vs. one plugin per skill), timing, and
ownership are deferred with it.

### Packaging: `make skills`

A `skills` target in the `Makefile` zips each `skills/{name}/` to
`{name}.skill` with `SKILL.md` at the archive root (not nested under a
`{name}/` prefix — the client expects `SKILL.md` at the top). It **skips skills
marked `status: placeholder`** in their `SKILL.md` front matter, so stubs like
`katalyst-migrate-schema` and `katalyst-migrate-storage` are never shipped. It is the single
packaging entry point, reused by the release job so local and CI packaging are
identical. A `make skill SKILL=katalyst-deploy` form packages one. `make clean` removes
the `.skill` artifacts alongside `bin/`.

### Release cycle

A tag-triggered GitHub Actions job (e.g. `on: push: tags: ['v*']`), separate
from the existing `test` job in `ci.yml`:

1. Build the cross-platform CLI binaries — the current `go build -o bin/katalyst .`
   produces only the host platform, so this introduces the GOOS/GOARCH matrix.
2. Run `make skills` to package every shippable skill under `skills/`
   (placeholders excluded).
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

The skills' shared bootstrap installs or locates the CLI by **fetching the
binary from the latest GitHub Release** (falling back to `go install`), not by
embedding binaries in each `.skill`. Embedding would bloat every download and
risk a skill whose bundled binary lags the Release; fetching keeps each `.skill`
small and the binary current. One bootstrap serves the whole family.

Tracking "latest" sidesteps version coupling for now (it's out of scope): the
bootstrap pulls whatever the newest Release ships. Pinning a skill to a specific
CLI version is a later concern, taken up when releases are cut often enough for
skew to bite.

## Open Questions

_None — resolved or deferred._ For the record:

- **Fetch, don't embed.** The shared bootstrap fetches the CLI binary from the
  latest GitHub Release (falling back to `go install`); binaries are not bundled
  in the `.skill`.
- **`define` is two skills.** `katalyst-identify-collections` and `katalyst-define-schemas` are
  discrete, cross-referencing skills rather than one merged `define` skill.
- **Enforce is a cluster.** `katalyst-deploy` (umbrella, knows both mechanisms) plus
  `katalyst-deploy-precommit-hook` and `katalyst-deploy-cli-gating` (the specific setups), set up
  once rather than a loop the agent re-runs each write.
- **`katalyst-overview` orientation skill added.** A broadly-triggered front door that
  carries katalyst's model/vocabulary and routes to the task skills.
- **`katalyst-` prefix on every skill.** Uniform prefix (name + artifact +
  directory) to disambiguate and group the family in a flat skill namespace;
  chosen over a `-with-katalyst` suffix.
- **Reshape is two placeholders.** `katalyst-migrate-schema` and `katalyst-migrate-storage`
  replace the single `reshape` skill; committed as stubs (`status: placeholder`)
  and excluded from release until they have content.
- **Channel 1 only.** The `.skill`-on-Releases download is in scope; marketplace
  plugins (Channel 2) are deferred.
- **Versioning deferred.** Skill↔CLI version coupling is out of scope; the
  bootstrap tracks the latest Release.

## Rejected alternatives

- **One mega-skill for the whole lifecycle.** Buries most of the lifecycle
  behind whichever job the `SKILL.md` leads with, and forces users to install
  katalyst-catalog/define machinery just to set up enforcement. A per-stage family
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
  distribution behind unsettled plugin infrastructure and ownership. The
  `.skill` download works today against plain GitHub Releases and the client's
  existing "Save skill" flow, so it is Channel 1 and the marketplace is a later
  add.
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
