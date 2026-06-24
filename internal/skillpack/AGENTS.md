# internal/skillpack

Packages product skills from `skills/` into `.skill` archives for distribution.
The `cmd/skillpack` command is only the CLI wrapper; archive discovery,
frontmatter parsing, placeholder filtering, and zip creation live here so they
can be tested directly.

## Conventions

- A packaged skill archive has `SKILL.md` at the archive root, not nested under
  the skill directory name. Preserve that shape; clients depend on it.
- `status: placeholder` skills are not shippable. Discovery should keep
  reporting them, but package commands must skip or reject them.
- The shared `bootstrap.sh` is copied into each archive when present and should
  keep its executable bit. Use zip headers that preserve file modes.
- Tests should exercise the package API with temp directories and inspect the
  produced zip contents directly, rather than shelling out to `cmd/skillpack`.
