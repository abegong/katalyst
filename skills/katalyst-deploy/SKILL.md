---
name: katalyst-deploy
description: >-
  Set up automatic enforcement of a katalyst project once, so its checks run on
  every change without anyone remembering to invoke them. Knows both mechanisms
  (a git pre-commit hook and CLI-gated writes), helps choose between them, and
  routes to the specific setup skill. Use when a user wants katalyst to "run
  automatically", "enforce on commit", "guard the content", or stop relying on
  manual `katalyst check` runs.
---

# Deploy katalyst enforcement

Katalyst checks are only a guardrail when they run on their own. Asking an agent
(or a person) to remember `katalyst check` before every change is fragile — the
check that matters is the one nobody had to remember. This skill sets enforcement
up **once** so it runs structurally from then on, no matter who or what does the
writing.

## Prerequisites

The `katalyst` CLI must be on `PATH`. If it is not, run the bundled bootstrap:

```bash
./bootstrap.sh
```

The project must already have a `.katalyst/` config (run `katalyst init` first if
not). Confirm checks pass today before wiring them into a gate:

```bash
katalyst check
```

## Choose a mechanism

There are two ways to make enforcement automatic. They are not exclusive — a
team can run both — but start with one.

| | Pre-commit hook | CLI-gated writes |
|---|---|---|
| **When it runs** | At `git commit` time | At write time, before the change lands |
| **Catches** | Anything staged for commit | Every write routed through the CLI |
| **Best for** | Git-backed content, human + agent commits | Agent workflows where writes should be validated as they happen |
| **Bypassable by** | `git commit --no-verify` | Writing to the files directly, outside the gate |
| **Setup skill** | **katalyst-deploy-precommit-hook** | **katalyst-deploy-cli-gating** |

Recommendation: if the content lives in a git repository and commits are the
natural checkpoint, **start with the pre-commit hook** — it is the simplest
durable gate and catches both human and agent changes. Reach for CLI gating when
writes need to be validated the moment they happen, before a commit exists, or
when there is no git workflow to hang the hook on.

## Route

- Pre-commit hook → follow **katalyst-deploy-precommit-hook**.
- CLI-gated writes → follow **katalyst-deploy-cli-gating**.

Both leave enforcement running automatically. Day-to-day, no one needs to invoke
katalyst by hand and no skill re-runs the check on every change — that is the
point of deploying it once.
