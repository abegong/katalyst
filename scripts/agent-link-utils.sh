#!/usr/bin/env bash

sync_skill_links_from_cursor() {
  local repo_root="$1"
  local target_dir="$2"
  local tool_name="$3"
  local cursor_skills_dir="$repo_root/.cursor/skills"
  local relative_cursor_skills_dir="../../.cursor/skills"

  if [ ! -d "$cursor_skills_dir" ]; then
    echo "No Cursor skills directory found at $cursor_skills_dir" >&2
    return 1
  fi

  mkdir -p "$target_dir"

  find "$cursor_skills_dir" -mindepth 2 -maxdepth 2 -name SKILL.md -exec dirname {} \; | sort |
  while IFS= read -r skill_dir; do
    local skill_name
    local target
    local link_target

    skill_name="$(basename "$skill_dir")"
    target="$target_dir/$skill_name"
    link_target="$relative_cursor_skills_dir/$skill_name"

    if [ -L "$target" ]; then
      if [ "$(readlink "$target")" = "$link_target" ]; then
        echo "Keeping $tool_name skill $skill_name"
        continue
      fi

      rm "$target"
    elif [ -e "$target" ]; then
      echo "Skipping $tool_name skill $skill_name: $target already exists and is not a symlink" >&2
      continue
    fi

    ln -s "$link_target" "$target"
    echo "Linked $tool_name skill $skill_name"
  done

  find "$target_dir" -maxdepth 1 -type l |
  while IFS= read -r target; do
    local skill_name
    local expected_link_target

    skill_name="$(basename "$target")"
    expected_link_target="$relative_cursor_skills_dir/$skill_name"

    if [ "$(readlink "$target")" != "$expected_link_target" ]; then
      continue
    fi

    if [ -f "$cursor_skills_dir/$skill_name/SKILL.md" ]; then
      continue
    fi

    rm "$target"
    echo "Pruned stale $tool_name skill $skill_name"
  done
}
