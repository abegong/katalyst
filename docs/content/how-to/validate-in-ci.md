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
| `2` | Usage error (e.g. no `katalyst.yaml` found) |

## Gate on validation

`katalyst check` over the whole project exits non-zero if any item has a
violation:

```bash
katalyst check
```

## Gate on formatting

`katalyst fix --check` writes nothing; it lists items whose frontmatter is
not canonical and exits `1` if any are found:

```bash
katalyst fix --check
```

## Example GitHub Actions step

```yaml
- name: Validate content
  run: |
    make build
    ./bin/katalyst check
    ./bin/katalyst fix --check
```

The `check` step enforces schema and structural rules; the `fix --check`
step enforces canonical frontmatter without modifying files. See
[Formatting]({{< relref "../explanation/formatting.md" >}}) for why `fix` is
non-destructive in this mode.

## See also

- [Configure checks for a collection]({{< relref "configure-rules.md" >}})
- [Configuration reference]({{< relref "../reference/configuration.md" >}})
