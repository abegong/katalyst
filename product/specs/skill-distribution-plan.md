# Plan — skill distribution

> **Status: planning.** Implements
> [`skill-distribution-spec.md`](skill-distribution-spec.md). No phases started.

Branch: `claude/practical-mayer-uuo45q` (PR
[#20](https://github.com/katabase-ai/katalyst/pull/20)).

## Strategy

Build the distribution machinery first, then fill the skills into it. The
pipeline (packaging → release upload → install via bootstrap) is the part with
real failure modes — archive layout, asset upload, binary fetch — so it gets
stood up and verified against the **deploy cluster** (the most concrete,
highest-value skills) before the rest of the family is authored in content. The
skill *content* is deliberately the last phase: it is the largest writing
effort and the least coupled to the build contract, and it is the part most
likely to keep changing after this branch.

The initial **shipping** family is seven skills — `katalyst-overview`, `katalyst-catalog`,
`katalyst-identify-collections`, `katalyst-define-schemas`, `katalyst-deploy`, `katalyst-deploy-precommit-hook`,
`katalyst-deploy-cli-gating`. Two Reshape-stage **placeholders**, `katalyst-migrate-schema` and
`katalyst-migrate-storage`, are committed as stubs (`status: placeholder`) to reserve the
names but are excluded from packaging until they have content.

The build contract is the spec's **test checklist**; each phase below lands the
piece that makes one or more of those boxes pass. Per `AGENTS.md` (behavior
arrives with a failing test), the build assertions are scaffolded as failing
checks before the machinery exists — here that means a script/CI step that
asserts the artifact shape and fails until the Makefile target produces it.

Two things this plan does **not** do (out of scope per the spec): build Channel 2
marketplace plugins, and implement skill↔CLI version coupling. The bootstrap
tracks the latest Release.

## Phases

### Phase 1 — Scaffold `skills/` and the pilot cluster

1. Create the top-level `skills/` directory with the seven shipping skill
   folders — `katalyst-overview/`, `katalyst-catalog/`, `katalyst-identify-collections/`, `katalyst-define-schemas/`,
   `katalyst-deploy/`, `katalyst-deploy-precommit-hook/`, `katalyst-deploy-cli-gating/` — each with a
   `SKILL.md` at its root and a `references/` dir. Add the two Reshape
   placeholders `katalyst-migrate-schema/` and `katalyst-migrate-storage/` with a `SKILL.md`
   carrying `status: placeholder` in its front matter and no real content.
   Every `SKILL.md` `name` matches its `katalyst-`-prefixed directory 1:1 (per
   the spec's naming convention), so the artifact name follows automatically.
2. Author the **deploy cluster** for real as the pilot that exercises the whole
   pipeline: `katalyst-deploy` (umbrella — knows both mechanisms, routes to the two
   specifics) plus `katalyst-deploy-precommit-hook` (pre-commit hook running `katalyst
   check`) and `katalyst-deploy-cli-gating` (CLI-gated directory access), cross-referencing
   each other. Leave `katalyst-overview`, `katalyst-catalog`, `katalyst-identify-collections`, and
   `katalyst-define-schemas` as front-matter-only stubs to be filled in Phase 6.
3. Add the shared `bootstrap` at `skills/bootstrap.…` as a placeholder that
   Phase 3 fleshes out.
4. Confirm placement is distinct from `.cursor/skills/` (contributor tooling)
   and that nothing here is synced by `scripts/agent-link-utils.sh`.

### Phase 2 — Packaging (`make skills`)

1. Add a `skills` target to the `Makefile` that zips each `skills/{name}/` to
   `{name}.skill` with `SKILL.md` at the **archive root** (no `{name}/`
   prefix). Exclude `skills/bootstrap.…` from being its own artifact if it
   sits at the top level — package it *inside* each `.skill` instead (decide in
   Phase 3 whether the bootstrap is copied into each skill dir at package time
   or committed into each).
2. Make `make skills` **skip** any skill whose `SKILL.md` front matter is
   `status: placeholder`, so `katalyst-migrate-schema`/`katalyst-migrate-storage` are not shipped.
   (`make skill SKILL=<name>` on a placeholder should error or no-op clearly.)
3. Add `make skill SKILL=<name>` to package a single skill.
4. Extend `clean` to remove the `*.skill` artifacts alongside `bin/`.
5. Add the new targets to `.PHONY` and document them briefly in the Makefile
   and `README.md`.
6. Add a small check (script or test) asserting a produced `.skill` unzips with
   `SKILL.md` at the root, and that placeholder skills produce no artifact —
   this is the failing assertion that Phase 2 turns green.

### Phase 3 — Shared bootstrap (fetch the CLI)

1. Implement the bootstrap so the skill installs/locates the CLI by **fetching
   the binary from the latest GitHub Release**, falling back to `go install
   github.com/katabase-ai/katalyst@latest`.
2. Detect OS/arch and pick the matching release asset name (must agree with the
   naming Phase 4 produces).
3. Make it idempotent: reuse an already-installed binary; only download when
   missing. No version pin (out of scope) — track latest.
4. Decide and implement how the bootstrap ships inside each `.skill` (single
   shared source copied in at package time, so there is one bootstrap to
   maintain).

### Phase 4 — Release workflow

1. Add a tag-triggered workflow (`on: push: tags: ['v*']`), separate from the
   `test` job in `ci.yml`.
2. Build cross-platform CLI binaries via a GOOS/GOARCH matrix (the current
   `go build -o bin/katalyst .` is host-only), naming each asset to match what
   the Phase 3 bootstrap fetches (e.g. `katalyst_<os>_<arch>`).
3. Run `make skills` in the job.
4. Upload the binaries **and** every `.skill` as assets on the Release for that
   tag, in one workflow run.
5. Leave per-PR CI unchanged.

### Phase 5 — Local dev symlink

1. Add a `make` target that symlinks each `skills/{name}/` into
   `.claude/skills/` so they auto-load in a working copy, modeled on
   `sync_skill_links_from_cursor` in `scripts/agent-link-utils.sh`.
2. Add `.claude/skills/` to `.gitignore` (the specific path, not all of
   `.claude/`) so the symlinks stay uncommitted.

### Phase 6 — Author the remaining skills

1. Fill `katalyst-overview/`, `katalyst-catalog/`, `katalyst-identify-collections/`, and `katalyst-define-schemas/`
   `SKILL.md`s with real content, self-contained (no references to the
   `docs/content/how-to/` guides). (The deploy cluster was authored in Phase 1;
   `katalyst-migrate-schema`/`katalyst-migrate-storage` stay placeholders.)
2. Write `katalyst-overview` as the orientation/router: katalyst's model and vocabulary,
   plus pointers to each task skill. Keep it task-free (distinct from `katalyst-catalog`).
3. Wire the **cross-references** between `katalyst-identify-collections` (points forward
   to `katalyst-define-schemas`) and `katalyst-define-schemas` (points back as prerequisite).
4. Keep each skill's `references/` content scoped to what the agent needs at
   runtime.

### Phase 7 — Verify and graduate

1. Cut a test tag (or dry-run the release job) and confirm the assets appear.
2. Install a downloaded `.skill` via the client's "Save skill" flow with no
   repo clone; confirm the bootstrap fetches the CLI and the skill drives it.
3. Run the verification checklist below.
4. Per `docs/content/contributing/how-we-plan.md`:
   on **done**, graduate durable content — fold the locked rationale (one
   committed source, fetch-not-embed, per-stage family, Channel 1 first) into
   the relevant `docs/deep-dives/` page, add any new vocabulary to the glossary,
   point `README.md` at the skills download, then retire this spec + plan.

## Verification checklist

Mirrors the spec's test checklist; these assertions prove the change.

- [ ] `make skills` produces one `{name}.skill` per **shippable** directory
      under `skills/`, each unzipping with `SKILL.md` at the archive root, and
      emits no artifact for `status: placeholder` skills.
- [ ] `make skill SKILL=<name>` packages a single skill.
- [ ] `make clean` removes the `.skill` artifacts.
- [ ] The local-dev target symlinks each `skills/{name}/` into `.claude/skills/`;
      the symlinks are git-ignored.
- [ ] On a tag, the release workflow uploads the cross-platform binaries and
      every `.skill` as assets on that Release.
- [ ] A `.skill` downloaded from a Release installs via "Save skill" with no
      repo clone, and its bootstrap fetches the matching CLI binary from the
      latest Release (with `go install` fallback).
- [ ] `make all` (vet + test + build) still green; per-PR CI unchanged.

## Deviations

_None yet._
