package cmd_test

import "testing"

// Root help is snapshot-tested from a fixture so the exact help contract is
// reviewable as plain text and shared with broader help-output snapshot tests.

func TestRoot_noArgs_printsGroupedHelp(t *testing.T) {
	stdout, stderr, err := runRoot(t)
	if err != nil {
		t.Fatalf("root with no args: %v", err)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got:\n%s", stderr)
	}
	snapshot(t, "help/root.txt", stdout)
}
