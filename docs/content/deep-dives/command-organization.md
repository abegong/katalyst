+++
title = "How the core commands are organized"
weight = 50
+++

# How the core commands are organized

The `katalyst` CLI carries two command grammars on purpose, and keeps them
apart. Knowing which grammar a command belongs to tells you how to invoke it,
what its selector means, and — when you add a command — where it goes.

## Two families

A top-level command is exactly one of two kinds.

**Blessed verbs** operate *over* content. The verb names a cross-cutting
operation; a [selector]({{< relref "../reference/commands.md" >}}) picks the
targets at any depth, and several may be passed at once:

```bash
katalyst check                    # whole project
katalyst check notes              # one collection
katalyst fix notes/dune schemas   # mixed depth, many targets
```

`check` and `fix` are the blessed verbs. `init` is a lone **project verb** —
a lifecycle operation on the project itself, with no noun and no selector.

**Resource nouns** operate *on* one kind of resource. The noun names the
resource type; a CRUD-shaped sub-verb does the work, at a fixed selector
depth:

```bash
katalyst collection list          # the set
katalyst item get notes/dune      # one item, depth 2
katalyst rules show object_required_field
```

`collection`, `item`, `schema`, and `rules` are resource nouns. A noun
invoked bare (`katalyst item`) prints help — it is never itself an action.

## The placement rule

When a new command arrives, one question decides its shape:

> Does this operation span resource kinds, or administer exactly one?

| Answer | Family | Shape |
|---|---|---|
| Spans kinds; about the content | Blessed verb | `katalyst <verb> [selector ...]` |
| Administers one resource kind | Noun sub-verb | `katalyst <noun> <verb> <selector>` |
| Acts on the project itself | Project verb | `katalyst <verb> [flags]` |

Concretely, the rule forbids three things:

- **No top-level CRUD verbs.** CRUD lives under its noun — `item add`, never a
  bare `add` that has to guess its noun.
- **No cross-cutting verb buried under a noun.** A content-wide operation
  stays a blessed verb; `check` does not become `item check` /
  `collection check`.
- **No bare noun as an action.** A noun with no sub-verb shows help; it never
  silently runs a default.

## Why separate them

The split is what makes the surface predictable for the humans and agents who
drive it:

- **One mental model per command.** "Is this an operation over my content, or
  on one resource?" answers how to call it before you read the help.
- **Selectors stay coherent.** Blessed verbs take a selector at any depth and
  accept many; noun sub-verbs fix the depth and report a wrong one as a usage
  error (exit 2). Mixing the two grammars would blur that contract.
- **It scales without churn.** A new resource becomes a noun with sub-verbs; a
  new cross-cutting operation becomes a blessed verb. Neither forces a
  redesign of the other.

The grouping is visible at the surface: `katalyst` with no arguments lists
**Verbs** and **Resources** under separate headings rather than one
alphabetized blob, so the grammar reads off the help screen.

`rules` was the last command brought into this scheme: it began as a bare noun
that listed when invoked directly, and was split into `rules list` /
`rules show` so it matches the other resource nouns.
