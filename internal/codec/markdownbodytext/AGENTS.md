# internal/codec/markdownbodytext

The markdown body text codec. It parses optional YAML, TOML, or JSON
frontmatter plus the markdown body into a format-neutral `Document`, and encodes
documents back to bytes while preserving the source frontmatter format.

Checks, inspectors, storage readers, and fix operations share this codec so
they agree on metadata, body text, and source line information.

## Conventions

- Keep this package a leaf. It should not import other `internal/` packages;
  callers bring parsed documents into their own domain types.
- Preserve source format on encode. Do not normalize TOML or JSON frontmatter to
  YAML unless the caller explicitly asks for that behavior through a new API.
- Line tracking is part of diagnostics. YAML currently has full frontmatter
  line mapping; TOML and JSON degrade gracefully. Tests for parser changes
  should include line-number expectations when behavior affects diagnostics.
- Use byte-exact inline test inputs for small parse/encode cases. Reach for
  fixtures only when the same document is shared across tests.
