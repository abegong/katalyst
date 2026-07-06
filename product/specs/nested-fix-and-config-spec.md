# Spec - nested fix and config paths

> **Status: planning.** Follow-on to the nested config authority work. This
> finishes two authority surfaces that are already visible in the branch:
> delegated `fix` behavior and non-default delegate `config:` directories.

## Overview

The nested config authority branch exposes two concepts that need to be real
before the feature ships:

- `authority.fix`, which says which config supplies canonicalization for
  `katalyst fix` and `fix --check`;
- delegate `config:`, which says where the child config directory lives relative
  to the delegated subtree.

Both are useful, but partial support would create false guarantees. A user who
sets `fix: file_nearest` expects child-owned files to be checked and fixed by
the child project during parent-root whole-project runs. A user who writes
`config: katalyst` expects Katalyst to load `<path>/katalyst/`, not silently
fall back to `<path>/.katalyst/`.

This spec makes those surfaces explicit and implementable.

## Value

These two pieces close the gap between the authority model and user-visible
behavior:

- Root hooks and CI can run `katalyst fix --check` once and still catch dirty
  child-owned content.
- Parent-root `fix` does not rewrite child-owned files with parent collection
  rules when the child owns fixing.
- Repositories that cannot or do not want to name the child config directory
  `.katalyst/` can still be delegated from the root.
- `project plan` and diagnostics report the actual child config path.

## Current State

In the current branch, `fix` accepts the same config-selection flags as `check`,
but it resolves selectors only against `plan.Root`. That means a whole-project
parent-root `fix --check` can miss child-owned files entirely when the parent has
no collection covering them, or can fix child-owned files using parent rules
when the parent has a broad collection.

The loader parses `NestedDelegate.Config` and `project plan` displays it, but
plan loading still treats `delegate.Path` as a normal project root with
`<path>/.katalyst/`. Non-default `config:` values are therefore accepted but not
honored.

## Design

### Delegate config paths

`config` is a directory, not a file. It is resolved relative to the delegate
`path`, not relative to the active root.

```yaml
nestedConfigs:
  delegates:
    - path: packages/site
      config: katalyst
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
```

This loads `packages/site/katalyst/` while treating `packages/site/` as the child
project root for paths inside the child config. A child base with `root: .`
therefore resolves to `packages/site/`, not `packages/site/katalyst/`.

Validation:

- `config` defaults to `.katalyst`.
- `config` must be relative.
- `config` must not clean to `.`, `..`, or any path beginning with `../`.
- `config` must point to a directory that exists when the delegate is active.
- Active delegate loading must use the configured directory, not hard-coded
  `.katalyst/`.

Direct discovery still walks for `.katalyst/`. A non-default config directory is
selected directly only through `--config <path-to-config-dir>`.

`--config` should accept:

- a project root containing `.katalyst/`;
- a `.katalyst/` directory;
- a `.katalyst/config.yaml` file;
- a non-default config directory such as `packages/site/katalyst/`;
- a config file inside a non-default config directory such as
  `packages/site/katalyst/config.yaml`.

For non-default config directories, the project root is the directory's parent.

### Fix authority

`fix` is a subsystem authority key for whole-project parent-root `katalyst fix`
and `katalyst fix --check` runs.

```yaml
nestedConfigs:
  delegates:
    - path: ongoing/creative-surface-area
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
        fix: file_nearest
```

Policies:

| Policy | Behavior |
|---|---|
| `root_nearest` | Parent-root `fix` behaves as it does today for that subtree. The child config does not participate. |
| `file_nearest` | Parent-root `fix` skips parent-selected files inside the delegated subtree and runs child fix over the child project's collections. |
| `compose` | Rejected for `fix` in this implementation. |

`fix --check` follows the same plan without writing. It exits 1 if either the
root-owned portion or a delegated child-owned portion would change.

Selector-taking parent-root runs remain root-namespace only. For example,
`katalyst fix notes/draft` resolves against the parent root's flat collection
namespace. Root-level selectors for child collections remain deferred to the
path-qualified selector and alias follow-on work.

Direct runs inside a child active root are ordinary root runs. They do not need
parent permission because the child is the active root.

### Why `fix: compose` is rejected

Check composition is double evaluation. Fix composition writes content. If root
and child transforms both apply to one file, Katalyst needs a separate order and
idempotence contract. Until that exists, accepting `fix: compose` would imply a
safe behavior that has not been designed.

## Examples

### CI catches dirty child-owned files

Parent config:

```yaml
nestedConfigs:
  delegates:
    - path: ongoing/blog
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
        fix: file_nearest
```

Child config defines `posts`. A dirty file under `ongoing/blog/posts/` is
reported by:

```bash
katalyst fix --check
```

even if the parent root has no `posts` collection.

### Parent does not rewrite child-owned files

Parent config has a broad collection:

```yaml
all:
  path: ongoing/blog/posts
```

With `fix: file_nearest`, a parent-root whole-project `katalyst fix` skips files
under `ongoing/blog/` for parent fixing and lets the child config own those
rewrites.

### Custom config directory

```yaml
nestedConfigs:
  delegates:
    - path: packages/site
      config: katalyst
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
```

The plan loads `packages/site/katalyst/` and reports that path in diagnostics
and `katalyst project plan`.

## Test Checklist

- A delegated child with `config: katalyst` loads `<path>/katalyst/`, not
  `<path>/.katalyst/`.
- Relative paths inside a non-default child config resolve against `<path>/`.
- `project plan` displays the non-default child config path.
- `--config <non-default-config-dir>` loads that config and uses the directory's
  parent as project root.
- `--config <non-default-config-dir>/config.yaml` behaves the same.
- Absolute delegate `config` values are rejected.
- Delegate `config` values that clean to `.` or escape with `..` are rejected.
- Active delegates with missing configured config directories fail loudly.
- Whole-project `fix --check` reports dirty child-owned files when
  `fix: file_nearest`.
- Whole-project `fix` rewrites dirty child-owned files through the child config
  when `fix: file_nearest`.
- Whole-project `fix` skips parent-root rewrites inside a child-owned subtree
  when `fix: file_nearest`.
- `fix: root_nearest` preserves current parent-root behavior.
- Direct `fix` inside a child active root behaves like an ordinary root run.
- `fix: compose` is rejected with a usage/config error.
- `--disable-nested-config` prevents delegated child fix execution.
- `--config` disables nested config discovery for `fix`, as it does for `check`.

## Documentation Updates

- `docs/content/reference/configs/nested-configs.md`: document non-default
  `config:` resolution and `fix` authority behavior.
- `docs/content/reference/cli.md`: keep `--config` wording broad enough to
  include non-default config directories.
- `product/specs/nested-config-authority-plan.md`: add implementation steps for
  this follow-on work or create a matching plan if the work grows.

## Out of Scope

- Ordered `fix: compose` write composition.
- Root-level selectors for delegated child collections.
- Root-defined aliases for delegated child collections.
- Automatically editing parent configs from `katalyst init --nested`.
