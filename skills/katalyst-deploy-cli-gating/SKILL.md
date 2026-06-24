---
name: katalyst-deploy-cli-gating
description: >-
  Gate writes to a katalyst project's content so every change is validated as it
  happens, rather than only at commit time. Use when a user wants writes checked
  the moment they occur, an agent workflow where edits must pass checks before
  they land, or enforcement without a git commit step. Paired with katalyst-deploy
  (chooses the mechanism) and katalyst-deploy-precommit-hook (the alternative).
---

# Deploy: CLI-gated writes

Route changes to the project's content through `katalyst` so each write is
validated immediately, before the change is considered done. Unlike the
pre-commit hook, this does not wait for a commit — it is the right gate for agent
workflows that edit content continuously and for content with no git checkpoint.

If you are not sure this is the right mechanism, start with **katalyst-deploy**,
which compares this against the pre-commit hook.

## Prerequisites

- `katalyst` on `PATH` (run `./bootstrap.sh` if not).
- A `.katalyst/` config, with `katalyst check` passing today.

## The gate

The mechanism is a discipline plus a check, not a lock on the filesystem: after
any write to the project's content, run `katalyst check` scoped to what changed
and treat a non-zero exit as "the write is not done."

For a single item, pass its selector so the gate is fast:

```bash
# After writing collection "notes", item "intro":
katalyst check notes/intro
```

A non-zero exit (`1` = violations, `2` = usage error) means the change must be
revised before it counts as landed. Re-run until it exits `0`.

## Make it structural for an agent

For an agent driving the CLI, encode the gate as a rule it cannot skip rather
than a step it might forget:

1. **Write, then check, every time.** After each edit to content under a
   collection, immediately run `katalyst check <collection>/<item>` and do not
   move on until it passes.
2. **Treat a failure as unfinished work,** not an error to report and ignore:
   read the diagnostic, fix the frontmatter or body, re-check.
3. **Fix formatting in the same pass** with `katalyst fix <selector>` so
   frontmatter stays canonical as you go.

## Notes

- This gate validates content the agent routes through it; a write made directly
  to the files, outside the discipline, is not caught. Where commits are the
  durable checkpoint, pair this with **katalyst-deploy-precommit-hook** so an
  ungated write still fails before it merges.
- Scope checks to the changed selector for speed; reserve a full-project
  `katalyst check` for a final sweep.
