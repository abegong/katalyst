# Plan - AGENTS.md coverage
> Issue: #35

## Current State

The issue's original directory list is stale. Since it was filed, several
packages have moved or collapsed:

- `internal/config` was folded into `internal/project`;
- `internal/query` moved under storage collection concerns and is being split
  further by the listing/predicate work;
- `internal/frontmatter` and `internal/validator` no longer appear as main
  ownership homes in the current tree;
- `internal/codec/markdownbodytext` now owns markdown/frontmatter parsing.

Existing local guidance already covers some important areas:

- `cmd/AGENTS.md`;
- `cmd/testdata/AGENTS.md`;
- `internal/checks/AGENTS.md`;
- `internal/checks/jsonschema/testdata/AGENTS.md`;
- `internal/examples/AGENTS.md`;
- `internal/fix/AGENTS.md`;
- `internal/inspect/AGENTS.md`;
- `internal/storage/collection/AGENTS.md`;
- root `AGENTS.md`.

## Goal

Finish #35 by ensuring every main directory has a concise local map of purpose,
conventions, and gotchas, without duplicating root guidance or long-form docs.

## Implementation Steps

### 1. Refresh the Directory Inventory

Generate a current inventory before writing docs:

```bash
find . -maxdepth 3 -type d \
  -not -path './.git*' \
  -not -path './docs/resources/_gen*'
```

Classify directories into:

- main ownership homes that should have `AGENTS.md`;
- subpackages already covered by a parent `AGENTS.md`;
- generated, testdata, or vendored paths that should not receive local guidance;
- obsolete paths from the original issue body.

### 2. Identify Missing Main-Directory Guidance

Likely candidates to audit:

- `docs/` - Hugo module, generated docs, build process, dogfooding expectations;
- `internal/project/` - project loader, selector resolution, collection/item
  aliases, config layout;
- `internal/storage/` - storage root conventions and backend boundaries;
- `internal/codec/` or `internal/codec/markdownbodytext/` - codec ownership and
  parse/encode expectations;
- any current top-level `internal/*` package not already covered by an
  `AGENTS.md`.

Avoid adding one-file docs to tiny implementation subdirectories when a parent
guide gives clearer ownership guidance.

### 3. Use a Consistent Shape

Each new or updated `AGENTS.md` should be short and local:

- what lives here;
- what must not live here;
- package boundaries and import direction;
- how to add a new thing;
- test or generation commands specific to the area;
- links to long-form docs when the rationale is already documented elsewhere.

Do not copy general repo rules from root `AGENTS.md`.

### 4. Verify Coverage

Before closing #35, produce a small checklist in the PR description or issue
comment:

- directories audited;
- AGENTS files added or updated;
- directories intentionally covered by a parent;
- obsolete paths removed from the original checklist.

## Acceptance Checklist

- [x] The current main-directory inventory has been audited.
- [x] Every main ownership directory has concise local guidance or a documented
      reason it is covered by a parent.
- [x] New files use a consistent format with existing `cmd/AGENTS.md` and
      package guidance.
- [x] Stale paths from the issue body are explicitly accounted for.
- [x] Relevant docs/checks/tests are run for any touched area.

## Completed Coverage

Added or confirmed local guidance for the current main ownership homes:

- `docs/AGENTS.md` - Hugo module boundaries, generated docs, and dogfooding
  checks;
- `internal/project/AGENTS.md` - project loading, selectors, and storage import
  direction;
- `internal/storage/AGENTS.md` - backend registry and opaque references;
- `internal/storage/collection/AGENTS.md` - existing collection read-stack
  guidance;
- `internal/codec/markdownbodytext/AGENTS.md` - markdown/frontmatter codec
  boundaries;
- `internal/checks/AGENTS.md`, `internal/inspect/AGENTS.md`,
  `internal/fix/AGENTS.md`, and `internal/examples/AGENTS.md` - existing local
  package guidance;
- `internal/skillpack/AGENTS.md` - skill archive packaging rules.

Original issue paths now covered by renamed or collapsed ownership homes:

- `internal/config/` is covered by `internal/project/`;
- `internal/frontmatter/` is covered by `internal/codec/markdownbodytext/`;
- `internal/query/` is covered by `internal/storage/collection/`;
- `internal/validator/` is no longer a current main directory; schema-backed
  validation lives under `internal/checks/jsonschema/`, covered by
  `internal/checks/AGENTS.md`.
