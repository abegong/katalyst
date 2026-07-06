# Spec - nested config authority

> **Status: implementing.** Resolves issue #127 by making nested `.katalyst/`
> behavior explicit. Default authority is `root_nearest`; delegation to child
> configs is opt-in and declared by the root config.

## Overview

Katalyst currently loads one `.katalyst/` project and runs the checks declared
there. Large content roots often contain semi-independent subprojects that want
their own local conventions:

- A vault-level `projects/` config owns broad shape rules.
- `projects/ongoing/creative-surface-area/` owns blog-specific post and draft
  conventions.
- Other project folders may own their own schemas, collections, and lifecycle
  rules.

The first instinct is to discover child `.katalyst/` directories and merge them
into the root run. That is too implicit. If parent and child configs can both
apply to the same file, Katalyst needs an explicit authority model so users can
predict which config governs which rules.

This spec makes the root config authoritative by default. A child `.katalyst/`
does not affect a root run unless the root config explicitly delegates authority
to it for a path and subsystem. The root owner can delegate broadly, narrowly,
or choose explicit composition where both root and child rules should run.

## Value

Nested configs unlock local ownership without giving up root governance:

- Root owners can keep global rules in one place.
- Subproject owners can keep local conventions beside the content they govern.
- `katalyst check` from the root can still be the one command in hooks and CI.
- Effective behavior is inspectable from the root config, not inferred from
  directory depth or hidden check-family rules.

The headline constraint: **no conditional implicit precedence.** If a child
config governs a subtree during a root run, the root config says so.

## Current State

Katalyst project discovery finds a `.katalyst/` config and treats that as the
active project. In the BrainPal vault example, running `katalyst check` from
`projects/ongoing/creative-surface-area` still loads the ancestor
`projects/.katalyst/` config when no local `.katalyst/` exists.

The relevant code is concentrated in four places:

| File | Contract today | Change pressure |
|---|---|---|
| `internal/project/loader.go` | `Load(start)` finds the nearest ancestor with `.katalyst/`, reads one project, resolves schemas, bases, collections, and filesystem check scopes. | This becomes the home for nested config discovery and authority parsing. |
| `internal/project/selector.go` | `Project.Resolve` expands selectors against one flat project namespace. Collection names are unique within that project. | Delegated child collections need an addressing model, or root runs must avoid selecting them by collection name in the first cut. |
| `cmd/check.go` | `katalyst check` runs filesystem checks only for no-selector root runs, then resolves collection/item selectors, then runs per-item and collection-scoped checks. | The command needs a check plan that can carry rule provenance and authority decisions across root and delegated configs. |
| `cmd/engine.go` | `newEngine` calls `loadConfigFromCWD`; `checksFor` builds checks from one project's collection and schema map. | The engine needs either an authority-resolved project view or a plan layer above the existing project engine. |
| `internal/storage/filesystemcheck/scope.go` | Filesystem check scopes are attached to one filesystem base and validated through `checks.BuildConfigured`. | `compose` and `file_nearest` need to decide which scopes survive for a delegated subtree. |

Today, a root config can enforce broad filesystem checks:

```yaml
type: filesystem
root: .
filesystemChecks:
  - path: .
    include:
      - "README.md"
      - "ongoing/**"
      - "episodic/**"
      - ".katalyst/**"
    checks:
      - kind: filesystem_unmatched_files
```

That works for top-level governance, but subproject-specific rules must also
live in the root config if they are to run from the root. Issue #127 proposes
discovering child configs; the unresolved design choice is which config has
authority when parent and child can both speak about the same content.

## Precedent

Oxlint uses a file-nearest model for source linting: each file uses the nearest
config, parent and child configs do not automatically merge, and shared rules
come through explicit `extends`. That is clean for source files because a source
file belongs to one package-like context.

Katalyst has a broader content-governance problem. A parent config may be
validating the container itself ("only these project directories belong here")
while a child config validates local content ("blog post folders have these
files"). Those two facts can both be legitimate. Katalyst should therefore make
authority explicit rather than copying either root-nearest or file-nearest
behavior unconditionally.

## Design

### Vocabulary

An **active root** is the Katalyst project selected by the command invocation.
With no explicit flag, it is the same project Katalyst would load today.

A **nested config** is a `.katalyst/` directory below the active root.

An **authority policy** says which config supplies rules for a path and
subsystem during a root run:

| Policy | Meaning |
|---|---|
| `root_nearest` | The active root config governs. Nested configs do not contribute rules for the matched scope. |
| `file_nearest` | The nearest delegated nested config governs. Root rules for the delegated subsystem do not also apply to the same units. |
| `compose` | Root and nested configs both contribute rules. This is explicit double-application. |

`root_nearest` is the default.

### Root-owned delegation

Nested behavior is declared in the active root's `.katalyst/config.yaml`.
Children cannot unilaterally change how a parent run treats them.

Proposed shape:

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

Fields:

| Key | Required | Default | Meaning |
|---|---|---|---|
| `discovery` | no | `none` | How nested configs are found: `none`, `explicit`, or `walk`. |
| `defaultAuthority` | no | `root_nearest` | Policy for discovered nested configs when no override applies. |
| `delegates` | no | `[]` | Path-scoped authority overrides. |

Delegate fields:

| Key | Required | Default | Meaning |
|---|---|---|---|
| `path` | yes | - | Subtree path relative to the active root. |
| `config` | no | `.katalyst` | Config directory relative to `path`. |
| `authority` | yes | - | Subsystem -> authority policy, or subsystem -> authority rule. |

`config` is a directory, not a file. It is resolved relative to the delegate
`path`, not relative to the active root. For the default case:

```yaml
path: ongoing/creative-surface-area
config: .katalyst
```

the child project root is `ongoing/creative-surface-area/`, and its config
directory is `ongoing/creative-surface-area/.katalyst/`.

Non-default config directory names are allowed for root-delegated runs:

```yaml
path: packages/site
config: katalyst
```

This loads `packages/site/katalyst/` while treating `packages/site/` as the
child project root for relative paths inside that config. Direct discovery still
uses `.katalyst/`; a child using a non-default config directory is only selected
directly when a command uses `--config packages/site/katalyst`.

The `config` path must be relative, must name a directory below `path`, and must
not contain `..` segments after cleaning. Invalid `config` values are config
errors, not silent fallbacks to `.katalyst/`.

A non-empty `delegates` block implies explicit discovery. A later
`discovery: walk` can scan for child `.katalyst/` directories, but the authority
rules still come from the root config. Walked children with only the default
`root_nearest` policy are discovered but inactive for check planning.

### Initializing nested projects

`katalyst init` should make nested projects deliberate. If the target directory
is inside an existing Katalyst project and does not already contain its own
`.katalyst/`, plain `katalyst init` exits with a usage error and writes
nothing:

```text
found parent Katalyst project at ../../.katalyst
refusing to create an implicit nested project
rerun with: katalyst init --nested
```

`katalyst init --nested` creates the local `.katalyst/` directory and reports
the implications:

- commands run from the child subtree now select the child as the active root;
- commands run from the parent root continue to use the parent config;
- parent-root runs ignore the child config until the parent delegates to it.

After creating the nested config, the command prints a parent-config snippet:

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

The command does not edit the parent config automatically. A future
`--update-parent` flag could add that behavior after the authority format is
stable.

If the target already contains `.katalyst/`, the existing "already initialized"
behavior applies.

### Subsystems

Authority is scoped explicitly by subsystem, and may be refined by check family
or check kind where a subsystem contains checks.

Initial subsystem keys:

| Subsystem | Meaning |
|---|---|
| `filesystemChecks` | Base-level raw filesystem scopes and their checks. |
| `collections` | Collection definitions: membership, patterns, item identity, and collection selectors. |
| `collectionChecks` | Checks attached to collections, including schemas folded through `schema:`. |
| `schemas` | Named schema resolution for delegated collection checks. |
| `fix` | Which config supplies item canonicalization for whole-project `katalyst fix` and `fix --check` runs. |

Each subsystem may be configured with a scalar policy:

```yaml
authority:
  collections: file_nearest
  filesystemChecks: compose
```

For check-bearing subsystems, the scalar form is shorthand for `default`. The
expanded form allows explicit per-family or per-kind overrides:

```yaml
authority:
  collectionChecks:
    default: file_nearest
    kinds:
      markdown_writing_tells: compose
    families:
      fileSystem: root_nearest
```

Resolution order is:

1. check kind override;
2. check family override;
3. subsystem default;
4. `nestedConfigs.defaultAuthority`;
5. implicit `root_nearest`.

No check type gets special behavior unless the root config names its kind,
family, or subsystem. That is the design point: conditional behavior is allowed,
but it is conditional because config says so.

`fix` is not check-bearing. It accepts only scalar `root_nearest` or
`file_nearest` in this implementation. `compose` for `fix` is rejected because
write composition needs a separately designed order and idempotence contract.

### Command behavior

Running from the active root:

1. Load the active root config exactly as today.
2. Load explicitly delegated nested configs.
3. Build a check plan from the root config.
4. For each delegated subtree and subsystem:
   - `root_nearest`: keep the root plan only.
   - `file_nearest`: replace the root subsystem plan for that subtree with the
     nearest delegated child plan.
   - `compose`: append the child subsystem plan beside the root plan.
   - expanded authority rules resolve per check kind/family before deciding
     which specific check instances to keep, replace, or compose.
5. Run the resulting plan and report which config produced each violation.

`katalyst fix` uses the same active-root and delegation model for whole-project
runs, but only for the `fix` subsystem:

- `fix: root_nearest` keeps today's behavior for that subtree. The parent root
  may fix files selected by parent collections, and the child config does not
  participate in a parent-root run.
- `fix: file_nearest` makes the child config responsible for fixing its own
  subtree during a parent-root whole-project run. Parent-root fix execution
  skips root-selected files under the delegated subtree; child fix execution
  runs over the child project's collections.
- `fix: compose` is a configuration error for now.

`fix --check` follows the same plan without writing. It exits 1 if either the
root-owned portion or a delegated child-owned portion would change.

Selector-taking runs remain root-namespace only in this first cut. For example,
`katalyst fix notes/draft` from the parent root resolves `notes/draft` against
the parent root's flat collection namespace. Root-level selectors for delegated
child collections are deferred with path-qualified selectors and root-defined
aliases.

Running inside a child project directly:

- If that child `.katalyst/` is the active root, its config behaves like any
  other root config. It does not need parent permission to govern its own direct
  run.
- Parent delegation matters only when the parent is the active root.

Explicit flags:

| Flag | Behavior |
|---|---|
| `--config <path>` | Load exactly that config and disable nested config discovery. |
| `--disable-nested-config` | Load the active root only, ignoring `nestedConfigs`. |
| `--project <dir>` | Select the active root explicitly. |

`--config` is the escape hatch for tools that need one stable config regardless
of the current directory. It accepts a project root, a config directory, or a
config file. When the config directory is not named `.katalyst`, its parent is
the project root. `--disable-nested-config` is the debugging escape hatch for
root configs that normally delegate.

### Diagnostics

Violations from nested configs must include the config root in diagnostics:

```text
ongoing/creative-surface-area/posts/foo/draft.md:1: markdown_requires_h1: missing H1
  config: ongoing/creative-surface-area/.katalyst
```

Katalyst exposes the resolved authority plan through `katalyst project plan`.
The command loads project configuration and prints the plan without running
checks. That keeps "which config governs this subtree?" with project
introspection instead of overloading `katalyst check` or introducing an
ambiguous dry-run mode for a command that already does not write.

```text
$ katalyst project plan

root: .katalyst
nested:
  ongoing/creative-surface-area/.katalyst
    collections: file_nearest
    collectionChecks: file_nearest
    filesystemChecks: compose
    schemas: file_nearest
```

An explicit plan view is part of the feature, not a nicety. The whole point is
to avoid invisible precedence.

## Examples

### Root owns container, child owns content

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

The root still validates broad project shape. The blog config supplies its post
collections and post checks when the root run reaches that subtree.

### Root and child both check filesystem shape

```yaml
nestedConfigs:
  discovery: explicit
  delegates:
    - path: ongoing/creative-surface-area
      authority:
        filesystemChecks: compose
```

The root can enforce "only `ongoing/**` belongs under `projects/`" while the
child enforces "post folders may contain only `README.md`, `outline.md`,
`draft.md`, `research/**`, and assets."

### Root intentionally ignores child config

```yaml
nestedConfigs:
  discovery: explicit
  defaultAuthority: root_nearest
  delegates:
    - path: ongoing/creative-surface-area
      config: .katalyst
      authority: {}
```

This records that the child config exists but does not activate it during root
runs. The plan-visibility surface should show the child as discovered and
inactive.

### Child owns fix canonicalization

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

`katalyst check` from the root validates the delegated subtree with the child
collections and checks. `katalyst fix --check` from the root also examines child
items with the child config. A dirty child file is reported even if the parent
root has no collection that names that file.

If the parent root has a broad collection that also matches child files,
`fix: file_nearest` prevents the parent from rewriting those files during a
whole-project parent run.

### Custom child config directory

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

The parent-root plan loads `packages/site/katalyst/` and resolves paths inside
that child config against `packages/site/`. Diagnostics and `project plan`
display `packages/site/katalyst`, not `packages/site/.katalyst`.

## Rejected Alternatives

### Auto-merge every discovered child config

Rejected. It creates surprising double-application and makes the effective
rules for a file depend on implicit ancestry. Users would have to inspect every
ancestor to know what happened.

### File-nearest for all checks

Rejected as the universal default. It is attractive for collection and item
checks, but it weakens root-level governance over container shape. A parent may
legitimately validate the tree that contains a child project.

### Check-family-specific implicit behavior

Rejected. "Filesystem checks compose, item checks nearest-win" sounds reasonable
until users have to debug it. If the behavior is conditional, the condition
belongs in config.

### Child opt-out

Rejected for root runs. A child project may define local rules, but it should
not be able to unilaterally stop the parent from governing a parent-initiated
run. Delegation is owned by the active root.

### Compose fix writes now

Rejected for this implementation. Check composition is just double evaluation,
but fix composition writes content. If root and child transforms both apply to
the same file, Katalyst needs an explicit order and an idempotence guarantee.
Until that is designed, `fix: compose` is a usage/config error.

## Test Checklist

- Root with no `nestedConfigs` behaves exactly as today.
- Root with `discovery: explicit` and no delegating authority loads but does not
  run child checks.
- `file_nearest` collection authority runs child collection checks for child
  items and suppresses root collection checks for the delegated subsystem.
- Per-kind authority overrides the subsystem default for that check type.
- Per-family authority overrides the subsystem default and is overridden by a
  per-kind rule.
- `compose` filesystem authority runs both root and child filesystem checks and
  reports both configs in diagnostics.
- `root_nearest` default ignores child checks even when a child `.katalyst/`
  exists.
- Direct run inside a child project treats the child as active root.
- Plain `katalyst init` inside an existing project exits with a usage error and
  writes no files.
- `katalyst init --nested` inside an existing project creates the child config
  and prints the parent delegation snippet.
- `--config` disables nested discovery.
- `--config` can load a non-`.katalyst` config directory and use its parent as
  the project root.
- `--disable-nested-config` ignores configured delegates.
- Diagnostics identify the config root that produced each violation.
- The plan-visibility surface prints root, child, subsystem, and authority
  decisions.
- A delegated child with `config: katalyst` is loaded from
  `<path>/katalyst/`, not `<path>/.katalyst/`.
- Invalid delegate `config` values that are absolute or escape the delegate
  path are rejected.
- Whole-project `fix --check` reports dirty child-owned files when
  `fix: file_nearest`.
- Whole-project `fix` skips parent-root rewrites inside a child-owned subtree
  when `fix: file_nearest`.
- Direct `fix` inside a child active root behaves like an ordinary root run.
- `fix: compose` is rejected until ordered write composition is specified.

## Documentation Updates

- `docs/content/reference/configs/nested-configs.md`: dedicated reference for
  `nestedConfigs`, authority policies, nested project initialization, diagrams,
  path resolution, and examples.
- `docs/content/reference/configs/_index.md`: link to nested config reference
  page from the config concept list.
- `docs/content/deep-dives/domain-model/base.md`: explain root authority,
  nested config delegation, and why filesystem/container checks may compose.
- `docs/content/reference/glossary.md`: add active root, nested config,
  authority policy, root-nearest, and file-nearest.
- `docs/content/how-to/configure-rules.md`: add a "large vault / monorepo"
  example.
- `docs/content/reference/cli.md`: document `--config`,
  `--disable-nested-config`, `--project`, and the plan explanation surface.
- `AGENTS.md`: add conventions for config authority and where nested-config
  logic lives.

## Out of Scope

- Implementing `extends` or reusable rule bundles.
- Root-level selectors for delegated child collections. Follow-on work should
  add path-qualified selectors and root-defined aliases.
- A general collection identity model for delegated child configs. Follow-on
  work should decide how collection name collisions are represented outside the
  check plan.
- Changing check type descriptors.
- Changing existing base, collection, schema, or check config formats outside
  the new `nestedConfigs` section.
- Making child configs automatically active during parent runs.
- Automatically editing the parent config from `katalyst init --nested`.
- CI or hook changes outside Katalyst's command behavior.

## Appendix: deferred selectors and collection identity

Nested config authority can ship before root-level selectors for delegated
collections. This appendix preserves the design context for the follow-on work.

### Selector grammar for delegated collections

`internal/project/selector.go` resolves selectors in one flat namespace:
`<collection>` and `<collection>/<item>`. A root run with delegated child configs
may have several `posts` collections, one per child. Diagnostics can include the
config path, but selectors need an address if root users target child
collections directly.

| Choice | What it buys | Cost |
|---|---|---|
| Defer nested collection selectors | Small first cut. Root `katalyst check` still validates delegated content wholesale. | Users cannot run `katalyst check` against one child collection from the root. |
| Path-qualified selectors, e.g. `ongoing/creative-surface-area:posts` | No new alias config. The path is unambiguous. | Long selectors. The colon form is new syntax. |
| Root-defined aliases, e.g. `csa:posts` | Short stable selectors. Aliases survive directory moves. | Adds another naming surface to `nestedConfigs`. |

Decision for this spec: defer nested collection selectors. Add
path-qualified selectors and root-defined aliases as follow-on work.

### Collection name collisions in the resolved plan

`project.Config.Collections` is a flattened list with project-wide unique names.
Delegated child configs break that assumption if two children define `posts`.
The check plan can hold provenance separately, but any API that exposes a single
`Collection(name)` lookup needs a collision policy.

| Choice | What it buys | Cost |
|---|---|---|
| Keep delegated collections out of `Config.Collections` | Preserves today's `Collection(name)` contract for root projects. | The check plan needs a separate structure for delegated collections. |
| Prefix child collection names internally | Reuses existing flat resolution machinery. | Leaks generated names into diagnostics and user APIs unless carefully hidden. |
| Replace flat collection lookup with `(configRoot, collectionName)` | Correct model for nested configs. | Larger API change across `cmd`, `internal/project`, inspectors, and item commands. |

Decision for this spec: keep delegated collections out of the root
`Config.Collections` flattening. Build a separate authority plan for `check`,
then revisit the project API when `collection`, `item`, and `inspect` need
nested selectors.
