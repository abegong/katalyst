#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
codex_skills_dir="$repo_root/.codex/skills"

source "$repo_root/scripts/agent-link-utils.sh"

sync_skill_links_from_cursor "$repo_root" "$codex_skills_dir" "Codex"
echo "Done. Restart Codex to pick up repo-local skills."
