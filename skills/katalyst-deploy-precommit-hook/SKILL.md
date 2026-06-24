---
name: katalyst-deploy-precommit-hook
description: >-
  Install a git pre-commit hook that runs `katalyst check` (and `katalyst fix
  --check`) so malformed frontmatter or non-canonical formatting fail the commit
  instead of landing. Use when a user wants katalyst enforced at commit time, a
  "pre-commit hook", or to block bad content before it reaches the branch. Paired
  with katalyst-deploy (chooses the mechanism) and katalyst-deploy-cli-gating
  (the alternative).
---

# Deploy: pre-commit hook

Wire `katalyst check` into a git pre-commit hook so a commit that would introduce
a violation fails before it lands. This is the simplest durable enforcement for
git-backed content: it catches both human and agent commits, and needs no daemon
or wrapper.

If you are not sure this is the right mechanism, start with **katalyst-deploy**,
which compares this against CLI-gated writes.

## Prerequisites

- `katalyst` on `PATH` (run `./bootstrap.sh` if not).
- A git repository with a `.katalyst/` config at its root.
- `katalyst check` passing today, so the hook starts from a clean baseline.

## What the hook runs

Two commands, matching the CI gate (see the validation exit-code convention:
`0` pass, `1` violations, `2` usage error):

- `katalyst check` — schema and structural checks over the project.
- `katalyst fix --check` — fails if any frontmatter is non-canonical, without
  writing.

## Install

Most projects should use the repo's hook mechanism rather than a bare
`.git/hooks/` file, so the hook is versioned and shared. If the project uses the
[pre-commit](https://pre-commit.com) framework, add a local hook; otherwise write
`.git/hooks/pre-commit` directly. A ready-to-copy script is bundled at
`references/pre-commit`.

Direct install:

```bash
cp references/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

Then verify the gate fires by staging a change that violates a check and
confirming the commit is rejected with the diagnostic.

## Notes

- The hook is bypassable with `git commit --no-verify`; that is expected — it is a
  guardrail, not a lock. Pair it with the same checks in CI (see the project's
  CI configuration) so a bypassed commit still fails the build.
- Keep the hook fast: it runs `check` over the whole project by default. For very
  large corpora, scope it to staged files if commit latency becomes a problem.
