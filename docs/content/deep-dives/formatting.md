+++
title = "Frontmatter and fix"
weight = 60
+++

# Frontmatter and fix

How Katalyst parses a markdown file's frontmatter, the in-memory document that
produces, and why [`fix`]({{< relref "../reference/cli.md" >}}) rewrites
that frontmatter the opinionated way it does. The codec (parse and encode) lives
in `internal/storage/collection/document`; the `fix` transform that drives the
canonical form, and the backend write that persists it, live in `internal/fix`
and `internal/storage/collection/filesystem` respectively.

## The markdown document

The unit of work is a file on disk with two optional regions:

- A **frontmatter** block at the very top of the file, in one of three formats
  detected by the opening fence:

  | Format | Fence | Example openers |
  |--------|-------|-----------------|
  | YAML   | `---` | Jekyll, Obsidian, Hugo |
  | TOML   | `+++` | Hugo, Obsidian, Jekyll |
  | JSON   | `{` ... `}` | Hugo |

  These are the three formats Hugo, Obsidian, and Jekyll emit. Whatever the
  source format, the parsed `Meta` is a plain `map[string]any`, so checks and
  inspectors never branch on format. `Document.Format` records the detected
  syntax so `fix` can re-emit a file in its own format rather than rewriting,
  say, TOML as YAML.
- A **body**, everything after the closing fence.

A document *may* have no frontmatter, in which case `check` reports it as an
error (the file claimed no metadata, so we couldn't check anything).

When parsed, a markdown document becomes a `document.Document`:

| Field            | Meaning |
|------------------|---------|
| `HasFrontmatter` | Did the file open with a recognized fence? |
| `Format`         | Detected syntax: `KindYAML`, `KindTOML`, or `KindJSON` |
| `Meta`           | Parsed frontmatter, normalized to `map[string]any` |
| `Body`           | Bytes after the closing fence, **never modified** except by `fix` |
| `Lines`          | JSON-pointer-path to 1-indexed source line |

The `Lines` index is what makes error messages locatable. It accounts for the
opening fence offset, so `Lines["/title"] = 2` means the `title` key is on line
2 of the original file.

**Line tracking is full for YAML only.** For TOML and JSON, `Lines` is empty
today; checks degrade gracefully (they emit the error without a line number).
Richer line tracking for the other formats is a planned follow-up.

## Why fix is deliberately opinionated

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

## Worked example

{{< katalyst-example-full "fix-normalize-frontmatter" >}}

## Why fix never injects missing values

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
2. **Line numbers are file-relative and 1-indexed.** The opening fence is line
   1, so the first key is typically line 2. (Populated for YAML today; see the
   line-tracking note above.)
3. **Format is preserved.** `fix` re-emits each file in its own frontmatter
   syntax and never converts between YAML, TOML, and JSON.
