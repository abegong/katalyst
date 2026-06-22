package cmd_test

import "testing"

func TestTopLevelHelpSnapshots(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		args    []string
	}{
		{name: "inspect help", fixture: "inspect-help.txt", args: []string{"inspect", "--help"}},
		{name: "init help", fixture: "init-help.txt", args: []string{"init", "--help"}},
		{name: "check help", fixture: "check-help.txt", args: []string{"check", "--help"}},
		{name: "fix help", fixture: "fix-help.txt", args: []string{"fix", "--help"}},
		{name: "collection help", fixture: "collection-help.txt", args: []string{"collection", "--help"}},
		{name: "item help", fixture: "item-help.txt", args: []string{"item", "--help"}},
		{name: "schema help", fixture: "schema-help.txt", args: []string{"schema", "--help"}},
		{name: "check-types help", fixture: "check-types-help.txt", args: []string{"check-types", "--help"}},
		{name: "inspectors help", fixture: "inspectors-help.txt", args: []string{"inspectors", "--help"}},
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
			want := mustHelpFixture(t, tc.fixture)
			if stdout != want {
				t.Errorf("help snapshot mismatch.\n--- got ---\n%s\n--- want ---\n%s", stdout, want)
			}
		})
	}
}
