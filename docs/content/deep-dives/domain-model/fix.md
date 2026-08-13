+++
title = "Fix"
weight = 62
+++

# Fix

Why [`katalyst fix`]({{< relref "../../reference/cli.md" >}}) rewrites
frontmatter the opinionated way it does. The parser and encoder live in
`internal/codec/markdownbodytext`, and the transform that drives the canonical
form lives in `internal/fix`, which does no IO. Reading and persisting go
through the item's base, so `fix` never assumes its content came from a file.

## Terms

| Term | Meaning |
|---|---|
| **Fix** | A command that rewrites existing content into Katalyst's canonical form when a check can supply a safe transformation. |
| **Canonical form** | The deterministic output format `fix` writes: preserved frontmatter syntax, sorted top-level keys, native encoder style, preserved body bytes, and one trailing newline. |
| **Report-only check** | A check that can report violations but cannot safely rewrite content. |
| **Check mode** | The `--check` form of `fix`: print what would change, write nothing, and exit 1 if any item is non-canonical. |
| **Text content** | The body an item exposes for `fix` to rewrite. Every filesystem item has one; a SQLite item has one when its collection maps a content column. |

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

**Fix is a text-form verb.**

`fix` operates on an item's serialized text, so it needs a collection that
exposes one. Every filesystem collection does, because the file *is* the text. A
SQLite collection does when it maps a content column. A collection of attributes
alone has no serialized form to canonicalize, and `fix` says so rather than
guessing:

```
fix requires a text content mapping; collection "notes" maps only attributes
```

The gate is that capability, not the backend's name. Naming the backend would be
the wrong test twice over: it would refuse a SQLite collection that does have a
body, and it would need a new branch for every base type added later.

The two halves of `fix` divide unevenly across backends, which is what makes the
capability the right question. Configured text fixes rewrite the body, and a
content column is a body like any other. Frontmatter canonicalization, though,
cannot change a row: a SQLite item's frontmatter is synthesized from its
attributes when the item is read, so it arrives sorted and canonically styled
already. Nothing is left for the canonical pass to do, which is why `fix` writes
the content column and leaves the attribute columns untouched.

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

1. Read the item's bytes through its base. A file is read as it sits on disk; a
   SQLite row is assembled into a document from its attributes and content
   column.
2. Parse to `Document`.
3. If no frontmatter, return verbatim, `fix` never invents structure.
4. Marshal `Meta` with top-level keys sorted alphabetically, in
   `Document.Format`'s native syntax and default style.
5. Re-assemble in the same format: `---\n<yaml>\n---\n<body>`,
   `+++\n<toml>\n+++\n<body>`, or `{...}\n<body>` for JSON. Body bytes are
   preserved verbatim; one trailing newline is enforced on the file.
6. Compare against the original. If unchanged, do nothing. Otherwise write it
   back through the base, a file by atomic replace (temp file + rename) and a
   row by updating its content column, or, with `--check`, print the item and
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

- [Markdown body text]({{< relref "../../reference/data-surfaces/markdown-body-text.md" >}})
  for how markdown documents parse before `fix` rewrites them.
- `go doc ./internal/fix` for the code-level transform contract.
