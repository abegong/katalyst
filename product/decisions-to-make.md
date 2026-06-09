# Decisions to make

Open design questions. Resolved decisions move to `decisions.md`.

## v1 command surface — beyond what's already implemented

Implemented so far: `init`, `validate`, `schema list`, `schema show`,
`fmt`. Candidates for the rest of v1, roughly in order of usefulness:

- `katalyst schema check` — sanity-check the schema files themselves
  (valid JSON Schema, no dangling `$ref`s, etc.).
- `katalyst ls` — list files and the schema each one matched against
  (great for debugging association rules).
- `katalyst explain <path>` — show, for one file, which schema matched,
  why, and what the validation result was.
- `katalyst infer <paths...>` — synthesize a starter schema from existing
  files. (See roadmap v0.4.)
- `katalyst watch` — re-validate on save.

Naming question: do we keep `validate` or rename to `check`? `check`
reads more like "is this OK?", `validate` is the Mongo term. Currently
keeping `validate`; reopen if we find a cleaner partition (e.g.
`check` for schema-files-are-sane, `validate` for documents-conform).

## Schema-on-schema validation

Should we validate user-supplied schemas against the JSON Schema
meta-schema at load time? The `santhosh-tekuri/jsonschema/v6` library
does some of this implicitly during compile, but we may want a louder
"your schema is malformed" message. Probably belongs in a future
`schema check` subcommand.

## `--allow-unmatched` and friends

D2 in `decisions.md` says unmatched files are errors. We'll likely want
escape hatches:

- `--allow-unmatched` on `validate` to downgrade to warning.
- A config-level `unmatched: error | warn | skip` knob.
- A way to mark whole directories as "metadata-free" (probably via a
  rule with `schema: null`).

Defer until we see real usage.

## Config file format

YAML is locked in for v0.1 (see D1). Open question: do we ever support
JSON as a fallback (for users who want to author the config from
another tool)? Probably yes, eventually, but not until someone asks.

## JSON Schema draft

Which draft do we target by default? Proposal: draft 2020-12 (latest),
but allow schemas to declare `$schema` explicitly and validate against
the declared draft. `santhosh-tekuri/jsonschema/v6` already supports
drafts 4 through 2020-12, so this is just a question of what we put in
the `init` template and what we document.

## Vocabulary

- "schema" vs "validator": settled on "schema" in user-facing copy.
- "frontmatter" vs "metadata": settled on "frontmatter" for the on-disk
  block, "metadata" for the parsed structure.

(These can move to `decisions.md` next time we touch it.)
