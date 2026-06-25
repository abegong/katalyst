package cmd_test

import "testing"

func TestTopLevelHelpSnapshots(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		args    []string
	}{
		{name: "inspect help", fixture: "help/inspect.txt", args: []string{"inspect", "--help"}},
		{name: "init help", fixture: "help/init.txt", args: []string{"init", "--help"}},
		{name: "check help", fixture: "help/check.txt", args: []string{"check", "--help"}},
		{name: "fix help", fixture: "help/fix.txt", args: []string{"fix", "--help"}},
		{name: "collection help", fixture: "help/collection.txt", args: []string{"collection", "--help"}},
		{name: "item help", fixture: "help/item.txt", args: []string{"item", "--help"}},
		{name: "schema help", fixture: "help/schema.txt", args: []string{"schema", "--help"}},
		{name: "skills help", fixture: "help/skills.txt", args: []string{"skills", "--help"}},
		{name: "check-types help", fixture: "help/check-types.txt", args: []string{"check-types", "--help"}},
		{name: "inspectors help", fixture: "help/inspectors.txt", args: []string{"inspectors", "--help"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runRoot(t, tc.args...)
			if err != nil {
				t.Fatalf("runRoot(%v): %v", tc.args, err)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got:\n%s", stderr)
			}
			snapshot(t, tc.fixture, stdout)
		})
	}
}
