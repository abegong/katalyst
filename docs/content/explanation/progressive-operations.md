+++
title = "Progressive Operations"
weight = 30
+++

# Progressive Operations

_How data interfaces evolve as query complexity grows. Each tier unlocks new operations — but requires structural commitments the previous tier doesn't._

The core thesis: many knowledge systems start as filesystems and progressively acquire database-like structure. The progression isn't arbitrary — each tier is driven by a class of operations that can't be satisfied at the previous level.

---

## Tier 1 — Filesystem

**Structural commitment:** none beyond path conventions

**Operations unlocked:**
- Read/write by path
- List/enumerate (directory traversal)
- Full-text search (grep-style, substring match)
- Vector/semantic search — operates on raw content; no schema required

**Limitations:**
- Queries are global scans (no index)
- No structured fields to filter on
- No relationships between files

---

## Tier 2 — Document Store

**Structural commitment:** optional schemas (e.g. frontmatter conventions). Not enforced, but consistently applied.

**Operations unlocked:**
- Query by structured fields across documents ("all people where `closeness: close`")
- Faceted search — filter + sort by frontmatter fields
- Vector search becomes schema-aware (can filter semantic results by field values)
- Field-level updates (change one field without rewriting the whole file)

**Limitations:**
- No enforced referential integrity — relationships are naming conventions, not constraints
- Aggregations are fragile (depend on field consistency)
- Many-to-many relationships require awkward denormalization

---

## Tier 3 — Relational

**Structural commitment:** schemas required, foreign keys, typed fields

**Operations unlocked:**
- Relational queries ("meetings attended by this person", "all open action items from meetings this month")
- Foreign key constraints — referential integrity enforced
- Aggregations ("intros sent per quarter, by status")
- Time series — just a table with a timestamp column; no special tier needed

**Limitations:**
- Many-to-many relationships require join tables (that's Tier 4)
- Schema migrations have real cost

---

## Tier 4 — Join Tables

**Structural commitment:** intersection tables for many-to-many relationships

**Operations unlocked:**
- True many-to-many queries ("all people who attended meetings tagged #fundraising")
- Proper intersection entities (a meeting_attendee row can carry its own fields: role, spoke_time, etc.)
- More complex relational queries without denormalization

---

## Tier 5 — Graph

**Structural commitment:** relationships are first-class entities with their own attributes and types

**Operations unlocked:**
- Multi-hop traversal ("who introduced me to someone who works at [firm]?")
- Relationship-typed queries ("what projects is this person a collaborator on vs. a contact for?")
- Path queries ("how am I connected to X?")