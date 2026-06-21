# CLI command grammar — noun/verb separation

> **Status: planning.** Establishes the rule that governs where a new
> top-level command goes. It refines, and does not replace, the surface
> defined in [`cli-spec.md`](cli-spec.md); that spec lists the commands, this
> one states the grammar they obey.

## Principle

**At the CLI level, noun-based commands are kept separate from verb-based
commands.** A top-level command is one of exactly two kinds:

- a **blessed verb** — `katalyst <verb> [selector ...]` — a cross-cutting
  operation that runs *over* content, or
- a **resource noun** — `katalyst <noun> <verb> <selector>` — a group whose
  sub-verbs act *on* one kind of resource.

The two grammars never blur into each other. A noun is never invoked bare as
though it were a verb; a cross-cutting verb is never buried under a noun.

## Problem

The surface today carries both grammars, which is correct — but it carries
them inconsistently, and nothing written down says which grammar a *new*
command should adopt. The concrete symptoms:

- **`rules` is a noun wearing a verb's clothes.** `katalyst rules` runs an
  action (it lists the registry) with no sub-verb, while the sibling
  resources `collection`, `item`, and `schema` all require one
  (`collection list`, `item get`, `schema show`). Same kind of thing,
  two different shapes.
- **No placement rule.** When the next command arrives, there is no stated
  test for "is this a blessed verb or a noun sub-verb?" — so the answer gets
  re-litigated each time, and the surface drifts.

This is a small, well-understood change to a convention, but it touches the
whole command tree and the choice has been made implicitly more than once, so
it earns a written rule.

## The two families

### Blessed verbs — operate *over* content

`katalyst check`, `katalyst fix`. The verb names a cross-cutting operation;
the **selector** picks targets at any depth, and several may be passed at
once:

```
katalyst check                       # whole project
katalyst check notes                 # one collection
katalyst check notes/dune schemas    # mixed depth, many targets
```

Defining traits (these are the membership test):

- The operation is **meaningful across more than one resource kind** — it is
  about the *content*, not about administering one resource type.
- It accepts a selector at **any depth**, and accepts **multiple** selectors
  (per [`cli-spec.md`](cli-spec.md) § Selector grammar).
- Adding a new resource kind must not require a new verb — the verb already
  spans them.

`init` is the lone **project verb**: a lifecycle operation on the project
itself, no noun and no selector. It belongs to this family by being a verb,
but it is its own shape (`katalyst init [--dir]`) and the selector rules do
not apply to it.

### Resource nouns — operate *on* one resource

`katalyst collection`, `katalyst item`, `katalyst schema`. The noun names a
resource type; the **sub-verb** is a CRUD-shaped operation at a **fixed
depth**:

```
katalyst collection list             # depth 0 (the set)
katalyst collection get  <c>         # depth 1
katalyst item       list <c>         # depth 1
katalyst item       get  <c>/<i>     # depth 2
```

Defining traits:

- The operation only makes sense **for that one resource kind**.
- The selector depth is **fixed and stated per sub-verb**; wrong depth is a
  usage error (exit 2).
- A bare noun (`katalyst item`) is **not an action** — it prints help/usage,
  never a default list.

## Placement rule (the one test)

When adding a command, ask: *does this operation span resource kinds, or
administer exactly one?*

| Answer | Family | Shape |
|---|---|---|
| Spans kinds; about the content | **Blessed verb** | `katalyst <verb> [selector ...]` |
| Administers one resource kind | **Noun sub-verb** | `katalyst <noun> <verb> <selector>` |
| Acts on the project itself | **Project verb** | `katalyst <verb> [flags]` |

What the rule forbids, concretely:

- **No top-level CRUD verbs.** Do not add bare `add` / `get` / `list` that
  guess their noun. CRUD lives under its noun (`item add`, not `add`).
- **No cross-cutting verb under a noun.** Do not split `check` into
  `item check` / `collection check`; a content-wide operation stays a blessed
  verb.
- **No bare noun as an action.** A noun with no sub-verb shows help — it does
  not silently run a default.

## Current surface, classified

| Command | Family | Conforms? |
|---|---|---|
| `init` | project verb | ✅ |
| `check`, `fix` | blessed verb | ✅ |
| `collection list` / `get` | noun | ✅ |
| `item list` / `get` / `add` / `update` / `delete` | noun | ✅ |
| `schema list` / `show` | noun | ✅ (see note) |
| `rules` (bare) | noun-shaped, verb-invoked | ❌ — the one offender |

Note on `schema`: it conforms structurally but its read verb is `show` where
`collection`/`item` use `get`. The sub-verb *vocabulary* is a separate
consistency question from noun/verb *separation*; it is out of scope here and
flagged as an open question only so the boundary is explicit.

## Bringing `rules` into the grammar

`rules` is a genuine resource noun — a catalog of check kinds read from the
engine registry (`internal/checks/`), reading no project. To obey the
principle it gains sub-verbs that mirror its existing behavior one-to-one:

| Today | Proposed |
|---|---|
| `katalyst rules` | `katalyst rules list` |
| `katalyst rules --family <f>` | `katalyst rules list --family <f>` |
| `katalyst rules <kind>` / `--kind <kind>` | `katalyst rules show <kind>` |

`--json` is preserved on both sub-verbs. The mapping is mechanical — the two
existing code paths (`runRulesList`, `runRulesDetail`) already split exactly
along the `list` / `show` line; this only moves the dispatch from flag/arg
sniffing into sub-commands. After the change, bare `katalyst rules` prints
help like every other noun.

This is the only behavioral change the principle requires. `check`, `fix`,
`init`, and the three resource nouns already conform.

## Open questions

1. **Migration for `rules`.** Is bare `katalyst rules` (and the positional
   `rules <kind>`) a hard break, or kept as a deprecated alias that maps to
   `rules list` / `rules show` with a stderr notice? Recommendation: hard
   break, matching `cli-spec.md`'s "hard rename, no alias" stance for the
   CRUD verbs — the tool is pre-1.0 and the alias cost outlives the
   convenience.
2. **`get` vs `show`.** Align `schema show` to `schema get` (or move
   `rules show` to `rules get`) so every read verb is one word? Tracked
   separately; not blocked by this spec.

## Test checklist

- [ ] `katalyst rules` (bare) prints help and exits 2 (or, if aliased, lists
      with a deprecation notice — pending Q1).
- [ ] `katalyst rules list` reproduces today's `katalyst rules` output.
- [ ] `katalyst rules list --family <f>` reproduces `rules --family <f>`.
- [ ] `katalyst rules show <kind>` reproduces today's `rules <kind>` detail.
- [ ] `--json` works on both `rules list` and `rules show`.
- [ ] No top-level command exists that is a bare CRUD verb.
- [ ] Each resource noun, invoked bare, prints help rather than acting.

## Graduation target

When this lands, the durable rule moves into permanent docs (per
[how-we-plan](../../docs/content/contributing/how-we-plan.md)):

- **`docs/deep-dives/`** — the noun/verb grammar and the placement rule, as
  CLI design rationale.
- **`docs/reference/`** — the `rules list` / `rules show` surface.
- **`AGENTS.md`** — a one-line pointer: new commands obey the placement rule.
