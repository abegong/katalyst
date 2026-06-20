+++
title = "Why Katalyst?"
weight = 10
+++

# Why Katalyst?

Katalyst is a **content consistency layer**, built especially with agent memory,
wikis, and knowledge bases in mind. The bet: structure should be *light* — cheap
to add, cheap to change — so an agent stays consistent without you restating the
rules every session. For the full argument, see
[Vision and scope]({{< relref "deep-dives/vision.md" >}}).

{{% hint info %}}
**New to Katalyst?** [Get started]({{< relref "getting-started.md" >}}) — install
the CLI, scaffold a `.katalyst/` project, and run your first checks in a few
minutes.
{{% /hint %}}

## What problems does it solve?

- **"I've told it how to organize things — more than once — and it's still
  inconsistent."** Encode the conventions as checks instead of repeating them in
  prompts.
- **"It burns tokens searching, and still misses things."** A known structure
  makes content addressable, not something to rediscover on every task.
- **"As the knowledge base grows, it gets top-heavy."** Validation keeps a
  large, evolving store navigable and safe to change.
- **"I can't fully delegate — it stores things in the wrong place and skips the
  details I care about."** Rules pin down where things go and what each item
  must capture.

## Who is it for?

<!-- TODO: refine personas before publishing -->

- **Agent builders** handing an agent a memory store, wiki, or knowledge base to
  maintain.
- **Teams with growing semi-structured content** that needs to stay navigable
  and safe to change.
- _More personas to come._

## Get started

- [Build the CLI and run your first checks]({{< relref "getting-started.md" >}}).
- [Contribute]({{< relref "contributing/_index.md" >}}) — how we plan and
  document changes.
