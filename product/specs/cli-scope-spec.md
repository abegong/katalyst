# Spec - CLI scope

> **Status: done.** Scopes Katalyst to a strictly local CLI tool: bases must be
> local and in-process, and the docs stop promising a server form factor or
> networked backends. All six decisions are locked and shipped; Open Questions
> is empty. See [the plan](./cli-scope-plan.md) for decision 6's implementation.

## Overview

Katalyst's docs describe a tool with a wider deployment surface than the code
has ever had: a server form factor, and bases extending to Postgres, S3, and
hosted APIs. Nothing in the codebase implements either. This spec draws the
boundary explicitly so the promise matches the product, and so future backend
proposals have a rule to be measured against instead of a case-by-case debate.

The boundary is one sentence: **a base must be local and in-process. No network,
no credentials, no daemon.** Everything else in this spec follows from it.

## Value

- **The docs stop over-promising.** A reader who takes "and hosted APIs" at face
  value expects a roadmap that does not exist.
- **Backend proposals get a test.** "Local and in-process" is checkable at review
  time. "And others later" is not.
- **Simplifications become bankable.** Committing to no network makes
  `context.Context` unnecessary on the data path, permanently, rather than a
  question reopened with every new base type.
- **A supported backend stops having a permanent hole.** Once SQLite is
  in-scope-forever rather than experimental, `fix` not working on it is a defect
  rather than an acceptable gap.

## Current State

### The server form factor is already gone

Commit `9a97721` removed the **Multiple form factors** section, which named a
linter, a CLI, and "a server that enforces rules on write operations for SQL and
NoSQL stores." Two orphaned sentences survived in `README.md` and
`docs/content/welcome.md` and were removed on this branch. No code ever backed
the idea: there is no listener, no `serve` command, and the only `net/http`
import is an outbound client in `cmd/skills.go` fetching skill bundles from the
GitHub Releases API.

### The remote-backend promise is prose, not machinery

Six doc sites promise backends the code has no affordance for:

| Site | Promise |
|---|---|
| `docs/content/deep-dives/domain-model/base.md:14` | "extend to backends such as Postgres, S3, and hosted APIs" |
| `base.md:22`, `docs/content/reference/glossary.md:22` | BaseType: "`postgresql`, `mongodb`, and others later" |
| `base.md:25` | Base reference: "a file path, S3 key, table name" |
| `docs/content/deep-dives/domain-model/collections.md:48-52` | Mapping table: Postgres, MongoDB, a REST API, an S3 bucket |
| `internal/storage/doc.go:6-7` | "filesystem and sqlite today; postgresql, mongodb later" |

What a networked base would need does not exist:

- **No `context.Context` anywhere in the repo.** No cancellation, no deadlines on
  any data path. The one timeout in the codebase is
  `http.Client{Timeout: 30 * time.Second}` at `cmd/skills.go:78`.
- **No connection configuration.** `BaseInstance` (`internal/project/loader.go:85-100`)
  has exactly one locator field, `Root string`, resolved as a filesystem path
  even for SQLite. No DSN, endpoint, credential, or TLS handling.
- **No transient-failure model.** The SQLite backend opens and closes a `*sql.DB`
  per call. No pooling, no retry, no distinction between "absent" and
  "unreachable."
- **No network dependency.** `modernc.org/sqlite` is pure Go, in-process, reading
  a local file.

### The seam is narrower than the docs claim

`CollectionDefinition` (`internal/storage/collection/collection.go:31`) covers
discovery only: `Scope`, `Collections`, `Items`, `Unmatched`, `Reference`.
Content read and write are not in the interface. They are six
`case storage.SQLite:` branches in `internal/project/project.go` (lines 139, 172,
195, 214, 228, 242) that assert to the concrete `*sqlitestore.Definition`, plus
three more backend conditionals in `cmd/fix.go:171` and `cmd/item.go:362,449`.
Two further leaks: `ItemPath` (`project.go:82-85`) hardcodes the filesystem
definition regardless of base type, and `Item.Path` is documented as a filesystem
path while SQLite fills it with a synthetic label.

The abstraction that would supposedly absorb Postgres does not absorb SQLite
without nine special cases.

## Design

### 1. SQLite stays

SQLite is a local file read in-process, with no network and no daemon. It
satisfies the rule and it is the only worked example proving that item and
collection are domain roles rather than file counts: one file is one item, one
row is one item, both correct. Removing it would delete roughly 1,300 lines and
cost the domain model its evidence. It stays.

### 2. The base rule

> **A base must be local and in-process. No network, no credentials, no daemon.**

This replaces "`postgresql`, `mongodb`, and others later" everywhere it appears.
It admits future local backends (DuckDB, Parquet, a git object store) and
excludes Postgres, MongoDB, S3, and hosted APIs by construction. It is the single
statement future backend proposals are measured against.

The rule governs **bases**. It does not govern check libraries: an
out-of-process `CheckLibrary` such as Vale remains in scope, because it is a
local subprocess rather than a networked service (see decision 5).

### 3. Cross-backend illustrations stay, scoped

The mapping table at `collections.md:48-52` is pedagogy, not roadmap. It is how
the docs argue that the collection/item hierarchy is not filesystem-specific, and
the argument is weaker without the contrast. The table stays verbatim, prefixed
with a scope note: Katalyst operates on local bases, and these rows show the
model is not tied to one storage shape.

The base-reference gloss is the one place this does not extend. It read "a file
path, S3 key, table name, or similar backend address," one line below the new
rule naming S3 as excluded, and a definition cannot cite as an example the thing
the paragraph above it rules out. The gloss keeps the opacity point without the
example: a reference is a file path or a table name, kept opaque rather than
path-shaped so a base that addresses its content some other way fits without
changing the model. `storage.Reference`'s doc comment loses "or object key
later" for the same reason.

The distinction that makes this consistent: the `collections.md` table maps
Katalyst's *vocabulary* onto systems the reader already knows, so naming
Postgres there teaches. A term definition naming S3 reads as a claim about what
Katalyst addresses, so it misleads.

### 4. Progressive operations describes the destination, not the host

`docs/content/deep-dives/why-katalyst/progressive-operations.md` tiers 3 to 5
(relational, graph) presume databases. Reframe rather than cut. The tiers
describe **what a knowledge base grows into**, and Katalyst's job is to enforce
the structural commitments that make each tier reachable, not to host the tier
itself. A corpus that passes its checks is a corpus that can be migrated into a
relational store; Katalyst is what gets it ready. That reading survives the
boundary intact and keeps the essay's argument.

### 5. No `context.Context` on the data path

Katalyst commits to no `context.Context` parameter on any data-path function.

Local file I/O does not usefully hang: `os.ReadFile` returns promptly or the disk
is broken. Cancellation earns its cost only against a resource that can block
indefinitely, which the base rule now excludes.

The commitment is cheap to reverse if the rule is ever broken. Retrofitting ctx
touches roughly 13 signatures (5 on `CollectionDefinition`, 8 on `Project`) and
30 call sites, all funneling through the single dispatch at `project.go:53`. The
expensive half is never implicated: the 29 `Run(ctx checks.Context)` methods stay
unchanged, because `checks.FileContext` carries fully materialized content and
checks are pure CPU over in-memory data.

**Out-of-process check libraries own their own timeouts.** `SchemaLibrary`
already anticipates a subprocess provider: `Available()` exists so an
out-of-process tool can probe its binary, `internal/checks/library_oop_test.go`
models a batched out-of-process run, and Vale is the named next library. A
subprocess can hang like a socket can. That timeout belongs inside the library
that spawns it, via `exec.CommandContext` with a context the library creates,
and never appears in `Check.Run`'s signature. Locality of the hang means
locality of the fix.

Note the naming hazard for future readers: `checks.Context` is an alias for
`FileContext`, so `Run(ctx Context)` already means "the file context." A
cancellation context introduced here would collide.

### 6. `fix` is a text-form verb

`fix` currently refuses SQLite collections outright
(`cmd/fix.go:171-172`, "not supported for sqlite collections yet"). That gate is
on the wrong property.

**The engine is already portable.** `internal/fix` does no file I/O and computes
the canonical form from bytes. The SQLite backend already produces exactly the
shape it consumes: `rawDocument` (`sqlite/collection.go:459`) synthesizes
frontmatter plus body and `Read` returns a `*markdownbodytext.Document`. What
blocks SQLite is plumbing: `fixOne` calls `os.ReadFile` and `filesystem.Write`
directly, bypassing `project.ReadItem` and `project.UpdateItem`, both of which
already have working SQLite branches. `cmd/item.go` routes correctly; `fix` is
the one verb that does not.

**What each half of `fix` means per backend:**

| Half | Filesystem | SQLite |
|---|---|---|
| `text_forbids` body fixes | Rewrites the body | Rewrites the content column, when one is mapped |
| Frontmatter canonicalization | Real: sorts keys, normalizes block style and trailing newline in the stored text | No-op by construction: attributes are columns, there is no stored key order, and the YAML being canonicalized was synthesized from a map moments earlier |

So the correct gate is a capability, not a backend name: **does this collection
expose a text body?** Every filesystem collection does. A SQLite collection does
when it maps a `ContentColumn`. A pure-attribute SQLite table does not, and for
it `fix` has nothing to operate on.

Design:

- Add `Collection.HasTextContent()` in `internal/storage/collection`, where
  `StorageType`, `ContentKind`, and `ContentColumn` already live. It consolidates
  the backend conditional into one place instead of scattering it.
- Route `fixOne` through `project.ReadItem` and `project.UpdateItem` instead of
  `os.ReadFile` and `filesystem.Write`.
- Replace the `StorageType == sqlite` gate with `HasTextContent()`, and make the
  error name the real reason: `fix requires a text content mapping; collection
  %q maps only attributes`.

This removes one of the nine backend conditionals rather than adding one, and it
follows the instruction already in `internal/storage/collection/sqlite/AGENTS.md`:
*"Do not make check families backend-aware to compensate."* Gate on capability,
not on backend identity.

## Open Questions

_None._

## Documentation updates

**Developer docs:**

- `internal/storage/doc.go`: replace "postgresql, mongodb later" with the base
  rule.
- `internal/storage/AGENTS.md`: state the rule as the criterion for adding a
  BaseType; today it says only "when its `CollectionDefinition` implementation
  exists."
- `internal/storage/collection/sqlite/AGENTS.md`: drop "`fix` is not part of the
  first SQLite cut" once decision 6 lands.
- `internal/checks/checks.go` package doc: record the no-ctx commitment and the
  out-of-process-owns-its-timeout rule, so a future subprocess library does not
  reach for a ctx parameter first.
- `.cursor/skills/write-domain-model/SKILL.md:122`: the Postgres/MongoDB mapping
  reference needs the same scope note as `collections.md`.

**User docs:**

- `docs/content/deep-dives/domain-model/base.md`: rewrite line 14 to the base
  rule; update the BaseType row; drop the S3-key example from the base-reference
  gloss while keeping its opacity point.
- `docs/content/reference/glossary.md:22`: BaseType row matches `base.md`.
- `docs/content/deep-dives/domain-model/collections.md`: add the scope note above
  the mapping table; table rows unchanged.
- `docs/content/deep-dives/why-katalyst/progressive-operations.md`: add the
  destination-not-host framing so tiers 3 to 5 read as what a corpus grows into.
- `docs/content/reference/cli.md` and the `fix` help text: state that `fix`
  requires a text content mapping.
- `README.md`, `docs/content/welcome.md`: done on this branch (server form factor
  removed).

## Test checklist

Decision 6 is the only behavior change. Per `AGENTS.md`, each arrives as a
failing test first.

- [ ] `HasTextContent` is true for a filesystem collection.
- [ ] `HasTextContent` is true for a SQLite collection with a `content` column,
      false without one.
- [ ] `fix` applies a `text_forbids` body fix to a SQLite item with a content
      column and persists it to that column.
- [ ] `fix` on a SQLite collection with no content column exits with the
      attributes-only message, not the old "not supported yet" text.
- [ ] `fix --check` on a SQLite content column reports the change without
      writing.
- [ ] Filesystem `fix` behavior is byte-for-byte unchanged (existing snapshots
      hold).
- [ ] A bad fix template on a SQLite item fails the run, matching the filesystem
      re-check contract in `applyTextFixes`.

## Rejected alternatives

- **Drop SQLite and collapse to filesystem only.** Removes about 1,300 lines and
  the whole `CollectionDefinition` interface. Rejected: SQLite is local and
  in-process, so it costs nothing against the rule, and it is the only evidence
  that the collection/item model generalizes.
- **Delete the cross-backend mapping table.** Rejected: it is how the docs teach
  that item and collection are roles, not file counts. A scope note is cheaper
  and keeps the argument.
- **Thread `context.Context` through the storage seam now as a hedge.** Costs 13
  signatures and 30 call sites for a capability the rule says will not be needed,
  and the retrofit is equally cheap later. Rejected as speculative.
- **Keep `fix` filesystem-only and drop the word "yet."** Rejected: a SQLite
  collection with a markdown content column is exactly the codec layer's
  dogfooding case, and under the base rule SQLite is permanent, so the hole would
  be permanent too.
