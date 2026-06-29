# Plan - filesystem checks and collection checks
> Spec: [Filesystem checks and collection checks](./filesystem-checks-spec.md)
> **Status: planning.**

## Current State

- `cmd/check.go` runs collection-attached checks only. With no selectors it
  resolves every collection through `project.Resolve`, runs per-item checks,
  scans wholesale-selected collections for unmatched files, then runs
  collection-scoped checks.
- `cmd/engine.go` builds checks from a `project.Collection`. It owns schema
  compilation, library availability checks, variant routing, per-item builders,
  and collection-scoped builders.
- `internal/project/loader.go` loads `.katalyst/bases/<name>.yaml` files into
  `BaseInstance` values. Legacy `.katalyst/storage/` remains readable, but the
  current code and tests use bases.
- `internal/storage/collection/parse.go` parses collection config. It owns
  `RawCheck`, folds `schema:` into an `object` check, parses check args through
  `checks.Parse`, and rejects SQLite collections that configure file-system
  family checks.
- `internal/checks/registry.go` records each check type's `Descriptor`, parser,
  per-item builder, and collection-scoped builder. `Descriptor.Scope` names
  collection-scoped runtime behavior. It has no attachment-configurableIn metadata.
- `internal/checks/checks.go` defines per-item `Context` and `Check`.
  `internal/checks/collection.go` defines `CollectionContext` and
  `CollectionCheck`. Those names mix runtime granularity with the future
  product term CollectionCheck.
- `internal/storage/collection/filesystem/collection.go` already contains the
  filesystem traversal pieces needed by filesystem scopes: doublestar matching,
  sorted item discovery, and unmatched-file walking.
- `cmd/check_types.go` and `cmd/gendocs/main.go` render check descriptors for
  CLI and generated docs. They currently render family, scope, severity, fields,
  and config examples.
- `cmd/check_test.go`, `internal/project/loader_test.go`,
  `internal/checks/registry_test.go`, and check-family tests are the main test
  homes for this change.

## Sequencing

| Phase | Focus | Scope |
|---|---|---|
| 1 | Failing contracts | loader tests, check CLI tests, registry tests, snapshots |
| 2 | Shared check metadata and config parsing | descriptor configuration sites, document needs, reusable raw check parsing |
| 3 | Filesystem scope config and expansion | base-level `filesystemChecks`, include/exclude matching, unmatched set |
| 4 | File and file-set runtime contexts | shared per-file context, set-level interface, collection compatibility |
| 5 | Filesystem check execution | no-selector execution, lazy parsing, parse-failure severity, diagnostics |
| 6 | Shared check types | configurableIn metadata, document-aware file-system checks, `filesystem_unmatched_files` |
| 7 | Documentation and verification | user docs, generated reference, developer docs, focused test suite |

The order keeps the suite honest. First pin the behavior, then add registry and
config shape, then build the filesystem runner and opt check types into it.

## Phases

### Phase 1 - Failing contracts

**Goal:** tests describe filesystem-attached checks before production code
exists.

1. **File:** `internal/project/loader_test.go`.
   Add load tests for `filesystemChecks` under a filesystem base:
   optional `name`, required `include`, default `path: .`, default
   `parseFailures: error`, explicit `parseFailures: warning`, and parsed
   nested `checks`.
2. **File:** `internal/project/loader_test.go`.
   Add rejection tests for missing `include`, unknown `parseFailures`, unknown
   check kind, a check kind that cannot be configured in `filesystem`, and
   `filesystemChecks` on a SQLite base.
3. **File:** `cmd/check_test.go`.
   Add a no-selector CLI test where a project has no collections, a filesystem
   scope includes `**/*.md`, and `filesystem_name_case` reports a bad Markdown
   filename.
4. **File:** `cmd/check_test.go`.
   Add a selector test proving `katalyst check notes` runs collection checks
   only and does not run unrelated filesystem scopes.
5. **File:** `cmd/check_test.go`.
   Add parse-failure tests for `filesystem_name_matches_field`: default
   `parseFailures: error` exits 1, while `parseFailures: warning` reports a
   warning and does not fail by itself.
6. **File:** `cmd/check_test.go`.
   Add a CLI test for `filesystem_unmatched_files`: a file under the scope root
   matching neither `include` nor `exclude` produces an unmatched-file
   diagnostic.
7. **File:** `cmd/testdata/snapshots/check/`.
   Add snapshots for filesystem diagnostics: path-rule failure, parse warning,
   and filesystem unmatched file.
8. **File:** `internal/checks/registry_test.go`.
   Add descriptor tests for supported configuration sites and document-needs metadata.

### Phase 2 - Shared check metadata and config parsing

**Goal:** the registry describes where a check attaches and one parser serves
collection and filesystem config.

1. **File:** `internal/checks/registry.go`.
   Add configuration-site constants, `Descriptor.ConfigurableIn []string`, and
   helpers such as `SupportsConfiguration(kind, site)` and
   `DescriptorConfigurableIn(d)`. Treat an empty list as `collection` during
   migration.
2. **File:** `internal/checks/registry.go`.
   Add document-needs metadata to `Descriptor`, for example
   `NeedsDocument bool`, plus `NeedsDocument(kind)`. Filesystem scopes use this
   to decide whether to parse selected files.
3. **File:** `internal/checks/config.go` (new).
   Move the reusable raw check shape out of
   `internal/storage/collection/parse.go`. Define `RawCheck`, key validation,
   and a `BuildConfigured` helper that folds optional object schema shorthands
   and calls `checks.Parse`.
4. **File:** `internal/storage/collection/parse.go`.
   Replace `RawCheck` and `buildChecks` with the shared checks config helper.
   Keep collection-specific schema shorthand and variant wiring behavior
   byte-for-byte compatible.
5. **File:** `internal/storage/collection/parse.go`.
   Keep SQLite collection rejection based on descriptor family or target support
   so existing behavior stays stable.
6. **File:** `cmd/check_types.go`.
   Include supported configuration sites in `check-types show` and JSON output through the
   descriptor. Keep existing scope and severity output.
7. **File:** `cmd/gendocs/main.go`.
   Render supported configuration sites on generated check-type pages. Keep generated docs
   deterministic.

### Phase 3 - Filesystem scope config and expansion

**Goal:** filesystem bases load named scopes and expand them into deterministic
file sets.

1. **File:** `internal/storage/filesystemcheck/scope.go` (new).
   Add `RawScope` and `Scope` types with `Name`, `Path`, resolved `Root`,
   `Include`, `Exclude`, `ParseFailures`, and parsed `Checks`.
2. **File:** `internal/storage/filesystemcheck/scope.go` (new).
   Add `Build` to validate scope config: `include` required,
   `parseFailures` is `error` or `warning`, `checks` required, and every check
   supports the `filesystem` target.
3. **File:** `internal/storage/filesystemcheck/scope.go` (new).
   Add deterministic expansion over `os.DirFS(scope.Root)` using doublestar:
   selected files match at least one include and no exclude; unmatched files
   are regular files that match neither include nor exclude.
4. **File:** `internal/storage/filesystemcheck/scope_test.go` (new).
   Test include/exclude matching, sorted selected files, sorted unmatched files,
   missing directories, invalid globs, and default labels for unnamed scopes.
5. **File:** `internal/project/loader.go`.
   Add `FilesystemChecks []filesystemcheck.RawScope` to `rawBaseInstance`.
   Build scopes only for `type: filesystem`, resolve paths against the base
   root, and store them on `BaseInstance`.
6. **File:** `internal/project/loader.go`.
   Reject `filesystemChecks` on non-filesystem bases with a load-time error.
   Preserve legacy `.katalyst/storage/` readability by parsing the same field
   there when that legacy directory is used.
7. **File:** `internal/project/project.go`.
   Add a `FilesystemCheckScopes()` accessor or expose the loaded scopes through
   `Config.Bases` in a way `cmd/check.go` can use without knowing raw config.
8. **File:** `internal/project/projecttest/projecttest.go`.
   Add a helper for filesystem scope config only if it removes repeated YAML
   from loader and CLI tests.

### Phase 4 - File and file-set runtime contexts

**Goal:** collection-attached and filesystem-attached checks share runtime
contexts without breaking existing collection checks.

1. **File:** `internal/checks/checks.go`.
   Add `FileContext` as the canonical per-file context. Keep `Context` as an
   alias or compatibility wrapper during the migration.
2. **File:** `internal/checks/collection.go`.
   Add `FileSetContext` with `Root`, `Files`, `Unmatched`, `Include`, and
   `Exclude`. Include enough metadata for existing set-level checks and the new
   unmatched-files check.
3. **File:** `internal/checks/collection.go`.
   Add `FileSetCheck` and `RunFileSetAll`. Keep `CollectionCheck` and
   `RunCollectionAll` as compatibility wrappers until collection callers move.
4. **File:** `internal/checks/filesystem/unique_filename.go`.
   Convert `UniqueFilename` to the file-set context, or add a compatibility
   adapter if full conversion waits until Phase 6.
5. **File:** `internal/checks/filesystem/index_file_required.go`.
   Convert `IndexFileRequired` to the file-set context, preserving diagnostics.
6. **File:** `internal/checks/structuredobject/unique_field.go`.
   Convert `UniqueField` to the file-set context with metadata read from each
   file context.
7. **File:** `cmd/check.go`.
   Update collection-scoped execution to build the new `FileSetContext` while
   preserving existing output and selector behavior.

### Phase 5 - Filesystem check execution

**Goal:** `katalyst check` with no selectors runs filesystem scopes before
collection checks.

1. **File:** `cmd/filesystem_check.go` (new).
   Add the filesystem check runner: expand each scope, build runnable file and
   file-set checks, run file checks per selected file, then run file-set checks.
2. **File:** `cmd/engine.go`.
   Add helpers that build checks from an arbitrary list of
   `checks.ConfiguredCheck`, separate from collection variant routing. Reuse
   library availability checks and non-object builders.
3. **File:** `cmd/filesystem_check.go` (new).
   Parse selected files lazily only when a configured check needs document data.
   Strip the `schema` directive from metadata the same way `checkItem` does.
4. **File:** `cmd/filesystem_check.go` (new).
   Implement `parseFailures`: default error-severity violations fail the run;
   `warning` emits advisory diagnostics and does not fail by itself.
5. **File:** `cmd/filesystem_check.go` (new).
   Format filesystem diagnostics as
   `filesystem <label>: <path>[:line]: [warning: ]<loc>: <message>`.
   Keep collection diagnostics unchanged.
6. **File:** `cmd/check.go`.
   In the no-selector path, run all filesystem scopes before collection
   resolution. With one or more selectors, keep existing collection-only
   behavior.
7. **File:** `cmd/check.go`.
   Fold filesystem runner failures into the existing exit code contract:
   validation failures return exit 1; config and glob usage errors return exit
   2.

### Phase 6 - Dual-target check types

**Goal:** existing file-system family checks work under filesystem scopes, and
the new unmatched-files check ships.

1. **File:** `internal/checks/kinds.go`.
   Add `CheckFilesystemUnmatchedFiles`.
2. **File:** `internal/checks/filesystem/unmatched_files.go` (new).
   Implement `filesystem_unmatched_files` as a file-set check over
   `FileSetContext.Unmatched`. It emits one violation per unmatched file and
   includes include/exclude patterns in the message.
3. **File:** `internal/checks/filesystem/*.go`.
   Add `ConfigurableIn: []string{"collection", "filesystem"}` to file-system check
   descriptors that work under both attachment points.
4. **File:** `internal/checks/filesystem/name_matches_field.go`,
   `parent_dir_matches_field.go`, and `referenced_files.go`.
   Mark metadata-aware checks as needing document data so filesystem scopes
   parse selected files before running them.
5. **File:** `internal/checks/structuredobject/unique_field.go`.
   Keep the historical `filesystem_unique_field` kind, mark it as
   `structuredObject`, `collection` and `filesystem` target compatible, and
   document that it needs metadata.
6. **File:** `internal/checks/filesystem/filesystem_test.go`.
   Add unit tests for the unmatched-files check and direct file-set behavior
   where useful.
7. **File:** `cmd/testdata/snapshots/check-types/`.
   Regenerate snapshots for `check-types list`, `show`, and JSON if descriptor
   output changes.

### Phase 7 - Documentation and verification

**Goal:** docs, skills, and generated reference describe the new attachment
model.

1. **File:** `docs/content/deep-dives/domain-model/checks.md`.
   Split data family, library, configuration site, and runtime granularity.
2. **File:** `docs/content/reference/configuration.md`.
   Document `filesystemChecks` under filesystem bases. Note the legacy
   `.katalyst/storage/` reader only if that page still documents legacy config.
3. **File:** `docs/content/reference/glossary.md`.
   Add CollectionCheck, FilesystemCheck, FileCheck, FileSetCheck, and
   configuration site. Update Check instance and Collection-scoped check.
4. **File:** `docs/content/how-to/configure-rules.md`.
   Add a pre-collection filesystem check workflow.
5. **File:** `internal/checks/AGENTS.md`.
   Document descriptor configurableIn metadata and the runtime naming distinction.
6. **File:** `internal/storage/collection/AGENTS.md`.
   Document that collection `checks:` remain collection-attached and filesystem
   checks live on filesystem bases.
7. **File:** `.cursor/skills/add-katalyst-check-type/SKILL.md`.
   Update the checklist so new check types declare supported configuration sites
   and document needs.
8. **File:** `docs/content/reference/check-types/`.
   Run `make docs-gen` after descriptor rendering changes.
9. **File:** `cmd/check_test.go`, `internal/project/loader_test.go`,
   `internal/checks/...`, `internal/storage/...`.
   Run
   `go test ./cmd ./internal/project ./internal/checks/... ./internal/storage/...`.
   If generated docs or examples change, run the broader documented suite.

## Key Files

| File | Role |
|---|---|
| `product/specs/filesystem-checks-spec.md` | source spec this plan implements |
| `cmd/check.go` | orchestrates no-selector filesystem checks and existing collection checks |
| `cmd/engine.go` | shared check building and library availability helpers |
| `cmd/filesystem_check.go` | new filesystem-scope runner and diagnostics |
| `cmd/check_test.go` | CLI behavior, exit codes, and filesystem diagnostics |
| `cmd/testdata/snapshots/check/` | golden filesystem check diagnostics |
| `internal/project/loader.go` | parses base-level `filesystemChecks` and stores built scopes |
| `internal/project/project.go` | exposes filesystem scopes to the command layer |
| `internal/project/loader_test.go` | config parsing and validation coverage |
| `internal/project/projecttest/projecttest.go` | optional config helper for filesystem scope tests |
| `internal/storage/filesystemcheck/scope.go` | new raw scope config, build validation, and file-set expansion |
| `internal/storage/filesystemcheck/scope_test.go` | filesystem scope matching tests |
| `internal/storage/collection/parse.go` | consumes shared check config helper for collection checks |
| `internal/storage/collection/filesystem/collection.go` | reference for deterministic filesystem traversal and matching |
| `internal/checks/registry.go` | descriptor configuration sites, document needs, and registry helpers |
| `internal/checks/config.go` | shared raw check config parser for both attachment points |
| `internal/checks/checks.go` | per-file context compatibility layer |
| `internal/checks/collection.go` | file-set context and compatibility wrappers |
| `internal/checks/kinds.go` | new `filesystem_unmatched_files` kind constant |
| `internal/checks/filesystem/unmatched_files.go` | new unmatched-files check |
| `internal/checks/filesystem/*.go` | configurableIn metadata and document-needs metadata for file-system checks |
| `internal/checks/structuredobject/unique_field.go` | metadata-aware set-level check configurable in both sites |
| `cmd/check_types.go` | CLI descriptor rendering for supported configuration sites |
| `cmd/gendocs/main.go` | generated check-type page rendering for supported configuration sites |
| `docs/content/deep-dives/domain-model/checks.md` | durable explanation of configurableIn vs family vs granularity |
| `docs/content/reference/configs/bases.md` | user-facing filesystemChecks config reference |
| `docs/content/reference/glossary.md` | new vocabulary |
| `docs/content/how-to/configure-rules.md` | pre-collection filesystem check workflow |
| `internal/checks/AGENTS.md` | developer convention for check descriptors |
| `internal/storage/collection/AGENTS.md` | developer convention for attachment homes |
| `.cursor/skills/add-katalyst-check-type/SKILL.md` | contributor workflow for new check types |

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Config home | base-level `filesystemChecks` | the current code's configured backend instance is a base, and the base owns root path resolution |
| Legacy storage config | parse the same field when legacy `.katalyst/storage/` is used | keeps legacy storage files aligned with base config behavior |
| Collection config key | keep collection `checks:` | the collection block already supplies the configuration site and existing configs keep working |
| Shared check parsing | move raw check parsing into `internal/checks` | collection and filesystem attachments need the same `kind` and args parser without duplicating `RawCheck` |
| ConfigurableIn metadata | descriptor-level `ConfigurableIn` with empty meaning collection | existing descriptors migrate incrementally and docs can render one source of truth |
| Document parsing | descriptor-level document-needs metadata | filesystem scopes parse lazily and path-only checks stay path-only |
| Parse failure policy | scope-level `parseFailures`, default `error` | CI stays strict by default while onboarding can opt into warnings |
| Runtime contexts | add file and file-set contexts with compatibility wrappers | the implementation can share checks without renaming every interface in one step |
| Filesystem unmatched files | new `filesystem_unmatched_files` check | raw subtree coverage is opt-in and separate from collection unmatched-file invariants |
| Selective filesystem execution | defer named-scope selectors | no-selector execution covers onboarding and CI while avoiding selector grammar churn |

## Documentation updates

- **Phase 7, File:** `docs/content/deep-dives/domain-model/checks.md`.
  Explain data family, library, configuration site, and runtime granularity.
- **Phase 7, File:** `docs/content/reference/configuration.md`.
  Add `filesystemChecks` keys, defaults, examples, parse-failure severity, and
  unmatched-files check usage.
- **Phase 7, File:** `docs/content/reference/glossary.md`.
  Add the new check vocabulary and update existing check-instance wording.
- **Phase 7, File:** `docs/content/how-to/configure-rules.md`.
  Add the first workflow that runs checks before collections exist.
- **Phase 7, File:** `internal/checks/AGENTS.md`.
  Record descriptor target and document-needs conventions.
- **Phase 7, File:** `internal/storage/collection/AGENTS.md`.
  Record where collection-attached checks and filesystem-attached checks live.
- **Phase 7, File:** `.cursor/skills/add-katalyst-check-type/SKILL.md`.
  Add target and document-needs steps to the new-check workflow.
- **Phase 7, File:** `docs/content/reference/check-types/`.
  Regenerate with `make docs-gen` after descriptor rendering changes.

## Out of Scope

- Selective filesystem scope execution such as `katalyst check --filesystem docs`.
- Renaming every internal `Check` and `CollectionCheck` symbol in one sweep.
- Running object schemas from filesystem scopes.
- Filesystem checks for SQLite or other non-filesystem base types.
- Inferring collections from filesystem scopes.
- Changing collection unmatched-file behavior.
- Adding default `include` patterns.
