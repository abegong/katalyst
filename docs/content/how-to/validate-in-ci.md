+++
title = "Validate in CI"
weight = 30
+++

# Validate in CI

Run Katalyst in continuous integration so malformed frontmatter and
non-canonical formatting fail the build instead of reaching `main`.

## Exit codes

Both gating commands use the same convention:

| Code | Meaning |
|---|---|
| `0` | Everything passed |
| `1` | One or more items failed (validation or formatting) |
| `2` | Usage error (e.g. no `.katalyst/` directory found) |

## Gate on validation

`katalyst check` over the whole project exits non-zero if any item has a
violation:

```bash
katalyst check
```

An item with a violation, here one missing its H1, prints the diagnostic and
the run exits 1, the non-zero status that fails the CI step:

{{< katalyst-example "ci-check-fails" >}}

## Gate on formatting

`katalyst fix --check` writes nothing; it lists items whose frontmatter is
not canonical and exits `1` if any are found:

```bash
katalyst fix --check
```

It prints one line per non-canonical item and exits 1, writing nothing. Here
`messy.md` has unsorted keys while `tidy.md` is already canonical:

{{< katalyst-example "ci-fix-check" >}}

## Example GitHub Actions step

```yaml
- name: Validate content
  run: |
    make build
    ./bin/katalyst check
    ./bin/katalyst fix --check
```

The `check` step enforces schema and structural checks; the `fix --check`
step enforces canonical frontmatter without modifying files. See
[Fix]({{< relref "../deep-dives/domain-model/fix.md" >}}) for why
`fix` is opinionated and non-destructive in this mode.

## See also

- [Configure checks for a collection]({{< relref "configure-rules.md" >}})
- [Configuration reference]({{< relref "../reference/configuration.md" >}})
