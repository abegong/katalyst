#!/usr/bin/env bash
set -euo pipefail

# Entry point for refreshing agent links.
#
#   scripts/setup-agent-links.sh            Run a full sync now.
#   scripts/setup-agent-links.sh --enable   Turn the git hooks on for this clone, then sync.
#   scripts/setup-agent-links.sh --disable  Turn the git hooks off for this clone.
#
# The opt-in flag lives in this clone's local git config (hooks.agentLinks). It is
# not committed, is per-developer, and covers every worktree of the clone.

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

case "${1:-}" in
  --enable)
    git -C "$repo_root" config hooks.agentLinks true
    git -C "$repo_root" config core.hooksPath .githooks
    echo "Enabled agent-link git hooks for this clone (hooks.agentLinks=true, core.hooksPath=.githooks)."
    ;;
  --disable)
    git -C "$repo_root" config --unset hooks.agentLinks 2>/dev/null || true
    echo "Disabled agent-link git hooks for this clone. Run with --enable to turn them back on."
    exit 0
    ;;
  "")
    ;;
  *)
    echo "Usage: scripts/setup-agent-links.sh [--enable|--disable]" >&2
    exit 2
    ;;
esac

bash "$repo_root/scripts/setup-codex-skills.sh"
bash "$repo_root/scripts/setup-claude-code.sh"
