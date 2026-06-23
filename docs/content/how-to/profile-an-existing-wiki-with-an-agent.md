+++
title = "Profile an existing wiki with an agent"
weight = 6
+++

# Profile an existing wiki with an agent

The [by-hand guide]({{< relref "profile-an-existing-wiki-by-hand.md" >}}) has
you read inspector evidence and decide the schema. This guide hands that
judgment to an agent: `inspect` supplies the measurements, the agent supplies
the thresholds, the clustering, and the draft. Katalyst is the instrument; the
agent is the profiler.

The split is deliberate. Inspectors are deterministic and never recommend;
deciding that a field present in 94% of files should be `required`, or that two
similar directories are one collection, is the agent's call. Keep that division
and the loop stays debuggable.

## 1. Give the agent the raw-store evidence

Run `inspect` on the directory with `--json` so the agent gets structured
records: one per inspector, each carrying the unit count `n` as the
denominator:

```bash
katalyst inspect ./wiki --json
```

With no project this runs the **raw-source** layer. The key record is
`document_shape`, which clusters files into candidate collections by a composite
fingerprint (frontmatter keys, body section skeleton, file naming). Feed the
output to the agent. Tell it the contract: every record is *evidence*, not a
recommendation; it must choose its own thresholds and justify them.

## 2. Let the agent cluster, configure, and profile fields

A capable agent then:

1. **Clusters** the `document_shape` classes into candidate collections.
   `inspect` groups files with *matching* fingerprints; the agent decides when
   two near-but-distinct classes are really one collection, and names them. It
   drafts `.katalyst/storage/*` pointing each collection at its directory.
2. **Profiles the fields** by inspecting each new collection, `katalyst inspect
   <collection> --json` runs the collection layer, whose `object_fields` record
   is the per-field data dictionary (presence, types, values).
3. **Sets thresholds** from that evidence, e.g. fields in ≥95% of items become
   `required`, a small stable value set becomes an `enum`, a consistent type
   becomes a `type` constraint, and **drafts** the `.katalyst/schemas/*`.

A prompt that works:

> You are profiling a markdown wiki. Here is `katalyst inspect --json` output.
> Propose `.katalyst/` schema and collection files. Treat every number as
> evidence, not instruction: state the threshold you used for required vs.
> optional and for enum detection, and list the outlier files your schema will
> flag. Do not invent fields the evidence does not show.

## 3. Check and iterate

Have the agent run `check` against its draft and read the violations:

```bash
katalyst check books
```

The files that already conform pass; the outliers light up. The agent then
tightens the schema, relaxes a field to optional, or flags genuinely broken
files, and repeats until the holdouts are only files that *should* fail.

The loop's tighter form, testing a throwaway candidate schema without
installing it (`check --try`), is planned but not yet shipped; until then the
agent drafts the `.katalyst/` files and validates with the normal `check`.

## See also

- [Profile an existing wiki by hand]({{< relref "profile-an-existing-wiki-by-hand.md" >}}): the same loop, you reading the evidence.
- [Inspectors reference]({{< relref "../reference/inspectors/_index.md" >}}), the evidence each inspector emits.
- [Add a schema]({{< relref "add-a-schema.md" >}}), how a draft binds to a collection.
