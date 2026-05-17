# Decisions to make

Open design questions. Each one should eventually move to a "Decisions" log
(or just disappear into the code) once resolved.

## Schema ↔ file association

How does `katabridge` know which schema applies to which markdown file?
The likely answer is "a combination," but we need to decide precedence and
defaults.

Candidate mechanisms:

1. **Global config with glob → schema map.** A `katabridge.yaml` at the repo
   root lists rules like:
   ```yaml
   schemas:
     book:   ./schemas/book.json
     person: ./schemas/person.json
   rules:
     - paths: "notes/books/**/*.md"
       schema: book
     - paths: "notes/people/**/*.md"
       schema: person
   ```
2. **Inline `schema:` key in frontmatter.** Each file names its own schema:
   ```yaml
   ---
   schema: book
   title: Dune
   ---
   ```
3. **Directory-local config.** A `.katabridge.yaml` per directory, with
   nearest-ancestor wins (like `.editorconfig`).
4. **Schema-side path declarations.** The schema itself declares which paths
   it applies to (inverted from #1).

Open questions:
- What wins when multiple mechanisms disagree? Proposal: inline `schema:` >
  directory-local > global glob rules.
- Should an unmatched file be an error, a warning, or silently skipped?
  Proposal: configurable, default = warning.
- Can a single file match multiple schemas (composed validation)? Proposal:
  yes, all must pass; this is useful for "base + specialization" schemas.

## v1 command surface — beyond the core three

In addition to `init`, `validate`, `schema list`, what belongs in v1?

Candidates, roughly in order of usefulness:

- `katabridge schema show <name>` — print a schema.
- `katabridge schema check` — sanity-check the schema files themselves
  (valid JSON Schema, no dangling `$ref`s, etc.).
- `katabridge check` — alias for `validate` with a friendlier name? Or
  reserve `check` for "schema files are well-formed" and `validate` for
  "documents conform to schemas"? Mongo uses `validator`/`validate`.
- `katabridge ls` — list files and the schema each one matched against
  (great for debugging association rules).
- `katabridge explain <path>` — show, for one file, which schema matched,
  why, and what the validation result was.
- `katabridge infer <paths...>` — synthesize a starter schema from existing
  files. (Probably v0.4 — see roadmap.)
- `katabridge fmt` — normalize frontmatter. (Probably v0.2.)
- `katabridge watch` — re-validate on save.

## Naming / vocabulary

- Do we say "schema" or "validator" (Mongo's term)? Proposal: "schema" in
  user-facing copy, "validator" only when referring to the runtime check.
- Do we say "frontmatter" or "metadata"? Proposal: "frontmatter" when
  talking about the on-disk YAML/TOML/JSON block; "metadata" when talking
  about the parsed structure.

## Config file format

`katabridge.yaml` vs `katabridge.json` vs `katabridge.toml` vs `.katabridgerc`?
Proposal: YAML by default, accept JSON as a fallback. Revisit if we add
schemas-in-config (where JSON Schema authoring inline might be nicer in
JSON).

## JSON Schema draft

Which draft do we target? Proposal: draft 2020-12 (latest), but allow
schemas to declare `$schema` explicitly and validate against the declared
draft. `santhosh-tekuri/jsonschema/v6` supports drafts 4 through 2020-12.

## Exit codes

What should `validate` exit with on partial failure?
- `0` — all valid
- `1` — one or more validation failures (expected, machine-readable)
- `2` — usage error / unreadable input
- `>=3` — internal error

This matches conventions of linters like `shellcheck`. Worth confirming.
