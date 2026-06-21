+++
title = "Formatting"
weight = 60
+++

# Formatting

Why `fix` rewrites frontmatter the way it does — and, just as importantly,
why it refuses to do more. This page explains the design rationale behind
the formatter; for the exact command surface, see the
[`fix` command reference]({{< relref "../reference/commands.md" >}}).

## Background

`fix` is katalyst's only mutating command. It reads each selected item,
re-emits its frontmatter in a single canonical form, and leaves everything
else untouched. The canonical form is deliberately narrow: top-level keys
sorted alphabetically, yaml.v3 default block style, exactly one trailing
newline on the file, and body bytes preserved verbatim.

The full per-item data flow lives in the [domain model]({{< relref
"domain-model.md" >}}) under *Lifecycle of `fix`*. This page is about the
*why*.

## Design rationale

**`fix` is opinionated so that it has nothing to configure.** A formatter
with options is a formatter that teams argue about and then pin in config.
By choosing one canonical shape — sorted keys, block style, one trailing
newline — `fix` becomes a pure function from "valid frontmatter" to
"canonical frontmatter." Running it twice is the same as running it once,
and two people running it on the same file get the same bytes. Determinism
is the whole point: `fix --check` can then be a meaningful CI gate, because
"would `fix` change this?" has one answer.

**`fix` formats; it does not author.** The formatter normalizes structure
that already exists. It never invents structure: a file with no frontmatter
is returned verbatim, because inventing an empty `---` block would be
authoring metadata the user never wrote. The same principle is why `fix`
refuses to inject missing values. If a schema requires a `status` field and
a document lacks it, `fix` will not add `status: ""` or guess a default —
that is content, not formatting, and katalyst does not put words in your
documents' mouths. Reporting the missing field is `check`'s job; supplying a
value is the author's.

**Body bytes are sacred.** This is a system-wide invariant, not a local
choice (see the [domain model]({{< relref "domain-model.md" >}}) invariants).
`fix` touches only the frontmatter region and the file's trailing newline;
interior body bytes round-trip exactly. A formatter you cannot trust with
your prose is a formatter you will not run.

## Trade-offs and alternatives

The cost of one canonical form is that it is *someone's* canonical form. Key
order is alphabetical rather than "most important first," and block style is
used even where flow style would be terser. These are arguable defaults, and
the deliberate choice is to make them un-arguable by not exposing knobs: a
configurable formatter trades a small ergonomic win for the determinism that
makes `fix --check` worth having.

An alternative would be a formatter that also fills in schema-required
defaults — a "fix everything" command. That was rejected on the same
content-vs-formatting line: the moment a tool writes values into your
documents, you have to review its judgment on every run, and the round-trip
guarantee is gone. Keeping `fix` to structure preserves a property worth
more than the convenience: you can run it blind, in bulk, in CI, and know it
changed nothing but shape.

## See also

- [`fix` command reference]({{< relref "../reference/commands.md" >}}) — the
  precise command surface and flags.
- [Domain model]({{< relref "domain-model.md" >}}) — the *Lifecycle of
  `fix`* steps and the body-bytes invariant.
- [Configuration]({{< relref "../reference/configuration.md" >}}) — how
  collections and checks are declared.
