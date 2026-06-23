# Collection variants — plan

> Spec: [Collection variants](./collection-variants-spec.md)
>
> **Status: implementing (all phases landed; graduation pending merge).**
> Adds an optional `variants:` layer to a collection: an
> ordered list of metadata-discriminated check groups, plus a
> `useExhaustiveVariants` flag. Discrimination reuses the `item list --filter`
> predicate grammar (`internal/query`), so routing lives entirely in the check
> engine and `internal/storage` is untouched. Phase 1 (export predicate
> evaluation), Phase 2 (config model), Phase 3 (engine routing), and Phase 4
> (docs + dogfood `weight`-on-content variant + this status) are all
> implemented; `make all` is green and dogfood `check` passes. Retiring this
> spec/plan (final graduation) waits on merge.

## Current State

- **Predicate evaluation is unexported.** `internal/query/filter.go`
  (`ParseFilter` → `Predicate`) parses `--filter` expressions, but the evaluator
  `func (p Predicate) match(meta map[string]any, typeMismatch string)` is
  unexported; only `query.Apply` (`query.go:53`) calls it, for `item list`. A
  caller outside `query` cannot evaluate a parsed predicate against an item's
  metadata. `internal/query` imports only stdlib + yaml — it does **not** import
  `internal/config`, so `config` may import `query` without a cycle.
- **A collection carries one check profile.** `internal/config/config.go:180`
  — `Collection{Name, Path, Dir, Pattern, Schema, Checks, Query, Storage}`.
  `buildCollection` folds the `schema:` shorthand into a leading
  `CheckInstance{Type: CheckObject, Schema}` and appends `checks:` normalized
  through `normalizeCheck`; `rawCollection` (`config.go:262`) is the YAML shape.
  There is no per-item variation.
- **The engine compiles one list per collection.** `cmd/engine.go:73`
  `checksFor(c, meta)` resolves object schemas by precedence
  (`--schema` > inline `schema:` key > the collection's `CheckObject` entries)
  and appends the non-object checks from `c.Checks`. It already receives the
  item's parsed `meta` — the input a metadata discriminator needs — but ignores
  it except for the inline-schema key. `checkItem` (`check.go`) and `itemStatus`
  (`check.go`, for `item list`) both build their check list via `checksFor` and
  run it through `checks.RunAll`.
- **`checks.Check` is a one-method interface.** `internal/checks/checks.go:29`
  — `Check interface { Run(Context) []Violation }`, exported, so `cmd` can
  supply a synthetic check for the unrouted-under-exhaustive case without
  touching `internal/checks`.
- **Invariant #4** (`docs/content/deep-dives/domain-model.md:324`) forbids
  glob-ordering "first match wins"; this work relaxes that clause *within* a
  collection (spec).

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Export predicate eval | `internal/query`: add exported `Predicate.Matches` (wraps `match`). Leaf, enabling; no behavior change. |
| 2 | Config model | `internal/config`: parse/validate `variants:` (`when`/`where` → `[]query.Predicate`, variant `schema`/`checks` normalized like a collection's) and `useExhaustiveVariants:`. Inert until Phase 3. |
| 3 | Engine routing | `cmd/engine.go`: route by first matching variant, compose base+variant additively, enforce exhaustiveness via a synthetic check. The behavior change. |
| 4 | Docs, dogfood, graduation | Reword invariant #4; configuration/how-to/glossary/storage docs; update `.katalyst/storage/local.yaml`; package/AGENTS docs; retire spec+plan. |

Each phase is **tests-first internally**: write the failing test sub-step, then
the code that makes it pass (a single up-front scaffold won't compile in Go).

## Phases

### Phase 1 — Export predicate evaluation (`internal/query`)

**Goal:** A parsed `Predicate` can be evaluated against any metadata map from
outside `query`, so config can validate and the engine can route.

1. **File:** `internal/query/filter_test.go` *(edit, failing first)* — add
   `TestPredicate_Matches`: a `ParseFilter("kind=section")` predicate `Matches`
   a `map[string]any{"kind": "section"}` and not `{"kind": "page"}`; an absence
   predicate (`!draft`) matches a map without `draft`; the `typeMismatch`
   argument behaves like `Apply`'s (skip vs. error). These mirror the existing
   `match` cases through the new exported door.
2. **File:** `internal/query/filter.go` *(edit)* — add exported
   `func (p Predicate) Matches(meta map[string]any, typeMismatch string)
   (bool, error)` that simply calls the existing unexported `match`. Keep
   `match` for `Apply`. Document that `Matches` is the per-item evaluator reused
   by variant discriminators, not only `item list`.
3. **Gate:** `go test ./internal/query/...` green.

### Phase 2 — Config: parse and validate variants

**Goal:** A collection definition accepts `variants:` and
`useExhaustiveVariants:`, validated at load; the model is inert (no engine reads
it yet) so this phase changes no `check` behavior.

1. **File:** `internal/config/config_test.go` *(edit, failing first)* — assert:
   a collection with two `variants` (each `when.where: [...]` + `schema`/`checks`)
   loads into `Collection.Variants` with predicates parsed and each variant's
   `schema:` folded into a leading `CheckObject` (mirroring the collection-level
   fold); the `when:` **string and list shorthands** desugar to `where:`; an
   invalid predicate (`ParseFilter` error) is a load error citing the collection
   and variant index; a variant `schema:` naming an unknown schema is a load
   error; a variant `checks:` entry is validated through `normalizeCheck`;
   `useExhaustiveVariants: true` is recorded (default `false`). Naming per
   `AGENTS.md`: `TestLoad_variantPredicateParsed`,
   `TestLoad_rejectsUnknownVariantSchema`, `TestLoad_whenShorthandDesugars`, …
2. **File:** `internal/config/config.go` *(edit)* — add:
   - `CollectionVariant{Where []query.Predicate; Checks []CheckInstance}` (the
     variant `schema:` is folded into a leading `CheckObject` in `Checks`, so the
     engine compiles base and variant through one path).
   - `Variants []CollectionVariant` and `UseExhaustiveVariants bool` on
     `Collection`.
   - `rawVariant{When rawWhen; Schema string; Checks []rawCheck}` and a `rawWhen`
     that unmarshals **either** a string, a `[]string`, or `{where: [...]}` into
     a predicate-string list (custom `UnmarshalYAML`); `variants []rawVariant`
     and `useExhaustiveVariants bool` on `rawCollection`.
   - In `buildCollection`: after the existing base-checks build, loop `rc.Variants`
     → `ParseFilter` each `where` string (error → `collection %q: variants[%d]:
     %w`), fold variant `schema` into a leading `CheckObject` and `normalizeCheck`
     the rest exactly as the collection-level path does, and append a
     `CollectionVariant`. Carry `UseExhaustiveVariants` through.
   - `config` now imports `internal/query`. (`query` does not import `config`;
     no cycle.)
3. **Gate:** `make all` green (existing suites unaffected — variants are
   additive and optional).

### Phase 3 — Engine: route, compose, enforce

**Goal:** An item runs base checks plus the first matching variant's checks; an
unrouted item under `useExhaustiveVariants` fails; everything else is unchanged.

1. **File:** `cmd/check_test.go` *(edit, failing first)* — fixtures with a
   variant collection, asserting:
   - an item matching variant A runs base + A's checks; an item matching B runs
     base + B's checks (e.g. `weight` required only on content pages, exempt on
     section indexes);
   - **first match wins**: with overlapping `where`, only the first variant's
     checks apply;
   - **additive object schemas**: an item is validated against the base schema
     *and* the matched variant's schema (both must pass);
   - an unrouted item runs base only when `useExhaustiveVariants` is false, and
     fails with `matches no variant` when true;
   - object-schema precedence holds: `--schema`/inline `schema:` still override
     the configured object tier (base **and** variant), while variant
     markdown/filesystem checks still run.
   Cover `item list` status parity (a variant violation counts in `itemStatus`).
2. **File:** `cmd/engine.go` *(edit)* — in `checksFor(c, meta)`:
   - select the matched variant: first `v` in `c.Variants` where **every**
     `v.Where[i].Matches(meta, c.Query.FilterTypeMismatch)` is true (a `Matches`
     error under `error` mode propagates out as today).
   - build `effective := append(slices.Clone(c.Checks), matched.Checks...)` and
     run the **existing** object-precedence switch and non-object loop over
     `effective` instead of `c.Checks` — so a variant's object check joins the
     configured tier (skipped under `--schema`/inline, exactly like the base
     object check) and its markdown/filesystem checks always run.
   - if no variant matched and `c.UseExhaustiveVariants`, append a synthetic
     `unroutedCheck{}` (a small `cmd`-local type implementing `checks.Check`
     whose `Run` returns one `Violation{Message: "matches no variant"}`), so
     both `checkItem` and `itemStatus` report it uniformly through `RunAll`.
   - the "no checks configured" guard stays, now over `effective`.
3. **Gate:** `make all` green; dogfood `check` (below) still passes.

### Phase 4 — Docs, dogfood config, graduation

**Goal:** Document the feature, turn it on for katalyst's own docs where
frontmatter allows, and retire the spec/plan.

1. **File:** `docs/content/deep-dives/domain-model.md` *(edit)* — reword
   invariant #4: keep "an item belongs to one collection"; scope
   "first match wins" to *within a collection, among its variants*; fold in the
   rationale (spec).
2. **File:** `docs/content/reference/configuration.md` *(edit)* — document
   `variants:` and `useExhaustiveVariants:`: the `when`/`where` discriminator as
   the `item list --filter` grammar, the string/list shorthand, first-match-wins,
   additive base+variant composition, the unrouted-item behavior, and that
   membership/`Unmatched` stay governed by `pattern`. Add the variant-schema note
   to the object-schema precedence section.
3. **File:** `docs/content/how-to/configure-rules.md` *(edit)* — add the
   section-index-vs-content-page worked example.
4. **Files:** `docs/content/reference/glossary.md` *(edit)* — add **Variant**
   and **Discriminator**. `docs/content/deep-dives/storage.md` *(edit)* — one
   line: variants discriminate by metadata (portable across StorageTypes);
   path-based discrimination is deferred. `docs/content/reference/commands.md` /
   the `item list --filter` reference *(edit)* — cross-link the shared grammar.
5. **File:** `.katalyst/storage/local.yaml` *(edit)* — add `section_index` and
   `content_page` variants split on `bookCollapseSection` (the `_index.md` case
   the `# issue #41` comments call out); require `weight` on content pages.
   Leave a comment that the generated `reference/check-types/**` pages still wait
   on deferred path discrimination (or a `generated:` marker), so `requires_h1`
   is **not** added yet. Confirm `make all` keeps dogfood `check` green.
6. **Files:** `internal/config/README.md`, `internal/query` package doc,
   `cmd/engine.go`/`internal/config/config.go` doc comments, `AGENTS.md`
   *(edit)* — variant routing lives in the engine keyed on item metadata;
   `query.Predicate.Matches` is reused beyond `item list`; the validation rules.
7. **File:** `product/specs/dogfood-docs-spec.md` *(edit)* — note per-page-type
   enforcement is **partially** unblocked (frontmatter-distinguishable types
   now; path-only types pending).
8. **Graduation:** set the spec Status to **done**, run the `how-we-plan.md`
   graduation checklist, delete `collection-variants-spec.md` + this plan.
9. **Gate:** `make all` and `make docs-gen-check` clean.

## Key Files

| File | Role |
|---|---|
| `internal/query/filter.go` | Add exported `Predicate.Matches` (edit) |
| `internal/config/config.go` | `CollectionVariant`, `Variants`/`UseExhaustiveVariants`, `rawVariant`/`rawWhen`, build + validate; import `query` (edit) |
| `cmd/engine.go` | Variant selection, additive compose, synthetic `unroutedCheck` (edit) |
| `cmd/check.go` | Unchanged shape; `checkItem`/`itemStatus` report variant violations via `RunAll` |
| `internal/{query,config}/*_test.go`, `cmd/check_test.go` | Tests-first per phase |
| `.katalyst/storage/local.yaml` | Dogfood: section-index/content-page variants (edit) |
| `docs/.../domain-model.md`, `configuration.md`, `configure-rules.md`, `glossary.md`, `storage.md` | Reword invariant #4; document variants (edit) |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Discriminator | Metadata predicate (the `--filter` grammar), not a glob | Metadata is the one property every item yields on every StorageType; a glob is filesystem-only and dead on a tabular backend (spec) |
| Where routing lives | The check engine (`checksFor`), keyed on `meta` | Variants pick *which checks run*, not *what an item is*; the engine already has the parsed metadata, so `internal/storage` stays untouched |
| Variant schema handling | Fold `schema:` into a leading `CheckObject` in the variant's `Checks` | Mirrors `buildCollection`; lets the engine compile base + variant through one object-precedence path, so `--schema`/inline override both tiers consistently |
| Compose semantics | Base always + first matching variant, additively | Exemption works without glob negation: a check lives in a variant exactly when some type must skip it (spec) |
| Unrouted under exhaustive | A synthetic `cmd`-local `checks.Check` | Uniform reporting through `RunAll` for both `check` and `item list`; no signature churn, no routing logic leaking into `internal/checks` |
| Predicate eval surface | Export `Predicate.Matches`, keep `match` | `config` (validate) and `cmd` (route) need it; `query` stays the single owner of predicate semantics |
| Config↔query direction | `config` imports `query` | `query` imports no `config`; validating predicates at load fails fast on a bad `where` |

## Out of Scope

- **Path-based discrimination.** No `path:` condition in v1; the `when:` block is
  shaped to accept one later (storage-type-scoped, evaluated behind the seam).
  Consequence: page types distinguished only by location — the generated
  `reference/check-types/**` pages — aren't discriminable yet, so their
  `requires_h1` exemption waits on the deferred condition or a `generated:`
  frontmatter marker (spec, "Deferred").
- **OR / NOT across conditions.** `where:` entries AND together; per-field
  negation (`!=`, `!field`) covers the common case. A nested any/all/not matcher
  waits for a real disjunctive case.
- **A richer strict mode.** `useExhaustiveVariants` is a single boolean; no
  per-variant "required" or coverage reporting.
- **Non-filesystem StorageTypes.** Unchanged by this work, which never touches
  `internal/storage`.

## Test checklist

Phase 1 (query):
- [ ] `Predicate.Matches` evaluates eq / absence / type-mismatch like `match`

Phase 2 (config):
- [ ] `variants` parse: `where` → `[]query.Predicate`; variant `schema` folded
      into a leading `CheckObject`
- [ ] `when` string and list shorthands desugar to `where`
- [ ] invalid predicate → load error (collection + variant index)
- [ ] unknown variant schema → load error; variant `checks` validated via
      `normalizeCheck`
- [ ] `useExhaustiveVariants` recorded; default `false`

Phase 3 (engine):
- [ ] item runs base + first-matching variant's checks
- [ ] first match wins on overlapping `where`
- [ ] base + variant object schemas both enforced (additive)
- [ ] unrouted: base only when lenient; `matches no variant` when exhaustive
- [ ] `--schema`/inline override base **and** variant object tier; variant
      markdown/filesystem checks still run
- [ ] `item list` status counts a variant violation

Phase 4 (docs + dogfood):
- [ ] invariant #4 reworded; `variants`/`useExhaustiveVariants` documented
- [ ] `.katalyst/storage/local.yaml` variants land; dogfood `check` green
- [ ] `make docs-gen-check` clean
