#!/bin/bash
set -euo pipefail

# Repo-specific setup for Claude Code on the web. Local sessions exit early.
if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR"

# Link contributor skills and AGENTS.md mirrors into .claude/ so they
# auto-load. The .claude/ mirror tree stays untracked apart from this hook
# and settings.json (see .gitignore).
./scripts/setup-claude-code.sh || echo "warning: setup-claude-code.sh failed; contributor skills and CLAUDE.md links may be missing" >&2

# Link the shippable product skills under skills/ into .claude/skills/.
./scripts/link-product-skills.sh || echo "warning: link-product-skills.sh failed; product skills may not auto-load" >&2

# Pre-fetch Go modules for the application module and the docs Hugo module
# so make test/vet/build and docs builds don't need further network access.
# Non-fatal: under a strict network policy the cached container may already
# have them, and a missing module surfaces at build time anyway.
go mod download || echo "warning: go mod download failed (restricted network?); builds may fetch on demand" >&2
(cd docs && go mod download) || echo "warning: docs module download failed; Hugo docs builds may need network access" >&2

# Build the katalyst binary; the repo dogfoods itself on its docs
# (./bin/katalyst check), so sessions expect ./bin/katalyst to exist.
make build || echo "warning: make build failed; run 'make build' before './bin/katalyst check'" >&2
