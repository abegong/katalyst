+++
title = "Variants"
weight = 60
+++

# Variants

A collection runs its base `schema`/`checks` against every item. **Variants**
let it run *extra* checks on a subset, chosen by the item's metadata. Each
entry in a collection's `variants:` list has a `when` discriminator and its own
`schema`/`checks`:

```yaml
pages:
  path: docs/content
  pattern: "**/*.md"
  schema: page                  # base: every page needs a title
  variants:
    - when: "bookCollapseSection"   # section landing pages have this flag
      schema: section_index
    - when: "!bookCollapseSection"  # every other page is a content page
      schema: content_page
      checks:
        - kind: object_required_field
          field: weight
        - kind: markdown_requires_h1
  useExhaustiveVariants: false   # default
```

**`when`** is a list of [`item list --filter`]({{< relref "../cli.md#filter-predicates" >}})
predicates (`field=value`, `field>=n`, `field=~regex`, `!field`, ...), evaluated
against the item's frontmatter. All entries must hold (AND). Three shapes are
accepted, the first two desugaring to the third:

```yaml
when: "kind=section"             # one predicate
when: ["kind=section", "w>1"]    # a list of predicates
when: { where: ["kind=section"] }
```

## Resolution

An item runs the base checks plus the checks of the **first** variant (in list
order) whose `when` it satisfies, at most one variant applies. A variant *adds*
to the base, so a check belongs in a variant exactly when some page type must
skip it: in the example, `weight` and the H1 requirement apply to content pages
but not section indexes. A variant may declare no checks at all (a deliberate
exemption).

An item that matches **no** variant runs the base checks alone. Set
**`useExhaustiveVariants: true`** to instead make an unmatched item a check
failure (`matches no variant`), so every item is provably accounted for.

Discrimination is by metadata only; selecting items by path or filename is not
supported yet (a page type distinguishable only by location needs a frontmatter
marker). `pattern` still governs collection **membership** and which files are
reported as [unmatched]({{< relref "../../deep-dives/domain-model/_index.md" >}}#invariants);
variants only route checks.
