+++
title = "How-to page template"
weight = 40
draft = true
+++

<!--
TEMPLATE — copy this file into docs/how-to/ and fill it in.
Derived from how-to/configure-rules.md.

This page IS: a task-oriented recipe for a reader who already knows the
basics and has a goal ("I want to X"). It is a sequence of steps that
accomplish that one goal.

This page is NOT: a tutorial (the reader is not learning the tool, they have
a job to do), a reference (link to it for the full option list), or an
explanation (link to it for the why).

Title it as the task, starting with a verb. Keep steps minimal and ordered.
-->

# <Verb the task>

One sentence naming the goal and the situation the reader is in.

## 1. <First action>

```yaml
<the minimal config or command for this step>
```

State defaults and the one decision the reader makes here, if any.

## 2. <Next action>

...continue with the smallest number of ordered steps that reach the goal...

## 3. Run it

```bash
<the command that confirms the task is done>
```

What success looks like.

## See also

- The related how-to for the adjacent task, and the reference page for full
  options (use `relref` once you copy this into `docs/how-to/`).
