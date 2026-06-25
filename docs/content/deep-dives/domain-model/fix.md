+++
title = "Fix"
weight = 62
+++

# Fix

Why [`katalyst fix`]({{< relref "../../reference/cli.md" >}}) rewrites
frontmatter the opinionated way it does. The parser and encoder live in
`internal/codec/markdownbodytext`; the transform that drives the canonical
form, and the backend write that persists it, live in `internal/fix` and
`internal/storage/collection/filesystem` respectively.

## Terms

| Term | Meaning |
|---|---|
| **Fix** | A command that rewrites existing content into Katalyst's canonical form when a check can supply a safe transformation. |
| **Canonical form** | The deterministic output format `fix` writes: preserved frontmatter syntax, sorted top-level keys, native encoder style, preserved body bytes, and one trailing newline. |
| **Report-only check** | A check that can report violations but cannot safely rewrite content. |
| **Check mode** | The `--check` form of `fix`: print what would change, write nothing, and exit 1 if any item is non-canonical. |

## Design rationale

**Fix is deliberately opinionated.**

`katalyst fix` rewrites frontmatter in one canonical form **in the file's own
format**: TOML stays TOML, JSON stays JSON, YAML stays YAML. `fix` never
converts between formats. Canonically, that means:

- the source format is preserved (same fence, same syntax),
- top-level keys sorted alphabetically,
- each format's default block/indent style: yaml.v3 block style, the `go-toml`
  default, two-space-indented JSON,
- exactly one trailing newline,
- body bytes preserved verbatim.

Because the canonical scalar styling is each library's default, a round-trip is
*meaning*-preserving rather than byte-identical: e.g. a double-quoted TOML
string re-emits single-quoted. Re-parsing the output always yields the same
`Meta`.

There are no style flags. `gofmt`, `black`, and `rustfmt` taught the same
lesson: a formatter's value comes from there being one obvious answer.
Configurability just re-creates the bikeshed. Users who want a different style
simply don't run `fix`. Because the body is preserved byte-for-byte, `fix` is
safe to run across an entire repo without touching prose.

**Trade-off:** comments inside the frontmatter block are not preserved. That is
by design (frontmatter is structured data, not prose) and will be revisited only
if it hurts in practice.

`--check` makes `fix` non-destructive: it writes nothing, prints the items that
*would* change, and exits 1. That is the CI form.

**Fix never injects missing values.**

An earlier idea had a mode that would add "sentinel" placeholder values for
missing required keys. It was dropped, and the safe-mutation story moved to a
later, opt-in command (working name `patch`).

Silently injecting placeholder values is hostile: it can mask real problems,
create merge conflicts, and produce documents that *pass* schema validation
while being semantically wrong. Katalyst would rather ship nothing than ship
that. A safer design, interactive or constrained to filling a schema's declared
`default:`, deserves its own command and explicit per-field opt-in. Until then,
`fix` only ever normalizes what is already there; it never creates structure (a
frontmatter-less file is returned untouched).

## Worked example

{{< katalyst-example-full "fix-normalize-frontmatter" >}}

## Lifecycle of fix

For each item:

1. Read bytes.
2. Parse to `Document`.
3. If no frontmatter, return verbatim, `fix` never invents structure.
4. Marshal `Meta` with top-level keys sorted alphabetically, in
   `Document.Format`'s native syntax and default style.
5. Re-assemble in the same format: `---\n<yaml>\n---\n<body>`,
   `+++\n<toml>\n+++\n<body>`, or `{...}\n<body>` for JSON. Body bytes are
   preserved verbatim; one trailing newline is enforced on the file.
6. Compare against the original. If unchanged, do nothing. Otherwise atomically
   rewrite (temp file + rename), or, with `--check`, print the path and
   accumulate exit-1 status.

## Invariants

1. **Body bytes are sacred.** No command except `fix` modifies them. Even `fix`
   only normalizes trailing whitespace and the leading separator; interior body
   bytes round-trip exactly.
2. **Format is preserved.** `fix` re-emits each file in its own frontmatter
   syntax and never converts between YAML, TOML, and JSON.
3. **No semantic values are invented.** `fix` only normalizes existing
   frontmatter and configured text fixes; it does not create missing metadata.

## See also

- [Markdown body text]({{< relref "../../reference/item-views/markdown-body-text.md" >}})
  for how markdown documents parse before `fix` rewrites them.
- `go doc ./internal/fix` for the code-level transform contract.
