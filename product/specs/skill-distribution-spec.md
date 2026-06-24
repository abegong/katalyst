# Skill distribution

> **Status: planning.** Commit a family of user-facing katalyst skills under
> `skills/`, versioned with the CLI, and ship each as a `.skill` attached to the
> existing GitHub Release. `make skills` packages them; the GoReleaser release
> uploads them; a shared bootstrap fetches the CLI at install.

## Overview

A *skill* — a `SKILL.md`, its references, and a bootstrap — teaches a
Claude/Cowork agent to drive the katalyst CLI for one job. Katalyst spans a
workflow: catalog the content you have, define its language and structure,
enforce that structure day to day, and reshape it as needs change
(`docs/content/welcome.md` names **Catalog**, **Define**, and **Reshape** as the
headline features; enforcement is the day-to-day use those set up). These skills
need a maintenance home that versions them with the CLI **and** a way for users
who never touch Git to install them. Those are separable: **where a skill is
versioned** and **how users obtain it** need not be the same place. This spec
commits the skills to the repo as the single source of truth and ships each as a
downloadable release artifact, so committing them never implies users need repo
access.

## Scope

In scope: **katalyst's own release cycle** — how skills are sourced, packaged,
and published through Channel 1 (the `.skill`-on-Releases download). Out of
scope:

- **The user's deployment cycle** — how an adopter wires katalyst into their
  environment (a pre-commit hook, a gate on writes to its content). That is skill
  *content* — what the **katalyst-deploy** cluster teaches — authored with those
  skills.
- **Channel 2, marketplace plugins** — recorded in Design as the future
  direction, not built here.
- **Skill ↔ CLI version coupling** — the bootstrap tracks the latest Release;
  pinning a skill to a CLI version is a later concern.

## Value

Committing the skills alongside the CLI keeps them in lockstep — a check rename
and the skill update that documents it land in one PR, reviewed together. But
"committed to the repo" reads as "clone the repo to get it," which is wrong for
the audience: these users run a Claude client, not `git`. Publishing a `.skill`
per skill to GitHub Releases breaks that false coupling. Maintainers get
versioned-with-the-tool authoring; users get a download link and a Settings
panel, and can install only the stage they need. One source feeds both channels,
so the committed-vs-plugin decision only ever changes install UX — never whether
repo access is required.

## Current State

- **No skills in the repo.** There is no `skills/` directory; the user-facing
  skills live outside this tree today. (Distinct from `.cursor/skills/`, which
  holds *contributor* agent skills — `write-spec`, `write-docs` — mirrored into
  `.claude/skills/` and `.codex/skills/` for people working *on* katalyst. The
  new skills are *product* artifacts for people *using* katalyst.)
- **A release pipeline already exists.** `.github/workflows/release.yml` fires
  on `v*` tags with `permissions: contents: write` and runs GoReleaser
  (`.goreleaser.yml`), which builds the cross-platform CLI matrix
  (linux/darwin/windows × amd64/arm64) and publishes a GitHub Release. It does
  **not** know about skills: it uploads only the binary archives and
  `checksums.txt`. (The `test`/`docs` jobs in `ci.yml` are a separate
  build/lint gate on PRs and `main`; they upload nothing.)
- **Release assets are archives, not raw binaries.** GoReleaser publishes
  `katalyst_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows) plus
  `checksums.txt`, so anything provisioning the CLI must download and unpack an
  archive, not a bare binary. `make build` still runs `go build -o bin/$(BINARY)
  .` for a single host-platform binary; `README.md` documents install via `go
  install github.com/abegong/katalyst@latest` or `make build` from source.
- **How-to guides are separate, human docs.** `docs/content/how-to/` holds task
  recipes for human readers, covering much the same task taxonomy the skills do
  (`add-a-schema`, `configure-rules`, `profile-an-existing-wiki-*`,
  `validate-in-ci`). Skills are **self-contained at runtime** (see Design): an
  agent gets everything it needs from the installed skill plus the CLI, with no
  dependency on the docs site being reachable. Relating the two editorially is a
  separate question, resolved in Design.
- **Skill symlinks have a precedent.** `scripts/setup-claude-code.sh` and
  `scripts/setup-codex-skills.sh` (both via `sync_skill_links_from_cursor` in
  `scripts/agent-link-utils.sh`) symlink each `.cursor/skills/*` into
  `.claude/skills/` and `.codex/skills/`. `.gitignore` excludes **all** of
  `.claude/` and `.codex/`, so those mirrors stay uncommitted with no per-path
  entry. This is the model the local-dev symlink reuses.

## Design

A family of skills, one committed source each; two delivery channels fed from
the same folders.

### The skill family

Skills track the workflow stages in `docs/content/welcome.md` — Catalog,
Define, Reshape — plus enforcement *setup* and an orientation skill that spans
them. Enforcement is not a welcome.md headline stage; it is the day-to-day use
Define establishes and Reshape revises, set up once by the **katalyst-deploy**
cluster.

| Skill | Stage | What it teaches the agent to do |
|---|---|---|
| **katalyst-overview** | Orientation (all) | What katalyst is, its model and vocabulary (collections, items, schemas, checks), and which skill to reach for. The front door and router; does no task work itself. |
| **katalyst-catalog** | Catalog | Take stock of existing content in a specific knowledge base, surface its candidate collections, get oriented. |
| **katalyst-identify-collections** | Define (1 of 2) | Identify the collections — the recurring kinds of item the knowledge base is full of. Points to **katalyst-define-schemas** as the next step. |
| **katalyst-define-schemas** | Define (2 of 2) | Define each collection's schema — the fields its items must have and the constraints they must satisfy. Points back to **katalyst-identify-collections** as its prerequisite. |
| **katalyst-deploy** | Enforce (setup) | Set up automatic enforcement *once*. Knows **both** mechanisms, helps choose, and routes to the two specific skills below. |
| **katalyst-deploy-precommit-hook** | Enforce (setup) | Install a pre-commit hook that runs `katalyst check`, so violations are caught at commit time. |
| **katalyst-deploy-cli-gating** | Enforce (setup) | Gate writes to the project's content through the CLI, so writes are validated as they happen. |
| **katalyst-migrate-schema** | Reshape | *Placeholder — no content yet.* Migrate content when a collection's schema changes. |
| **katalyst-migrate-storage** | Reshape | *Placeholder — no content yet.* Migrate when the storage layer changes. |

Each is a separate job with a different agent posture, so each is a separate
skill — independently discoverable and installable. The set is additive:
packaging and the release take whatever shippable skills exist under `skills/`,
so stages can land one at a time without reworking the pipeline.

**Naming convention: every skill is `katalyst-`-prefixed.** Channel 1 installs
each `.skill` individually into one flat namespace in the client, with no
enclosing folder or plugin to group them — so generic names (`overview`,
`catalog`, `deploy`) would collide with unrelated skills and give the agent's
selection a weaker signal. A uniform prefix disambiguates, clusters the family
in any sorted list, matches the "use **katalyst**" phrasing, and yields clear
artifact names (`katalyst-deploy.skill`). The prefix is the shipped identity: it
is the `name` in each `SKILL.md`, the `.skill` artifact name, **and** the
directory under `skills/`, kept 1:1 so there is no dir→name mapping to drift.
(Chosen over a `-with-katalyst` suffix, which would scatter the skills under
their action letter instead of grouping them.)

The define stage is **two discrete skills**, not one: `katalyst-identify-collections`
(name the collections) precedes `katalyst-define-schemas` (formalize each
collection's fields and constraints). They **cross-reference** each other — identify points
forward, define-schemas points back — so the two-step flow is explicit without
merging two jobs an agent invokes at different times into one skill.

### Orientation: the `katalyst-overview` skill

`katalyst-overview` is the family's front door. It carries katalyst's mental
model and vocabulary — collections, items, schemas, checks, the workflow — and
routes an agent to the right task skill for the goal at hand. It does no task
work itself, which keeps it distinct from `katalyst-catalog`: that one takes
stock of a *specific* knowledge base, while `katalyst-overview` explains
katalyst-the-tool independent of any repo. Broadly triggered and a candidate to
install by default, it is how an agent learns katalyst exists and which skill to
load — the same "don't make the agent guess" concern the deploy cluster
addresses, met at the discovery layer.

### Enforcement is deployed, not invoked

Enforcement setup is a **cluster**, not a runbook the agent re-runs on every
write. Relying on an agent to *choose* to run `check`/`fix` each time is fragile
— the guardrail only holds when it is structural. `katalyst-deploy` is the
umbrella skill: it knows **both** mechanisms, helps pick between them, and routes
to the specific skill. `katalyst-deploy-precommit-hook` installs a pre-commit
hook that runs `katalyst check`; `katalyst-deploy-cli-gating` gates write access
to the project's content through the CLI. Either way enforcement is set up once
and then runs automatically, no matter which agent — or human — does the writing;
the day-to-day loop needs no skill of its own. The three cross-reference each
other, the same way the two define skills do. (How each mechanism is wired is the
skills' *content*, out of scope per [Scope](#scope).)

### Reshape: placeholders

The Reshape stage is two **placeholder** skills with no content yet:
`katalyst-migrate-schema` (content migration when a collection's schema changes)
and `katalyst-migrate-storage` (when the storage layer changes). Committed to
reserve the names and capture intent, they are marked `status: placeholder` in
their front matter and **excluded from packaging/release** until they carry real
content — so the additive pipeline already knows about them without shipping
empty skills.

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
tree is shipped product. Naming them apart keeps the two audiences from colliding
in one folder.

### Relationship to the how-to guides

Skills and the how-to guides under `docs/content/how-to/` are **parallel
coverage of one task taxonomy for two audiences** — an agent driving the CLI and
a human reading recipes — not one source generated from the other and not
strangers. The mapping is already close to one-to-one:

| How-to guide | Matching skill |
|---|---|
| `profile-an-existing-wiki-by-hand` / `…-with-an-agent` | **katalyst-catalog** |
| `add-a-schema` | **katalyst-define-schemas** |
| `configure-rules` | **katalyst-define-schemas** (its checks) |
| `validate-in-ci` | **katalyst-deploy** cluster |

The relationship has three parts:

- **Shared taxonomy and vocabulary.** Both organize around the lifecycle stages
  and use the glossary's terms, so a task is never named one way for humans and
  another for agents. Kept consistent by review — a change to a task's CLI
  surface updates both the guide and the skill in the same PR — **not** by
  generating one from the other.
- **Bidirectional cross-links at the human-facing edges.** Each how-to guide with
  a matching skill points to it ("prefer to have an agent do this? install
  `katalyst-<x>`," with the download link); each skill's *human-facing* front
  matter points back to the guide for the manual path. Discovery flows both ways.
- **Runtime self-containment is preserved.** The cross-link is editorial metadata,
  not a runtime dependency: a skill's executable `SKILL.md` body still carries
  everything the agent needs, so it functions with the docs site unreachable. The
  back-link is for a human reading the skill, never a fetch the agent performs.

This is the deliberate middle ground between two extremes the spec rejects:
generating skills from the guides (couples agent behavior to human docs and to a
reachable docs site) and treating them as unrelated (drift, and a worse
experience for a user who meets one and not the other).

### Channel 1 (now): `.skill` on GitHub Releases

A `.skill` is a zip of a skill directory with `SKILL.md` at its root. `make
skills` produces one `.skill` per shippable skill under `skills/` (e.g.
`katalyst-deploy.skill`, `katalyst-catalog.skill`); the release attaches them to
the GitHub Release for each tag, beside the CLI archives. Users download the
skill(s) they want from the releases page and install through the client's
**Settings → Capabilities → "Save skill."** No clone, no Git.

### Channel 2 (out of scope): marketplace plugins

Recorded as the future direction, not built here. When one-click install is
wanted, wrap the **same** `skills/{name}/` folders in marketplace plugins users
add by URL. The plugin contains the folder verbatim, so this is additive — no
rework of the source, and the `.skill` downloads keep working alongside it. The
single committed source is what makes Channel 2 a later add rather than a
rewrite; its shape (one bundle plugin vs. one plugin per skill), timing, and
ownership are deferred with it.

### Packaging: `make skills`

A `skills` target in the `Makefile` zips each `skills/{name}/` to `{name}.skill`
with `SKILL.md` at the archive root (not nested under a `{name}/` prefix — the
client expects `SKILL.md` at the top). It **skips skills marked `status:
placeholder`** in their `SKILL.md` front matter, so stubs like
`katalyst-migrate-schema` and `katalyst-migrate-storage` are never shipped. It is
the single packaging entry point, reused by the release so local and CI packaging
are identical. A `make skill SKILL=katalyst-deploy` form packages one. `make
clean` removes the `.skill` artifacts alongside `bin/`.

### Release cycle

The tag-triggered release already exists: `.github/workflows/release.yml` runs
GoReleaser on every `v*` tag and publishes the cross-platform binary archives.
This spec **extends that existing release** rather than adding a parallel
workflow:

1. The GoReleaser build matrix already produces the cross-platform CLI archives
   — no new GOOS/GOARCH work.
2. Run `make skills` (a GoReleaser `before` hook) to package every shippable
   skill under `skills/` before the release publishes.
3. Attach all `.skill` files to the same Release as extra assets (GoReleaser's
   `release.extra_files`), alongside the binary archives and `checksums.txt`.

Per-PR CI is unchanged: it builds and tests the CLI. The skills are plain files
versioned in the same repo, so the existing review of a PR is what keeps a skill
current with the CLI it drives — no separate gate.

### Local dev: symlink, uncommitted

A `make` target symlinks each `skills/{name}/` into `.claude/skills/` so they
auto-load in a working copy, following the `sync_skill_links_from_cursor` pattern
already in `scripts/`. `.gitignore` already excludes all of `.claude/` (and
`.codex/`), so the symlinks stay uncommitted with no new ignore entry.

### Binary provisioning: the shared bootstrap

The skills' shared bootstrap installs or locates the CLI by **fetching the CLI
archive from the latest GitHub Release and unpacking it** (falling back to `go
install github.com/abegong/katalyst@latest`), not by embedding binaries in each
`.skill`. Because the Release ships `katalyst_<version>_<os>_<arch>.tar.gz`
(`.zip` on Windows), the bootstrap detects OS/arch, downloads the matching
archive, and extracts the `katalyst` binary — it does not fetch a bare binary.
Embedding would bloat every download and risk a skill whose bundled binary lags
the Release; fetching keeps each `.skill` small and the binary current. One
bootstrap serves the whole family.

Tracking "latest" sidesteps version coupling for now (it's out of scope): the
bootstrap pulls whatever the newest Release ships. Pinning a skill to a specific
CLI version is a later concern, taken up when releases are cut often enough for
skew to bite.

## Open Questions

_None._ The decisions that were open — fetch-vs-embed, define as two skills,
enforce as a cluster, the orientation skill, the `katalyst-` prefix, reshape as
placeholders, Channel 1 first, and deferred version coupling — are folded into
Design above; the paths not taken are in [Rejected alternatives](#rejected-alternatives).

## Documentation updates

Land with the work, not after (see `docs/contributing/how-we-document.md`):

- **Root `AGENTS.md`** — add `skills/` to the Layout section and a one-line
  convention: product skills live there, `katalyst-`-prefixed, 1:1 dir↔name;
  contributor skills stay in `.cursor/skills/`. Point at the distribution
  deep-dive for the *why*.
- **`docs/deep-dives/`** — graduate the locked rationale (one committed source,
  fetch-don't-embed, the lifecycle skill family and `katalyst-` naming, deploy
  as setup, Channel 1 before Channel 2) into a distribution page at **done**.
  `vision.md` and `core-concepts.md` already frame the skills/tools split; this
  page explains how skills are shipped.
- **`docs/content/how-to/`** — add the cross-links: each guide with a matching
  skill (`add-a-schema`, `configure-rules`, `profile-an-existing-wiki-*`,
  `validate-in-ci`) gains a pointer to its skill and the download. The back-links
  live in each skill's front matter, authored with the skill.
- **`docs/reference/glossary.md`** — add *skill*, *`.skill`*, *bootstrap*, and
  *channel* as defined here.
- **`README.md`** — point the install section at the skills download alongside
  the CLI install.
- **Go doc comments** — none; the change is build machinery (Makefile,
  GoReleaser, a bootstrap script), not new Go packages.
- **`.cursor/skills/`** — no change; this section exists only to record that the
  product skills are deliberately *not* added there.

## Rejected alternatives

- **One mega-skill for the whole lifecycle.** Buries most of the workflow behind
  whichever job the `SKILL.md` leads with, and forces users to install
  catalog/define machinery just to set up enforcement. A per-stage family matches
  the "tools and skills" framing in `welcome.md` and lets users install only what
  they need.
- **Keep the skills outside the repo (own repo or gists).** Decouples each skill
  from the CLI it documents; a check rename and its skill update would land in
  separate PRs with no shared review — exactly the drift this design prevents.
- **Generate skills from the how-to guides (one procedural source).** Couples
  agent behavior to human docs and to the docs site being reachable at runtime;
  the two audiences diverge. Skills stay self-contained at runtime instead —
  related to the guides by a shared taxonomy and cross-links, not generation (see
  [Relationship to the how-to guides](#relationship-to-the-how-to-guides)).
- **Ship only marketplace plugins, skip the `.skill` download.** Defers all
  distribution behind unsettled plugin infrastructure and ownership. The `.skill`
  download works today against plain GitHub Releases and the client's existing
  "Save skill" flow, so it is Channel 1 and the marketplace is a later add.
- **Commit the skills under `.cursor/skills/` with the contributor skills.**
  Conflates two audiences in one tree and pulls shipped artifacts into the local
  agent-sync machinery. A separate top-level `skills/` keeps product and
  contributor tooling distinct.

## Test checklist (what the build contract asserts)

- [ ] `make skills` produces one `{name}.skill` per **shippable** directory
      under `skills/`, each unzipping with `SKILL.md` at the archive root, and
      emits no artifact for `status: placeholder` skills.
- [ ] `make skill SKILL=<name>` packages a single skill.
- [ ] `make clean` removes the `.skill` artifacts.
- [ ] The local-dev target symlinks each `skills/{name}/` into `.claude/skills/`,
      and the symlinks are git-ignored.
- [ ] On a `v*` tag, the release uploads the cross-platform CLI archives **and**
      every shippable `.skill` as assets on that Release.
- [ ] A `.skill` downloaded from a Release installs via the client's "Save skill"
      flow with no repo clone, and its bootstrap fetches and unpacks the matching
      CLI archive (with `go install` fallback).
