package cmd_test

import (
	"errors"
	"strings"
	"testing"
)

// exitCode extracts the process exit code an error carries, or -1.
func exitCode(err error) int {
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		return coded.Code()
	}
	return -1
}

func TestArity_missingArgGivesUsageHintExit2(t *testing.T) {
	// schema get needs a name; the message should be the standard arity
	// grammar with a usage hint, not Cobra's "accepts 1 arg(s)".
	_, _, err := runRoot(t, "schema", "get")
	if got := exitCode(err); got != 2 {
		t.Fatalf("exit code = %d, want 2 (err: %v)", got, err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "usage: katalyst schema get <name>") {
		t.Errorf("missing usage hint: %q", msg)
	}
	if strings.Contains(msg, "arg(s)") {
		t.Errorf("leaked Cobra arity text: %q", msg)
	}
}

func TestArity_noArgCommandsRejectExtraArgsExit2(t *testing.T) {
	tests := [][]string{
		{"schema", "list", "unexpected"},
		{"collection", "list", "unexpected"},
	}
	for _, args := range tests {
		args := args
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, _, err := runRoot(t, args...)
			if got := exitCode(err); got != 2 {
				t.Fatalf("exit code = %d, want 2 (err: %v)", got, err)
			}
			if !strings.Contains(err.Error(), "too many arguments") {
				t.Errorf("unexpected message: %q", err.Error())
			}
		})
	}
}

func TestArity_tooManyArgsExit2(t *testing.T) {
	_, _, err := runRoot(t, "inspect", "a", "b")
	if got := exitCode(err); got != 2 {
		t.Fatalf("exit code = %d, want 2 (err: %v)", got, err)
	}
	if !strings.Contains(err.Error(), "too many arguments") {
		t.Errorf("unexpected message: %q", err.Error())
	}
}

func TestUnknownFlag_isUsageErrorExit2(t *testing.T) {
	// Cobra's flag-parse errors default to exit 1; the root FlagErrorFunc
	// routes them through usageErr (exit 2).
	_, _, err := runRoot(t, "inspect", "--nope", ".")
	if got := exitCode(err); got != 2 {
		t.Fatalf("exit code = %d, want 2 (err: %v)", got, err)
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("unexpected message: %q", err.Error())
	}
}
