+++
title = "Nested configs"
weight = 75
+++

# Nested Configs

Nested configs let a root Katalyst project delegate authority to config
directories inside its tree. The default child config directory is `.katalyst/`,
but an explicit delegate can name a different directory.

They are inactive unless the root config delegates to them. A child config can
define local rules, but it does not affect a root run until the root config
names the child path and the subsystems it delegates.

```yaml
# .katalyst/config.yaml
nestedConfigs:
  discovery: explicit
  defaultAuthority: root_nearest
  delegates:
    - path: ongoing/creative-surface-area
      config: .katalyst
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        filesystemChecks: compose
        schemas: file_nearest
```

## Keys

| Key | Required | Default | Meaning |
|---|---|---|---|
| `discovery` | no | `none` | How nested configs are found: `none`, `explicit`, or `walk`. |
| `defaultAuthority` | no | `root_nearest` | Authority policy for discovered nested configs when no subsystem override applies. |
| `delegates` | no | `[]` | Path-scoped nested config delegations. |

`discovery: explicit` loads only the children listed under `delegates`.
`discovery: walk` scans the tree for child `.katalyst/` directories. A walked
child still follows `defaultAuthority`, so with the default `root_nearest` it is
discovered but inactive.

## Creating A Nested Project

A nested project is a `.katalyst/` directory inside another Katalyst project.
It changes config discovery for commands run from that subtree.

Before a nested project exists:

```text
projects/
  .katalyst/                  active root from projects/ and child paths
  ongoing/
    creative-surface-area/
      posts/
```

After `katalyst init --nested`:

```text
projects/
  .katalyst/                  active root from projects/
  ongoing/
    creative-surface-area/
      .katalyst/              active root from this subtree
      posts/
```

The child config becomes the active root for direct commands inside
`ongoing/creative-surface-area/`. Parent-root commands still use
`projects/.katalyst/`.

```text
direct child run:

ongoing/creative-surface-area/.katalyst
  runs as the active root

parent root run:

projects/.katalyst
  uses root rules unless nestedConfigs delegates authority
```

Plain `katalyst init` inside an existing project should refuse to create an
implicit nested project:

```text
$ katalyst init
found parent Katalyst project at ../../.katalyst
refusing to create an implicit nested project
rerun with: katalyst init --nested
```

`katalyst init --nested` creates the local `.katalyst/` directory and prints a
snippet for the parent config:

```yaml
nestedConfigs:
  discovery: explicit
  delegates:
    - path: ongoing/creative-surface-area
      config: .katalyst
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
```

The command does not edit the parent config. The parent root remains
authoritative until its config delegates authority to the child.

## Delegate Keys

| Key | Required | Default | Meaning |
|---|---|---|---|
| `path` | yes | - | Subtree path relative to the active root. |
| `config` | no | `.katalyst` | Config directory relative to `path`. |
| `authority` | yes | - | Subsystem authority policies for this nested config. |

`config` is a directory, not a file. It resolves relative to the delegate
`path`, while paths inside the child config still resolve relative to the child
project root.

```yaml
nestedConfigs:
  delegates:
    - path: packages/site
      config: katalyst
      authority:
        collections: file_nearest
```

This loads `packages/site/katalyst/` as the config directory and treats
`packages/site/` as the child project root. A child base with `root: .` points at
`packages/site/`, not `packages/site/katalyst/`.

Direct discovery still walks for `.katalyst/`. To run a non-default config
directory directly, pass it with `--config`.

The root config owns delegation. A nested config cannot opt out of root
authority during a root run, and it cannot make itself active without a root
delegate.

## Authority Policies

An authority policy decides which config supplies rules for a delegated subtree.

| Policy | Meaning |
|---|---|
| `root_nearest` | The active root config governs. The nested config contributes no rules for the matched subsystem. |
| `file_nearest` | The nearest delegated nested config governs. Root rules for the delegated subsystem do not also apply to the same units. |
| `compose` | Root and nested configs both contribute rules. This is explicit double-application. |

`root_nearest` is the default. Use `file_nearest` when the child owns the local
content model. Use `compose` when the root and child both need to validate the
same subtree, such as broad container rules plus local folder rules.

`fix` does not support `compose`. Fix composition would mean two configs can
rewrite the same file, which needs an ordering and idempotence contract. Use
`root_nearest` or `file_nearest` for `fix`.

## Subsystems

Authority is scoped by subsystem.

| Subsystem | Meaning |
|---|---|
| `filesystemChecks` | Base-level raw filesystem scopes and their checks. |
| `collections` | Collection definitions: membership, patterns, item identity, and collection selectors. |
| `collectionChecks` | Checks attached to collections, including schemas folded through `schema:`. |
| `schemas` | Named schema resolution for delegated collection checks. |
| `fix` | Canonicalization rules used by `katalyst fix` and `fix --check`. |

Each subsystem accepts a scalar policy:

```yaml
authority:
  collections: file_nearest
  filesystemChecks: compose
```

For check-bearing subsystems, the scalar form is shorthand for `default`.
Use the expanded form to override by check family or check kind:

```yaml
authority:
  collectionChecks:
    default: file_nearest
    families:
      fileSystem: root_nearest
    kinds:
      markdown_writing_tells: compose
```

Resolution order is:

1. check kind override;
2. check family override;
3. subsystem default;
4. `nestedConfigs.defaultAuthority`;
5. implicit `root_nearest`.

No check type gets special behavior unless the root config names its kind,
family, or subsystem.

## Command Behavior

When you run `katalyst check` from the active root, Katalyst loads the root
config, loads the delegated child configs, resolves the authority plan, and runs
the checks in that plan for whole-project runs. Root-level selectors still use
the active root's flat collection namespace.

When you run `katalyst fix` or `katalyst fix --check` from the active root with
no selectors, `fix: file_nearest` skips root rewrites inside that delegated
subtree and runs fix against the child project's collections. With
`fix: root_nearest`, root behavior is preserved and the child config does not
participate in the parent-root fix run.

Selector-taking parent-root fix runs stay in the root namespace. For example,
`katalyst fix notes/draft` resolves `notes/draft` against the active root's
collections.

When you run `katalyst check` inside a child project and that child is the active
root, the child config behaves like any other root config. Parent delegation only
matters when the parent is the active root.

Run `katalyst project plan` to inspect the resolved authority plan without
running checks.

Use these flags to make config selection explicit:

| Flag | Behavior |
|---|---|
| `--config <path>` | Load exactly that project root, config directory, or `config.yaml` file and disable nested config discovery. |
| `--disable-nested-config` | Load the active root only, ignoring `nestedConfigs`. |
| `--project <dir>` | Select the active root explicitly. |

## Examples

### Root Owns The Container, Child Owns Content

```yaml
nestedConfigs:
  discovery: explicit
  delegates:
    - path: ongoing/creative-surface-area
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
```

The root keeps its broad project shape rules. The child supplies its post
collections and post checks.

### Root And Child Both Check Filesystem Shape

```yaml
nestedConfigs:
  discovery: explicit
  delegates:
    - path: ongoing/creative-surface-area
      authority:
        filesystemChecks: compose
```

The root can enforce allowed top-level project paths. The child can enforce
allowed files inside each post folder.

### Root Records A Child But Does Not Activate It

```yaml
nestedConfigs:
  discovery: explicit
  defaultAuthority: root_nearest
  delegates:
    - path: ongoing/creative-surface-area
      config: .katalyst
      authority: {}
```

The child config is known to the root but contributes no rules to root runs.

### Child Owns Fixes

```yaml
nestedConfigs:
  discovery: explicit
  delegates:
    - path: ongoing/creative-surface-area
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
        fix: file_nearest
```

A root hook can run `katalyst fix --check` once and still catch dirty files
owned by the child config. A root `katalyst fix` run lets the child rewrite
those files instead of applying root collection rules to them.

### Child Uses A Custom Config Directory

```yaml
nestedConfigs:
  discovery: explicit
  delegates:
    - path: packages/site
      config: katalyst
      authority:
        collections: file_nearest
        collectionChecks: file_nearest
        schemas: file_nearest
```

The child config lives at `packages/site/katalyst/`. Commands run directly
against it can use `katalyst check --config packages/site/katalyst`.

## See Also

- [Discovery]({{< relref "discovery.md" >}}), root config discovery and config file layout.
- [Bases]({{< relref "bases.md" >}}), filesystem and SQLite base configuration.
- [Checks]({{< relref "checks.md" >}}), check attachment and object-schema precedence.
