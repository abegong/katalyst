package cmd_test

import "testing"

// wantRootHelp is the exact output of `katalyst` with no arguments. It is kept
// as one inspectable block so a reviewer can read the whole help surface at a
// glance and see the noun/verb grouping (Verbs vs Resources) the command tree
// is designed around — see docs/deep-dives/command-organization.md.
const wantRootHelp = `katalyst validates structured metadata (frontmatter) on
markdown files against JSON Schema documents.

Usage:
  katalyst [command]

Verbs:
  check       Run configured checks against the selected items.
  fix         Apply deterministic, safe fixes to the selected items.
  init        Prepare the current directory as a katalyst project.
  inspect     Profile a directory of markdown files and report its shape.

Resources:
  collection  Inspect collections defined under .katalyst/collections/.
  item        List, inspect, and mutate items within collections.
  rules       Inspect the check kinds the engine can enforce, grouped by family.
  schema      Inspect schemas defined under .katalyst/schemas/.

Additional Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

Flags:
  -h, --help      help for katalyst
  -v, --version   version for katalyst

Use "katalyst [command] --help" for more information about a command.
`

func TestRoot_noArgs_printsGroupedHelp(t *testing.T) {
	stdout, stderr, err := runRoot(t)
	if err != nil {
		t.Fatalf("root with no args: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got:\n%s", stderr)
	}
	if stdout != wantRootHelp {
		t.Errorf("root help mismatch.\n--- got ---\n%s\n--- want ---\n%s", stdout, wantRootHelp)
	}
}
