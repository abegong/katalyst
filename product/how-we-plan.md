# How we plan

How we use specs and plans to design, build, and document changes.

## Two doc types

- **Spec** — *what* and *why*: the problem, the design, domain-model and
  architecture impact, open questions. A spec is a design doc in
  `product/specs/` (e.g. [`cli-spec.md`](specs/cli-spec.md)).
- **Plan** — *how*: the step-by-step implementation, broken into phases, that
  references its spec. A plan is a `{slug}-plan.md` in `product/specs/`. A
  spec can exist without a plan; a plan always references a spec.

For small, well-understood changes, skip the ceremony: a new entry in
[`decisions-to-make.md`](decisions-to-make.md) that graduates to
[`decisions.md`](decisions.md) is often the whole "spec." Don't write a full
spec doc when a decision record will do.

### What each contains

- **Spec:** Problem · Design (including domain-model impact) · Open
  questions · (optional) rejected alternatives and context.
- **Plan:** a reference to its spec · phases → steps · a **test checklist**
  (which becomes the failing tests — see step 4) · a running log of
  deviations.

## Lifecycle

| Status | Meaning |
|---|---|
| **planning** | Spec being written or refined; plan may not exist yet |
| **implementing** | Plan being executed; work in progress |
| **done** | Shipped and verified; ready to graduate content |
| **shelved** | Deprioritized; kept for reference |

Track status as a line at the top of the spec/plan, and overall sequencing in
[`roadmap.md`](roadmap.md). No separate summary file until the set of
in-flight specs is large enough to need one.

### Typical flow

1. **Write the spec.** Problem, design, open questions. Status: **planning**.
2. **Resolve open questions.** Fold resolutions into the design; record each
   locked choice in [`decisions.md`](decisions.md) with a D-number, and drop
   it from [`decisions-to-make.md`](decisions-to-make.md).
3. **Write the plan.** Phases and steps that reference the spec. Status stays
   **planning** until work begins.
4. **Implement, tests first.** Per [`AGENTS.md`](../AGENTS.md), new behavior
   arrives with a failing test. Scaffold the spec's test checklist as
   pending/failing tests, then make them pass. Status: **implementing**.
   Update the plan as you go — mark phases done, note deviations.
5. **Ship and verify.** `make all` green. Status: **done**.
6. **Graduate content** into permanent docs (below).

## Graduating content

When a spec reaches **done**, pull its durable content into permanent docs
and retire the spec. Targets — see [`how-we-document.md`](how-we-document.md)
for what belongs in each:

- **`AGENTS.md`** — new conventions, required patterns, gotchas.
- **`product/` architecture** ([`domain-model.md`](domain-model.md) +
  [`domain-model-mapping.md`](domain-model-mapping.md), glossary) — domain or
  subsystem changes.
- **[`decisions.md`](decisions.md)** — the locked rationale (D-numbers). This
  is also katalyst's historical record of *why*, so a spec's rejected
  alternatives and context live on here after the spec is gone.
- **`docs/` + `README.md`** — user-facing usage, once the surface is stable.
- **Go doc comments** — package/API-level behavior.

Evergreen architecture docs (domain model, general model, connectors) are
*not* specs and don't get retired — they're updated in place.

### Graduation checklist

When moving a spec to **done**:

- [ ] `AGENTS.md` updated with any new conventions/gotchas.
- [ ] `product/` architecture and domain model (incl.
      `domain-model-mapping.md`) reflect the change; new vocabulary added to
      the glossary.
- [ ] Locked decisions recorded in `decisions.md` (D-numbers); resolved
      questions removed from `decisions-to-make.md`.
- [ ] User-facing changes reflected in `docs/` and `README.md`.
- [ ] Spec/plan deleted (or marked **shelved**); rationale preserved in
      `decisions.md`.
