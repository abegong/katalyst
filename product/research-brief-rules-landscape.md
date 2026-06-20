# Research Brief: Rules, Guidelines & Conventions for Knowledge Bases

> **Purpose.** Survey other projects, tools, and communities that define and
> enforce rules on files, metadata, and content, and decide which katalyst
> should replicate. This brief tells the research team *what to look for*, *how
> to capture it*, and *how to prioritize* — so findings come back comparable
> and decision-ready, not as a pile of links.

---

## 1. Context: what katalyst is

katalyst defines and enforces **schemas and checks for structured metadata
(frontmatter) on markdown files** — think JSON Schema meets a content linter,
for a knowledge base. Checks are organized into three families:

- **`objects`** — structured frontmatter fields (required field, type, enum,
  number range, string length, schema validation).
- **`markdown`** — relationships between frontmatter and body (title matches
  H1, requires H1, single H1, no heading-level jumps, required section, code
  fence has language).
- **`filesystem`** — filename and path conventions (matches slug, extension in
  set, kebab-case, no spaces, parent dir in set, filename prefix).

The team should start with this current inventory in hand as the **dedup
baseline** — log a finding as "novel" only after confirming katalyst doesn't
already have it.

---

## 2. The goal and the lens

We are cataloging **rules** — *nameable, teachable assertions about content* —
that other tools or communities enforce, to decide which katalyst should adopt.

Classify every finding on three axes so it maps onto our model:

### Axis A — Family
`objects` · `markdown` · `filesystem` · **or** "new family we don't have"
(e.g. text-encoding, cross-file link integrity, content-type fitness).

### Axis B — Scope
- **Per-item** — decidable from one file.
- **Collection / project-scoped** — needs to see siblings, the link graph, or
  project state (uniqueness, orphans, required index files, duplicates).

This axis predicts architectural cost, so tag it explicitly.

### Axis C — Determinism level (the most important column)

How the verdict is produced. This predicts implementation cost *and* UX. **This
is a dimension, not a filter** — we are deliberately expanding past
deterministic rules into judgment-based ones.

- **L0 — Deterministic.** Pure function of the file(s). Same input → same
  verdict, no threshold. *(Everything katalyst ships today.)*
  *e.g.* filename is kebab-case; `year` is an integer.
- **L1 — Contextual / heuristic.** Decidable by a fixed algorithm, but needs
  context beyond one file — siblings, the link graph, git history, an external
  index — usually plus a tunable threshold. Reproducible *given the corpus and
  the knob*.
  *e.g.* orphan note (no inbound links); stale (untouched > N months);
  near-duplicate (embedding similarity > τ).
- **L2 — Judgment.** Requires semantic understanding; produced by an evaluator
  (LLM or human) and returns a grade/confidence, not a hard pass/fail.
  *e.g.* "the title accurately summarizes the body"; "this note is atomic — one
  concept"; "the tags are actually relevant"; "the summary field is consistent
  with the content."

**L1 and L2 are the expansion this project exists for.** L0 is table-stakes
coverage; L1/L2 are the differentiators. They map to different katalyst
machinery:

| Level | Likely machinery |
|---|---|
| L0 | Current `Check.Run(Context) → []Violation` contract |
| L1 | A collection-scoped check interface (sees siblings / graph / history) |
| L2 | A new evaluator-backed tier returning confidence + rationale |

---

## 3. Two coordinated sweeps

Run two searches sharing **one catalog**:

1. **L0 deterministic sweep** — the linter / schema landscape. Fast,
   exhaustive, table-stakes coverage.
2. **L1 / L2 judgment sweep** — PKM methodologies and editorial norms. Slower,
   where the genuinely novel rules live.

---

## 4. Where to look

### 4a. Deterministic landscape (L0, some L1)

| Category | Why relevant | Seeds |
|---|---|---|
| Markdown / prose linters | Closest to `markdown` family | markdownlint, remark-lint, textlint |
| Frontmatter / content schemas | Directly `objects` family | Astro content collections, Hugo/Zola taxonomies & archetypes, Jekyll/Decap CMS, Sanity/Contentful/TinaCMS, Frontmatter CMS (VS Code) |
| Filename / structure linters | `filesystem` family + collection tier | ls-lint, folderslint, eslint-plugin-project-structure, eslint-plugin-filenames, Steiger, Knip (orphan/unused detection) |
| Schema / data validation | Field-level rule vocabulary | JSON Schema, MongoDB validation, Zod/Yup, Pydantic, Cerberus, Frictionless (tabular), CUE |
| Config / policy linters | Mature rule UX + cross-file rules | yamllint, Spectral (OpenAPI/AsyncAPI), OPA/conftest, kubeconform, tflint |
| General-purpose linter design | Meta-patterns (see §6) | ESLint, RuboCop, Ruff, golangci-lint, Stylelint |

### 4b. Judgment landscape (L1 / L2)

The richest sources are communities that **codified their judgment into named,
teachable norms** — they already turned "good content" into discrete rules,
sometimes with remediation templates.

| Source | What it yields | Example "rules" |
|---|---|---|
| Zettelkasten / evergreen-notes (Andy Matuschak, Ahrens) | Note-quality norms | atomic (one concept per note); concept-oriented not source-oriented; densely linked; self-contained title-as-claim |
| Wikipedia policies & maintenance templates | Judgment codified at scale, *with* a violation vocabulary | verifiability / "citation needed"; notability; neutral POV; "stub"; orphan; "needs update" |
| Diátaxis & docs style guides (Google, Microsoft, Write the Docs) | Content-type fitness + editorial rules | a page is purely tutorial *or* reference, not mixed; explanation "explains why, not how"; reading-level / tone |
| Digital-garden / PKM tooling (Obsidian graph analytics, Dataview, Foam) | L1 graph/freshness signals | orphans, hubs, dead internal links, staleness, broken backlinks |
| Prose & semantic linters (Vale, LanguageTool, alex) | The L1↔L2 boundary | terminology consistency vs. a glossary; inclusive language; "matches house style" |
| LLM-eval / "AI linting" frameworks | How others operationalize L2 | rubric-graded checks, LLM-as-judge patterns, confidence thresholds |

> Note: katalyst's own docs are organized by Diátaxis, so "does this doc mix
> tutorial and reference?" is both a dogfooding example and a real L2 rule to
> spec.

---

## 5. Capture template (use verbatim, one entry per *rule*)

- **Rule name + source tool/community**
- **What it asserts** (one sentence)
- **Family** (Axis A) · **Scope** (Axis B) · **Determinism level** (Axis C)
- **What context decides it** — one file / siblings / link graph / git history
  / external corpus / semantic understanding
- **Evaluation mechanism** — rule engine · heuristic + threshold ·
  embedding/similarity · LLM-as-judge · human review
- **Verdict shape** — boolean · score + threshold · graded + confidence
- **Config shape** — how the user parameterizes it (copy a real snippet)
- **Remediation** — deterministic fix · suggestion · advisory-only
  *(Wikipedia maintenance templates are a good model: the "fix" is flagging,
  not editing.)*
- **False-positive failure mode** — *required for every L1/L2 entry*: how the
  rule could be wrong, and what that costs.
- **Do we already have it?** — map to an existing katalyst kind, or "novel"
- **Verdict** — replicate / generalizes-an-existing-check / out-of-scope /
  needs-new-architecture

---

## 6. Look for *mechanisms*, not just rules

The highest-leverage findings are often rule-*system* design patterns — one
mechanism unlocks many rules. Per tool, note how it handles:

- **Rule identity** — stable IDs/codes (Ruff `E501`, markdownlint `MD013`)
- **Severity levels** — error / warn / off, per-rule overrides
- **Inline disabling** — in-file suppression (`<!-- disable -->`)
- **Config inheritance / presets** — shareable rule packs, "recommended" sets
- **Rule composition** — OR-ing rules (ls-lint's `|`), `any_of`
- **Scoping** — glob/directory-scoped rule application
- **Custom / plugin rules** — user-defined-check escape hatch
- **Cross-file rules** — *how* they implement the collection-scoped tier
- **Confidence & uncertainty** *(L2-specific)* — how judgment tools surface
  confidence, set thresholds, and handle false positives

---

## 7. How to prioritize

Determinism is a **cost/risk axis**, not a filter. Plot value against
feasibility:

- **L0, common, per-item** → quick wins, table-stakes coverage.
- **High-value, L2** → the differentiators: judgment rules people genuinely
  want (atomicity, title-summarizes-body, stale/contradictory content) that no
  deterministic linter can offer.
- **Rare + collection-scoped or L2** → roadmap items behind an architecture
  decision.

Score each novel rule on: **value to a KB maintainer × frequency across sources
× feasibility at its determinism level** — where feasibility includes tolerance
for false positives. An over-eager L2 rule is worse than no rule.

---

## 8. Guardrails (anti-goals)

- **Every rule must be nameable and teachable.** A maintainer should read the
  rule name and know what it asks, even if a model adjudicates it. In:
  "the note covers exactly one concept." Out: "make the note better."
- **Markdown / knowledge-base shaped.** Generic code-linting rules (ESLint JS,
  RuboCop Ruby) are for *mechanism* inspiration only — don't catalog individual
  code rules.
- **Dedup against the current inventory** (§1) before logging anything novel.
- **L1/L2 entries must record a false-positive failure mode** — it determines
  whether a judgment rule is shippable.

---

## 9. Deliverables

1. **Comparison matrix** — rules × the §5 columns (one row per rule).
2. **Synthesis memo** — the top ~10 replicate-now rules, the mechanisms (§6)
   worth adopting, and anything that forces an architecture decision (new
   family, collection-scoped interface, or L2 evaluator tier).

The memo feeds directly into katalyst specs (e.g. the filesystem-checks spec).

---

## 10. Open decisions to lock before kickoff

- **Scope of families** — limit to the three current families, or actively hunt
  for *new* families (text-encoding, cross-file link integrity, content-type
  fitness)?
- **Depth of L2 findings** — should the team propose the *evaluation mechanism*
  per L2 rule (sketch how katalyst would adjudicate it), or just catalog the
  rule and leave the "how" to spec time?
