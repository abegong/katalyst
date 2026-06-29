+++
title = "Configs"
weight = 30
bookCollapseSection = true
aliases = ["/reference/configuration/"]
+++

# Configs

Katalyst reads a `.katalyst/` directory, found by walking upward from the
current working directory to the nearest ancestor that contains one. That
ancestor is the repo root; all relative paths resolve against it.

The config reference is organized by concept:

- [Discovery]({{< relref "discovery.md" >}}): root discovery, symlinks, file layout, formats, and legacy storage paths.
- [Schemas]({{< relref "schemas.md" >}}): schema file discovery, naming, and schema handles.
- [Bases]({{< relref "bases.md" >}}): backend stores and their top-level config keys.
- [Collections]({{< relref "collections.md" >}}): collection membership, identity, paths, and per-collection files.
- [Checks]({{< relref "checks.md" >}}): `schema:` shorthand, `checks:` entries, text rules, and object-schema precedence.
- [Variants]({{< relref "variants.md" >}}): conditional check routing with `when`.
- [Listing]({{< relref "listing.md" >}}): default behavior for `katalyst item list`.

For *why* the config is shaped this way, see [How collections
work]({{< relref "../../deep-dives/domain-model/collections.md" >}}). To set one up step by
step, see [Configure checks for a
collection]({{< relref "../../how-to/configure-rules.md" >}}).

## Layout

```
.katalyst/
  config.yaml          # optional: listing defaults and discovery settings
  schemas/             # one JSON Schema file per named schema
    book.json
  bases/               # one file per base
    local.yaml         # a base + the collections it declares
    local/             # optional: one file per collection (escape hatch)
      books.yaml
```

## See also

- [Check types reference]({{< relref "../check-types/_index.md" >}}), every check type.
- [Bases]({{< relref "../../deep-dives/domain-model/base.md" >}}), the base /
  collection-mapping model and its lineage.
- [Collections]({{< relref "../../deep-dives/domain-model/collections.md" >}}), the
  config/collection model and rationale: schema resolution, variants,
  unmatched-as-error.
