#!/bin/sh
# Shared guard for the agent-link refresh git hooks (post-checkout/-merge/-rewrite).
#
# Two guards, both biased toward NOT running:
#   1. Opt-in — does nothing unless this clone enabled it:
#        git config hooks.agentLinks true   (or: scripts/setup-agent-links.sh --enable)
#   2. Change-aware — when given a ref range, skips unless a path the sync
#      actually reads (AGENTS.md / *.AGENTS.md / .cursor/skills) changed.
#
# Usage: agent-links-hook.sh [<oldref> <newref>]
#   With two refs  -> run only if a relevant path changed between them.
#   With no refs   -> force a full sync (fallback when a range is unavailable).
#
# Failures are non-fatal (warn only).

[ "$(git config --bool hooks.agentLinks 2>/dev/null)" = "true" ] || exit 0

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

if [ -n "$1" ] && [ -n "$2" ]; then
  changed="$(git -C "$repo_root" diff --name-only "$1" "$2" -- \
    'AGENTS.md' '*/AGENTS.md' '.cursor/skills/' 2>/dev/null || true)"
  [ -n "$changed" ] || exit 0
fi

bash "$repo_root/scripts/setup-agent-links.sh" \
  || echo "warning: agent link setup refresh failed" >&2
