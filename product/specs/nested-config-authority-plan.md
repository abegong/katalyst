# Plan - nested config authority
> Spec: [Nested config authority](./nested-config-authority-spec.md)
> **Status: planning.**

## Current State

Katalyst loads one active project and executes checks from that project only.
Nested `.katalyst/` directories change direct command discovery today, but they
have no explicit parent-run semantics.

| File | Current behavior | Plan pressure |
|---|---|---|
| `internal/project/loader.go` | `Load(start)` finds the nearest ancestor `.katalyst/`, reads one root, and flattens bases and collections into one `Config`. | Split root discovery from exact-root loading, parse `nestedConfigs`, and load delegated child configs without flattening them into the root config. |
| `internal/project/project.go` | `Project` wraps one `Config` and exposes collections, filesystem scopes, selector resolution, item IO, and unmatched-file scans. | Keep direct project behavior stable while adding a resolved plan model that can reference several loaded configs. |
| `internal/project/selector.go` | Selectors address the flat project namespace: `<collection>` and `<collection>/<item>`. | Preserve this for root selectors. Defer path-qualified delegated selectors. |
| `internal/storage/filesystemcheck/scope.go` | Filesystem scopes are built inside one filesystem base and expanded without config provenance. | Preserve scope parsing, then wrap scopes with provenance and authority during planning. |
| `cmd/init.go` | `init` refuses to overwrite a target that already contains `.katalyst/`, then writes the default config and base. | Detect an ancestor project and require `--nested` before creating a child project. |
| `cmd/check.go` | `check` builds one engine, resolves selectors against one project, runs filesystem checks only for no-selector runs, then runs item and collection checks. | Run a resolved check plan with root and delegated config provenance. |
| `cmd/filesystem_check.go` | Filesystem diagnostics show scope name and path. | Include the config root for violations from nested scopes. |
| `cmd/engine.go` | One engine owns one project config and schema cache. | Keep the engine per config, then execute multiple engines through a command-level plan. |
| `cmd/root.go` | Top-level help has verbs and resource nouns. There is no `project` command. | Add a singleton `project` resource with `plan` as its first subcommand. |
| `cmd/cli_style_test.go` | Every resource noun must expose `list`. | Add a documented singleton-resource exception for `project`. |

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Failing tests | Add behavior tests and snapshots for config parsing, nested init, project plan output, and check authority. |
| 2 | Project loader model | Add exact-root loading, `nestedConfigs` parsing, authority validation, and plan data structures. |
| 3 | Nested init | Implement `katalyst init --nested` and ancestor-project refusal. |
| 4 | Project plan command | Add `katalyst project plan`, command-style exceptions, and plan rendering. |
| 5 | Check execution | Execute the resolved plan for root, delegated, composed, and nearest-child checks. |
| 6 | Documentation and verification | Graduate the planned user docs, CLI reference, glossary, and code conventions. |

The ordering keeps user-facing behavior pinned before internals change, then
ships the read-only plan surface before changing `check`.

## Phases

### Phase 1

**Goal.** Pin the requested behavior with failing tests and snapshots.

1. Add loader tests for `nestedConfigs`.

   **File:** `internal/project/loader_test.go`

   Add tests for default `root_nearest`, `discovery: explicit`, `discovery:
   walk`, non-empty `delegates` implying explicit discovery, invalid discovery
   values, invalid authority policies, invalid subsystem names, and per-kind /
   per-family authority precedence.

2. Add nested project fixture helpers only where duplication appears.

   **File:** `internal/project/projecttest/projecttest.go`

   Add helpers for parent-plus-child `.katalyst/` fixtures if the tests repeat
   more than the existing `WriteProject` helper should absorb.

3. Add init tests for nested project creation.

   **File:** `cmd/init_test.go`

   Add failing tests that plain `katalyst init --dir child` inside an existing
   project exits with code 2 and writes nothing, and that `katalyst init --nested
   --dir child` creates the child config and prints the parent delegation
   snippet.

4. Add project-plan command tests.

   **File:** `cmd/project_test.go` (new)

   Add tests for `katalyst project plan` with no nested config, an explicit
   delegated child, an inactive root-nearest child, and `--disable-nested-config`.

5. Add project-plan snapshots.

   **File:** `cmd/testdata/snapshots/project/plan.txt` (new)

   Pin the human-readable authority plan layout. Normalize temporary roots with
   `normTmp`.

6. Add top-level help snapshots for `project`.

   **File:** `cmd/testdata/snapshots/help/project.txt` (new)

   Pin `katalyst project --help` after the command is added.

7. Add check authority tests.

   **File:** `cmd/check_test.go`

   Add failing tests for unchanged root behavior with no `nestedConfigs`, root
   discovery with inactive child configs, `file_nearest` collection authority,
   `compose` filesystem authority, per-kind overrides, per-family overrides, and
   config-root diagnostics.

8. Add command-style tests for the singleton project resource.

   **File:** `cmd/cli_style_test.go`

   Update root help order expectations and add an explicit exception that the
   singleton `project` resource does not need a `list` subcommand.

### Phase 2

**Goal.** Load nested config authority without changing command behavior.

1. Split nearest-root discovery from exact-root loading.

   **File:** `internal/project/loader.go`

   Add `FindRoot(start string) (string, error)` for nearest-root discovery and
   `LoadRoot(root string) (*Config, error)` for exact-root loading. Keep
   `Load(start)` as the compatibility wrapper: find the root, then load it.

2. Add parent-root discovery for init.

   **File:** `internal/project/loader.go`

   Add `FindParentRoot(target string) (string, error)` or an equivalent helper
   that starts at the target parent and returns `ErrNotFound` when no ancestor
   project exists.

3. Parse `nestedConfigs`.

   **File:** `internal/project/loader.go`

   Extend `rawConfig` with `NestedConfigs`. Parse it into `Config` after the
   existing schema and base settings are read, so config format errors still
   point at `.katalyst/config.yaml`.

4. Define authority types and validators.

   **File:** `internal/project/nested.go` (new)

   Add `NestedConfigSettings`, `NestedConfigDiscovery`, `NestedDelegate`,
   `AuthorityPolicy`, `AuthorityRule`, and `AuthoritySubsystem`. Normalize
   defaults, validate enum values, sort delegates by path, and reject duplicate
   delegate paths.

5. Add authority resolution helpers.

   **File:** `internal/project/authority.go` (new)

   Add helpers that resolve a policy for a subsystem, then for check-bearing
   subsystems by kind, family, default, `nestedConfigs.defaultAuthority`, and
   implicit `root_nearest`.

6. Keep flat project APIs stable.

   **File:** `internal/project/project.go`

   Keep `Project.Collections`, `Project.Collection`, and `Project.Resolve`
   rooted in one config. Do not expose delegated child collections through the
   root project's flattened collection list.

7. Update project package conventions.

   **File:** `internal/project/AGENTS.md`

   Document that the loader owns `nestedConfigs` parsing and that delegated
   child configs stay outside the root `Config.Collections` flattening.

### Phase 3

**Goal.** Make nested project initialization explicit.

1. Add the `--nested` flag.

   **File:** `cmd/init.go`

   Add `--nested` as a boolean flag. When the target has no local `.katalyst/`
   but has an ancestor project, plain `init` returns a usage error and writes no
   files.

2. Print the nested-init implication and parent snippet.

   **File:** `cmd/init.go`

   After `init --nested` creates the child config, print the direct-run and
   parent-run implications plus a `nestedConfigs.delegates` snippet using the
   child path relative to the parent root.

3. Preserve existing init behavior outside parent projects.

   **File:** `cmd/init.go`

   Keep the current fresh-project scaffold unchanged when no ancestor project
   exists. Keep the existing "already exists" refusal when the target already
   has `.katalyst/`.

4. Update init help snapshots.

   **File:** `cmd/testdata/snapshots/help/init.txt`

   Regenerate the snapshot so `--nested` appears in help.

### Phase 4

**Goal.** Expose the resolved authority plan without running checks.

1. Add a resolved project-plan model.

   **File:** `internal/project/plan.go` (new)

   Add `Plan`, `PlanOptions`, `PlanRoot`, `PlanDelegate`, and
   `PlanAuthority` types. `BuildPlan` loads the active root, loads explicit
   delegates, marks root-nearest delegates inactive, and records config roots
   and subsystem policies.

2. Respect explicit config-selection options.

   **File:** `internal/project/plan.go` (new)

   Add `PlanOptions.ConfigPath`, `PlanOptions.ProjectDir`, and
   `PlanOptions.DisableNestedConfig`. `ConfigPath` loads exactly that config and
   disables nested discovery. `ProjectDir` selects the active root. The disable
   option ignores `nestedConfigs`.

3. Add the singleton `project` command.

   **File:** `cmd/project.go` (new)

   Add `newProjectCmd` and `newProjectPlanCmd`. The parent command has no
   default action. The `plan` subcommand runs with no args and prints the
   resolved authority plan.

4. Register `project` in root help.

   **File:** `cmd/root.go`

   Add `project` to the resources group before `collection`, because it
   describes the loaded project as a whole.

5. Update command grammar conventions.

   **File:** `cmd/AGENTS.md`

   Document `project` as a singleton resource noun. It uses subcommands but does
   not expose `list`, because there is one active project per command
   invocation.

6. Update CLI style tests for the singleton resource.

   **File:** `cmd/cli_style_test.go`

   Update root command order and add a named exception for `project` in the
   list-subcommand assertion.

7. Render the authority plan.

   **File:** `cmd/project_plan.go` (new)

   Print root path, nested config path, active or inactive status, and resolved
   authority for each subsystem. Keep the output deterministic by sorting
   delegates and subsystem rows.

8. Update help snapshots.

   **File:** `cmd/testdata/snapshots/help/root.txt`

   Regenerate the root help snapshot so `project` appears in Resources.

### Phase 5

**Goal.** Run checks from the resolved authority plan.

1. Add executable check-plan types.

   **File:** `cmd/check_plan.go` (new)

   Add command-local types for planned projects, planned collections, planned
   filesystem scopes, and config provenance. Keep this layer in `cmd` because
   it describes execution, not raw config loading.

2. Build a root-only plan that preserves today's behavior.

   **File:** `cmd/check_plan.go` (new)

   For projects without `nestedConfigs`, build one planned project from the
   active root. Selectors, unmatched-file scans, filesystem checks, and
   collection checks must match current output.

3. Add delegated collection execution.

   **File:** `cmd/check_plan.go` (new)

   For `collections`, `collectionChecks`, and `schemas` under `file_nearest`,
   route files inside the delegated subtree to the nearest child project. Keep
   root selectors flat for this first cut.

4. Add filesystem authority execution.

   **File:** `cmd/check_plan.go` (new)

   For `filesystemChecks`, keep root scopes for `root_nearest`, replace
   delegated subtree scopes for `file_nearest`, and append child scopes for
   `compose`.

5. Apply per-kind and per-family authority.

   **File:** `cmd/check_plan.go` (new)

   Split configured checks by subsystem, check family, and check kind before
   composing or replacing check instances. Use the authority resolver from the
   `project` package.

6. Run the check plan.

   **File:** `cmd/check.go`

   Replace the single-project execution path with check-plan execution. Preserve
   the existing selector behavior for the active root and no-selector whole
   project runs.

7. Carry one engine per config.

   **File:** `cmd/engine.go`

   Add a constructor that accepts a loaded `project.Config`. Keep schema caches
   scoped per config so child schemas resolve against the child root.

8. Add config provenance to item diagnostics.

   **File:** `cmd/check.go`

   Print the config root after violations produced by a nested config. Keep
   root-only diagnostics unchanged unless a provenance line is needed.

9. Add config provenance to filesystem diagnostics.

   **File:** `cmd/filesystem_check.go`

   Carry the planned config root into filesystem scope execution and print it
   for nested-scope violations.

10. Wire config-selection flags.

    **File:** `cmd/check.go`

    Add `--config`, `--project`, and `--disable-nested-config` to `check`.
    Route them into the plan builder. `--config` disables nested discovery.

11. Mirror config-selection flags for `fix --check` where needed.

    **File:** `cmd/fix.go`

    Add the same config-selection flags only for behavior that the spec covers:
    project selection and nested disabling. Keep delegated write behavior out of
    scope unless `fix` execution needs the plan for `fix --check`.

### Phase 6

**Goal.** Make the implemented behavior discoverable and verify the branch.

1. Update nested config reference.

   **File:** `docs/content/reference/configs/nested-configs.md`

   Replace draft wording with implemented behavior. Document `katalyst project
   plan`, `init --nested`, authority policies, and config-selection flags.

2. Update CLI reference.

   **File:** `docs/content/reference/cli.md`

   Add `project` to the command grammar and document the shared `--config`,
   `--project`, and `--disable-nested-config` flags if they are shared across
   validating commands.

3. Update glossary vocabulary.

   **File:** `docs/content/reference/glossary.md`

   Add active root, nested config, authority policy, root-nearest,
   file-nearest, and composed authority.

4. Update base/domain-model rationale.

   **File:** `docs/content/deep-dives/domain-model/base.md`

   Explain why root filesystem/container checks may compose with child content
   checks and why delegation is root-owned.

5. Add a large-project how-to.

   **File:** `docs/content/how-to/configure-rules.md`

   Add a short vault or monorepo example that initializes a child project and
   delegates collection authority from the parent.

6. Update root code conventions.

   **File:** `AGENTS.md`

   Add the permanent convention that nested-config loading and authority parsing
   live in `internal/project`, while command execution planning stays in `cmd`.

7. Regenerate snapshots.

   **File:** `cmd/testdata/snapshots/help/*.txt`

   Run the snapshot update path for changed help surfaces and review the diff.

8. Verify.

   **File:** `Makefile`

   Run `make all`, then `./bin/katalyst check`. Use temporary Go caches in
   sandboxed environments.

## Key Files

| File | Role |
|---|---|
| `product/specs/nested-config-authority-spec.md` | Source spec and authority decisions. |
| `product/specs/nested-config-authority-plan.md` | This implementation plan. |
| `internal/project/loader.go` | Root discovery, exact-root loading, raw config parsing, and backward-compatible `Load`. |
| `internal/project/nested.go` | Nested config settings, delegates, authority policies, and validation. |
| `internal/project/authority.go` | Subsystem, family, and kind authority resolution. |
| `internal/project/plan.go` | Resolved project plan for root and delegated configs. |
| `internal/project/project.go` | Stable single-config project API. |
| `internal/project/selector.go` | Existing flat selector grammar, unchanged in this first cut. |
| `internal/project/loader_test.go` | Loader and authority parsing tests. |
| `internal/project/projecttest/projecttest.go` | Shared fixture helpers for parent and child projects if needed. |
| `internal/project/AGENTS.md` | Project package ownership rules. |
| `cmd/init.go` | Nested initialization behavior. |
| `cmd/init_test.go` | Init behavior tests. |
| `cmd/project.go` | Singleton project resource command. |
| `cmd/project_plan.go` | Human-readable plan rendering. |
| `cmd/project_test.go` | Project plan command behavior tests. |
| `cmd/root.go` | Command registration and root help order. |
| `cmd/cli_style_test.go` | Command grammar enforcement and singleton project exception. |
| `cmd/AGENTS.md` | CLI command grammar and singleton resource convention. |
| `cmd/check_plan.go` | Executable check plan assembled from resolved project authority. |
| `cmd/check.go` | Check command execution and diagnostics. |
| `cmd/filesystem_check.go` | Filesystem scope execution and config provenance diagnostics. |
| `cmd/engine.go` | Per-config check engine and schema cache. |
| `cmd/fix.go` | Config-selection flag parity for fix behavior covered by the spec. |
| `cmd/testdata/snapshots/help/root.txt` | Root help snapshot. |
| `cmd/testdata/snapshots/help/project.txt` | Project help snapshot. |
| `cmd/testdata/snapshots/help/init.txt` | Init help snapshot after `--nested`. |
| `cmd/testdata/snapshots/project/plan.txt` | Project plan output snapshot. |
| `docs/content/reference/configs/nested-configs.md` | User reference for nested configs. |
| `docs/content/reference/configs/_index.md` | Link to nested config reference. |
| `docs/content/reference/cli.md` | CLI grammar and shared flag reference. |
| `docs/content/reference/glossary.md` | New vocabulary. |
| `docs/content/deep-dives/domain-model/base.md` | Rationale for root-owned delegation and composed filesystem checks. |
| `docs/content/how-to/configure-rules.md` | Large-project setup example. |
| `AGENTS.md` | Permanent cross-package conventions after graduation. |
| `Makefile` | Verification entry point. |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Default authority | `root_nearest` | Parent projects keep governance unless they delegate authority explicitly. |
| Delegation owner | Active root config | Child configs do not unilaterally change parent-run behavior. |
| Plan visibility command | `katalyst project plan` | Authority inspection belongs to project introspection, not `check` or a read-only dry-run flag. |
| `project` command shape | Singleton resource noun | A command invocation has one active project. `project plan` reads that singleton, so a `list` subcommand has no meaningful output. |
| Exact-root loading | Add `LoadRoot` beside `Load` | Parent plans must load child configs exactly, not rediscover the parent through nearest-root walking. |
| Child collection identity | Keep delegated collections out of root `Config.Collections` | The first cut preserves the flat root selector contract and avoids generated collection names. |
| Executable plan location | Build execution plan in `cmd` | The project package owns config and authority. The command layer owns check execution, engines, selectors, and diagnostics. |
| Schema cache scope | One engine per config | Child schemas resolve relative to child config roots and should not share path/name assumptions with the parent. |
| Diagnostics provenance | Add config-root context at command output time | Check implementations stay unaware of nested config authority. |

## Documentation Updates

| Phase | File | Update |
|---|---|---|
| 3 | `cmd/testdata/snapshots/help/init.txt` | Show `--nested`. |
| 4 | `cmd/AGENTS.md` | Document the singleton `project` resource exception. |
| 4 | `cmd/testdata/snapshots/help/root.txt` | Show `project` in Resources. |
| 4 | `cmd/testdata/snapshots/help/project.txt` | Add top-level help for `project`. |
| 6 | `docs/content/reference/configs/nested-configs.md` | Document implemented `nestedConfigs`, init implications, diagrams, and `project plan`. |
| 6 | `docs/content/reference/cli.md` | Add `project` and config-selection flags. |
| 6 | `docs/content/reference/glossary.md` | Add active root, nested config, authority policy, root-nearest, file-nearest, and composed authority. |
| 6 | `docs/content/deep-dives/domain-model/base.md` | Explain root-owned authority and composed filesystem/container checks. |
| 6 | `docs/content/how-to/configure-rules.md` | Add the large-project nested config example. |
| 6 | `AGENTS.md` | Record package ownership for nested authority logic. |

## Out of Scope

- Implementing `extends` or reusable rule bundles.
- Path-qualified selectors for delegated child collections.
- Root-defined aliases for delegated child collections.
- A general `(configRoot, collectionName)` collection identity model outside
  the check plan.
- Automatically editing the parent config from `katalyst init --nested`.
- Making child configs active during parent runs without root delegation.
- Changing check type descriptors.
- Changing existing base, collection, schema, or check config formats outside
  `nestedConfigs`.
- CI or hook changes outside Katalyst command behavior.
