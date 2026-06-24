#!/usr/bin/env bash
# Symlink each product skill under skills/ into .claude/skills/ so they auto-load
# in a working copy. Mirrors scripts/agent-link-utils.sh's
# sync_skill_links_from_cursor, but the source is the committed product skills
# (skills/), not the contributor skills (.cursor/skills/). .gitignore excludes
# all of .claude/, so these symlinks stay uncommitted.
#
# Placeholders are skipped: a skill with `status: placeholder` in its SKILL.md
# front matter is not yet shippable and should not auto-load.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skills_dir="$repo_root/skills"
target_dir="$repo_root/.claude/skills"
relative_skills_dir="../../skills"

if [ ! -d "$skills_dir" ]; then
  echo "No product skills directory found at $skills_dir" >&2
  exit 1
fi

mkdir -p "$target_dir"

find "$skills_dir" -mindepth 2 -maxdepth 2 -name SKILL.md -exec dirname {} \; | sort |
while IFS= read -r skill_dir; do
  skill_name="$(basename "$skill_dir")"

  if grep -qE '^status:[[:space:]]*placeholder' "$skill_dir/SKILL.md"; then
    echo "Skipping placeholder skill $skill_name"
    continue
  fi

  target="$target_dir/$skill_name"
  link_target="$relative_skills_dir/$skill_name"

  if [ -L "$target" ]; then
    if [ "$(readlink "$target")" = "$link_target" ]; then
      echo "Keeping skill $skill_name"
      continue
    fi
    rm "$target"
  elif [ -e "$target" ]; then
    echo "Skipping skill $skill_name: $target exists and is not a symlink" >&2
    continue
  fi

  ln -s "$link_target" "$target"
  echo "Linked skill $skill_name"
done
