+++
title = "Formatting rationale"
weight = 60
+++

# Formatting rationale

*Why* `katalyst fix` works the way it does — in particular, why it is
opinionated and why it refuses to invent values.

## `fix` is deliberately opinionated

`katalyst fix` rewrites frontmatter in one canonical form:

- top-level keys sorted alphabetically,
- yaml.v3 default block style (no flow-style maps or sequences),
- strings unquoted where safe, double-quoted otherwise,
- exactly one trailing newline,
- body bytes preserved verbatim.

There are no style flags. `gofmt`, `black`, and `rustfmt` taught the same
lesson: a formatter's value comes from there being one obvious answer.
Configurability just re-creates the bikeshed. Users who want a different
style simply don't run `fix`. Because the body is preserved byte-for-byte,
`fix` is safe to run across an entire repo without touching prose.

**Trade-off:** comments inside the frontmatter block are not preserved. That
is by design — frontmatter is structured data, not prose — and will be
revisited only if it hurts in practice.

`--check` makes `fix` non-destructive: it writes nothing, prints the items
that *would* change, and exits 1. That is the CI form.

## Why `fix` never injects missing values

An earlier idea had a `--fix` mode that would add "sentinel" placeholder
values for missing required keys. It was dropped, and the safe-mutation
story moved to a later, opt-in command (working name `patch`).

Silently injecting placeholder values is hostile: it can mask real problems,
create merge conflicts, and produce documents that *pass* schema validation
while being semantically wrong. Katalyst would rather ship nothing than ship
that. A safer design — interactive, or constrained to filling a schema's
declared `default:` — deserves its own command and explicit per-field
opt-in. Until then, `fix` only ever normalizes what is already there; it
never creates structure (step 3 of its lifecycle returns a frontmatter-less
file untouched).

## See also

- [Domain model]({{< relref "domain-model.md" >}}) — the `fix` lifecycle in
  context.
- [Validate in CI]({{< relref "../how-to/validate-in-ci.md" >}}) — using
  `fix --check` as a gate.
