package cmd_test

import (
	"strings"
	"testing"
)

func TestSkillsList(t *testing.T) {
	stdout, stderr, err := runRoot(t, "skills", "list")
	if err != nil {
		t.Fatalf("skills list: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got:\n%s", stderr)
	}

	for _, want := range []string{
		"Agent skills (7)",
		"- katalyst-overview",
		"- katalyst-catalog",
		"- katalyst-identify-collections",
		"- katalyst-define-schemas",
		"- katalyst-deploy",
		"- katalyst-deploy-precommit-hook",
		"- katalyst-deploy-cli-gating",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("skills list missing %q:\n%s", want, stdout)
		}
	}
}
