# Docs reorganization — plan

> Spec: [Docs reorganization](./docs-reorganization-spec.md)

## Current State

- **Hugo site.** `hugo.yaml` sets `contentDir: docs`, theme `hugo-book`, baseURL
  `https://katabase-ai.github.io/katalyst/` (GitHub Pages — the site is public).
  Built by `make docs-build`, served by `make docs-serve`. Content files use TOML
  `+++` frontmatter; the sidebar is theme-driven (`BookSection: "*"`), so each
  section needs an `_index.md`.
- **Current `docs/`.** Flat: `_index.md`, `getting-started.md`, `commands.md`,
  `configuration.md`, `manifesto.md`, `technical-spec.md`, `rules/` (per-check
  pages + family `_index.md`), and an empty `examples/`.
- **`product/` design docs.** `general-model.md`, `progressive-operations.md`,
  `domain-model.md`, `domain-model-mapping.md`, `connectors.md`, `decisions.md`,
  `decisions-to-make.md`, `roadmap.md`, `how-we-document.md`, `how-we-plan.md`,
  `specs/cli-spec.md`, and a stale top-level `cli-spec.md` (duplicate).
- **Checks engine.** `internal/checks/{checks.go,object.go,markdown.go,filesystem.go}`
  implement the `Check` interface. `internal/config/config.go` defines the
  `CheckKind` constants and dispatches them in a `switch` (~L320–423). There is no
  machine-readable per-kind descriptor today — the rule generator must add one.
- **Skills.** `.cursor/skills/{write-spec,write-docs,add-katalyst-rule}/SKILL.md`
  encode the current taxonomy and reference `product/decisions.md` and
  `product/how-we-*.md`.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Skeleton + guidelines | `docs/` section tree, portal, `documentation-guide.md` + reference/explanation templates |
| 2 | Rule reference generator | checks descriptor registry + generator + CI gate; replace `docs/rules/` |
| 3 | Explanation migration | move conceptual docs to `explanation/`, fold in the checks engine, distribute decision rationale |
| 4 | User docs | `reference/configuration`, `tutorials/getting-started`, how-to, glossary; derive remaining templates |
| 5 | Process + skills + cleanup | rewrite & move `how-we-*`, repoint 3 skills, publish roadmap, delete retired files |

Phases 1–2 are structure/code and independent of content; 3–4 move content; 5
finalizes governance and deletes retired files. CLI reference and the example
harness are out of scope (separate PRs).

## Phases

### Phase 1 — Skeleton + guidelines

**Goal:** Stand up the Diátaxis section tree and the contributor guide so later
phases drop files into stable homes.

1. **File:** `docs/tutorials/_index.md`, `docs/how-to/_index.md`,
   `docs/reference/_index.md`, `docs/explanation/_index.md`,
   `docs/contributing/_index.md` *(new)* — each with `+++` `title` + `weight`
   (10/20/30/40/50) so hugo-book renders an ordered sidebar. One-sentence lede per
   section.
2. **File:** `docs/_index.md` — rework the portal into two tracks: "Use Katalyst"
   (tutorials → how-to → reference) and "Contribute" (explanation → contributing).
   Replace the current flat link list.
3. **File:** `docs/contributing/documentation-guide.md` *(new)* — the "where does
   this go?" decision tree (user vs contributor; which quadrant), the
   `product/specs/` spec/plan pointer, and a link to the glossary (Phase 4).
4. **File:** `docs/contributing/templates/reference.md`,
   `docs/contributing/templates/explanation.md` *(new)* — the two quadrant
   templates, each with the Diátaxis "this page IS X, is NOT Y" guardrail. Set
   `draft = true` so the public `docs-build` excludes them (Q5: only these two
   now).
5. **Gate:** `make docs-build` succeeds; the five sections render in the sidebar.

### Phase 2 — Rule reference generator

**Goal:** Generate `docs/reference/rules/` from the checks engine so a new check
can't ship undocumented.

1. **File:** `internal/checks/registry.go` *(new)* — a `Descriptor` type (kind id,
   one-line summary, required/optional config fields) and a `Descriptors()` slice
   covering all 15 kinds, keyed to the `CheckKind` constants in
   `internal/config/config.go`.
2. **File:** `internal/checks/registry_test.go` *(new, write first/failing)* —
   assert every `CheckKind` dispatched in `config.go`'s switch has a `Descriptor`
   and vice-versa. This is the no-orphan guarantee.
3. **File:** `cmd/gendocs/main.go` *(new)* — render one
   `docs/reference/rules/<family>/<kind>.md` per descriptor plus the family and
   section `_index.md`, mirroring today's `docs/rules/` grouping.
4. **File:** `Makefile` — add a `docs-gen` target (`go run ./cmd/gendocs`) and a CI
   no-drift check (`docs-gen` then `git diff --exit-code docs/reference/rules`).
5. **File:** `docs/rules/` *(delete)* — replaced by generated
   `docs/reference/rules/`; fold the family `_index.md` intros into the template.
6. **Gate:** `registry_test.go` green; `make docs-gen` clean; `make docs-build`
   green.

### Phase 3 — Explanation migration + reconciliation

**Goal:** Move conceptual docs into `explanation/`, describing shipped reality,
with decision rationale distributed in.

1. **File:** move `product/general-model.md`, `product/progressive-operations.md`,
   `product/domain-model.md`, `product/domain-model-mapping.md`,
   `product/connectors.md` → `docs/explanation/` — add `+++` frontmatter + weights,
   convert internal links to `{{< relref >}}`, label `connectors.md` "future / not
   shipped."
2. **File:** move `docs/manifesto.md`, `docs/technical-spec.md` →
   `docs/explanation/`; trim `technical-spec.md` overlap with `general-model.md`
   and the roadmap.
3. **File:** `docs/explanation/domain-model.md` — fold in the shipped 15-check
   engine (markdown + filesystem checks, `checks:` lists) the current lifecycle
   omits. Keep the surface at shipped `validate`/`rules:`; mark `check`/`collections:`
   planned with a pointer to `specs/cli-spec.md`.
4. **File:** `docs/explanation/configuration.md` *(new)* — absorb D1/D2 rationale
   (why YAML, nearest-ancestor discovery, resolution precedence) at the shipped
   `rules:` surface.
5. **File:** `docs/explanation/formatting.md` *(new)* — absorb D3 (no auto-injected
   values) and D4 (opinionated `fmt`) rationale.
6. **Gate:** `make docs-build` green; grep finds no broken `relref`.

### Phase 4 — User docs

**Goal:** Complete the user-facing pages at the shipped surface and derive the
remaining templates from them.

1. **File:** move `docs/configuration.md` → `docs/reference/configuration.md` —
   keep the shipped `rules:`/`checks:` surface; cross-link to
   `explanation/configuration.md`; remove step-by-step content (→ how-to).
2. **File:** `docs/how-to/configure-rules.md`, `docs/how-to/add-a-schema.md`,
   `docs/how-to/validate-in-ci.md` *(new)* — task-oriented, hand-written examples
   at the shipped surface (Q4 interim).
3. **File:** move `docs/getting-started.md` → `docs/tutorials/getting-started.md`;
   expand into a guided lesson (`make build`, `katalyst init`, `katalyst
   validate`).
4. **File:** `docs/reference/glossary.md` *(new)* — from the `domain-model.md`
   Vocabulary table.
5. **File:** `docs/contributing/templates/tutorial.md`,
   `docs/contributing/templates/how-to.md` *(new)* — derive each from the real
   pages just written (Q5).
6. **Gate:** `make docs-build` green; links resolve.

### Phase 5 — Process docs, skills, cleanup

**Goal:** Rewrite the governance docs for the new taxonomy, repoint the skills,
and delete retired files.

1. **File:** rewrite `product/how-we-document.md` →
   `docs/contributing/how-we-document.md` — new taxonomy: `docs/` for users +
   contributors; `AGENTS.md` co-located code conventions; `product/` ephemeral
   specs only; no `decisions.md`; rationale in `explanation/`.
2. **File:** rewrite `product/how-we-plan.md` → `docs/contributing/how-we-plan.md`
   — drop the "record D-number in `decisions.md`" step; resolutions fold into
   `explanation/`; update graduation targets (AGENTS.md, explanation, docs
   reference, README).
3. **File:** `.cursor/skills/write-spec/SKILL.md`, `.cursor/skills/write-docs/SKILL.md`,
   `.cursor/skills/add-katalyst-rule/SKILL.md` — drop `product/decisions.md`
   references; point process/taxonomy links at `docs/contributing/how-we-*`; point
   the rule-reference link at `docs/reference/rules/`; in `add-katalyst-rule`, add
   the step to register a `Descriptor` in `internal/checks/registry.go`.
4. **File:** move `product/roadmap.md` → `docs/contributing/roadmap.md` — describe
   the shipped surface; mark future items planned.
5. **File:** delete `product/decisions.md`, `product/decisions-to-make.md`
   (rationale now in `explanation/`; open questions filed as GitHub issues) and the
   stale top-level `product/cli-spec.md` (canonical stays at
   `product/specs/cli-spec.md`).
6. **File:** `README.md`, root `AGENTS.md` — fix pointers to moved files.
7. **Gate:** `make all` green; `make docs-build` green; repo-wide grep finds no
   references to deleted paths.

## Key Files

| File | Role |
|---|---|
| `hugo.yaml` | Hugo config; `contentDir: docs`, hugo-book (unchanged) |
| `docs/_index.md` | Portal (reworked) |
| `docs/{tutorials,how-to,reference,explanation,contributing}/_index.md` | Section landing pages (new) |
| `docs/contributing/documentation-guide.md` | Guidelines + decision tree (new) |
| `docs/contributing/templates/*.md` | Quadrant templates (new; `draft`) |
| `internal/checks/registry.go` | Per-kind descriptors; generator source (new) |
| `cmd/gendocs/main.go` | Rule-reference generator (new) |
| `Makefile` | `docs-gen` target + CI no-drift check (edited) |
| `docs/reference/rules/` | Generated rule pages (replaces `docs/rules/`) |
| `docs/explanation/*` | Migrated conceptual docs + distributed rationale |
| `docs/contributing/how-we-*.md` | Rewritten process docs |
| `docs/contributing/roadmap.md` | Published roadmap (moved) |
| `.cursor/skills/{write-spec,write-docs,add-katalyst-rule}/SKILL.md` | Repointed skills |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| CLI reference | Out of scope; lands with the rebuild | Generating from Cobra now would enshrine `validate`/`fmt`; decouples the reorg from the rebuild |
| Documented surface | Describe shipped reality (`rules:`, `validate`) | Avoids documenting unshipped `check`/`collections:`; the rebuild reconciles |
| Rule-doc source | Generated from `internal/checks/registry.go` descriptors | Single source of truth; CI gate blocks undocumented checks |
| Decision rationale | Distributed into `explanation/`, no `decisions.md` | Rationale lives with its topic |
| Templates | Ship reference + explanation now; derive the rest | Templates earned by a real page, not guessed |
| Template files | `draft = true` under `docs/contributing/templates/` | In-repo for contributors, excluded from the public build |
| Frontmatter | Keep `+++` TOML | Match existing docs and hugo-book |

No `decisions.md` exists to mirror these into (it is deleted in Phase 5); they
graduate into `docs/contributing/how-we-document.md` and the relevant
`explanation/` pages when the reorg ships.

## Out of Scope

- **CLI reference** (`docs/reference/cli/`) and the Cobra doc generator — rebuild
  branch.
- **Surface reconciliation** to `check`/`fix`/`collections:` — rebuild branch.
- **Executable example harness** (testscript/txtar under `cmd/testdata/`,
  `docs/examples/`) — separate PR after the rebuild (Q4).
- **Tutorial/how-to templates** beyond the first real page (Q5).
- **README rewrite** beyond pointer fixes.
