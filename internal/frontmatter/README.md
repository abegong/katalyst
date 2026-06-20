# internal/frontmatter

Parses and formats markdown frontmatter, with source-line tracking, and owns
the canonical form `katalyst fix` rewrites files into.

## The markdown document

The unit of work. A file on disk with two optional regions:

- A **frontmatter** block, fenced by `---` lines at the very top of the
  file. YAML today; TOML / JSON are planned.
- A **body**, everything after the closing fence.

A document *may* have no frontmatter, in which case `check` reports it as an
error (the file claimed no metadata, so we couldn't check anything).

When parsed, a markdown document becomes a `frontmatter.Document`:

| Field            | Meaning |
|------------------|---------|
| `HasFrontmatter` | Did the file open with `---`? |
| `Meta`           | Parsed YAML, normalized to `map[string]any` |
| `Body`           | Bytes after the closing fence, **never modified** except by `fix` |
| `Lines`          | JSON-pointer-path → 1-indexed source line |

The `Lines` index is what makes error messages locatable. It accounts for
the opening `---` fence offset, so `Lines["/title"] = 2` means the
`title:` key is on line 2 of the original file.

## Why `fix` is deliberately opinionated

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

## Lifecycle of `fix`

For each item:

1. Read bytes.
2. Parse to `Document`.
3. If no frontmatter, return verbatim — `fix` never invents structure.
4. Marshal `Meta` with top-level keys sorted alphabetically, yaml.v3
   default block style.
5. Re-assemble: `---\n<sorted yaml>\n---\n<body>`. Body bytes are
   preserved verbatim; one trailing newline is enforced on the file.
6. Compare against the original. If unchanged, do nothing. Otherwise
   atomically rewrite (temp file + rename) — or, with `--check`, print the
   path and accumulate exit-1 status.

## Invariants

1. **Body bytes are sacred.** No command except `fix` modifies them. Even
   `fix` only normalizes trailing whitespace and the leading separator;
   interior body bytes round-trip exactly.
2. **Line numbers are file-relative and 1-indexed.** The opening `---`
   fence is line 1, so the first YAML key is typically line 2.
