+++
title = "How we plan"
weight = 30
+++

# How we plan

How we use specs and plans to design, build, and document changes.

## Two doc types

- **Spec:** *what* and *why*: the problem, the design, domain-model and
  architecture impact, open questions. A spec is a design doc in
  `product/specs/{slug}-spec.md`.
- **Plan:** *how*: the step-by-step implementation, broken into phases, that
  references its spec (`{slug}-plan.md`). A spec can exist without a plan; a
  plan always references a spec.

For small, well-understood changes, skip the ceremony: a GitHub issue
capturing the decision is often enough. Don't write a full spec when an issue
will do.

## Where rationale and open questions live

There is **no `decisions.md`** and no ADR log. When a change locks in a
choice, its rationale graduates next to the thing it explains: for a package
or subsystem, into package docs (a `doc.go` when long, or co-located package
`README.md`), and for a genuinely cross-cutting concept, into the relevant
[`deep-dives/`]({{< relref "../deep-dives/_index.md" >}}) page. Either way it
is written into prose beside what it explains. When the choice supersedes an
earlier approach, that same home notes the old approach and why it changed.

Open questions get no standing file. While a change is in flight they live in
its `product/specs/` spec; otherwise they are GitHub issues.

## Lifecycle

| Status | Meaning |
|---|---|
| **planning** | Spec being written or refined; plan may not exist yet |
| **implementing** | Plan being executed; work in progress |
| **done** | Shipped and verified; ready to graduate content |
| **shelved** | Deprioritized; kept for reference |

Track status as a line at the top of the spec/plan.

### Typical flow

1. **Write the spec.** Problem, design, open questions. Status: **planning**.
2. **Resolve open questions.** Fold each resolution into the design. The
   locked rationale is destined for the thing it explains, package docs for
   subsystem choices, or a cross-cutting `deep-dives/` page: at graduation,
   not a separate log.
3. **Write the plan.** Phases and steps that reference the spec.
4. **Implement, tests first.** Per `AGENTS.md`, new behavior arrives with a
   failing test. Scaffold the spec's test checklist as failing tests, then
   make them pass. Status: **implementing**. Update the plan as you go.
5. **Ship and verify.** `make all` green. Status: **done**.
6. **Graduate content** into permanent docs (below), then delete the spec.

## Graduating content

When a spec reaches **done**, pull its durable content into permanent docs
and retire the spec. Targets, see [How we
document]({{< relref "how-we-document.md" >}}) for what belongs where:

- **`AGENTS.md`:** new code conventions, required patterns, gotchas.
- **Package docs:** package/API behavior and subsystem architecture
  rationale (including rejected alternatives and "why not X"), in a `doc.go`
  when long.
- **`docs/deep-dives/`:** cross-cutting conceptual changes and rationale
  that no single package owns.
- **`docs/reference/`:** the precise surface; for checks, regenerate
  `reference/check-types/` with `make docs-gen`.
- **`docs/how-to/` and `docs/getting-started.md`:** user-facing usage.
- **`docs/reference/glossary.md`:** new vocabulary.
- **`README.md`:** pointer/overview updates.

Evergreen deep-dive docs (the storage layer, progressive operations) and the
per-package `README.md` files under `internal/` are *not* specs and don't get
retired: they're updated in place.

### Graduation checklist

When moving a spec to **done**:

- [ ] `AGENTS.md` updated with any new conventions/gotchas.
- [ ] Package architecture/rationale captured in package docs (`doc.go` when
      long); `docs/deep-dives/` touched for cross-cutting model changes.
- [ ] `docs/reference/` updated; `make docs-gen` run if a check changed.
- [ ] New vocabulary added to the glossary.
- [ ] User-facing changes reflected in `docs/how-to`, `getting-started.md`,
      and the `README.md`.
- [ ] Open questions closed as issues or removed.
- [ ] Spec/plan deleted (or marked **shelved**).
