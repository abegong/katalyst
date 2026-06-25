package cmd_test

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/cmd"
	"github.com/spf13/cobra"
)

func TestCLIStyle_topLevelCommandsStayGrouped(t *testing.T) {
	root := cmd.NewRootCmd()
	for _, command := range topLevelCommands(root) {
		switch command.GroupID {
		case "verbs", "resources":
		default:
			t.Errorf("%s: top-level command must be registered in Verbs or Resources group", command.Name())
		}
	}
}

func TestCLIStyle_rootHelpOrderStaysIntentional(t *testing.T) {
	root := cmd.NewRootCmd()
	got := commandNames(topLevelCommands(root))
	want := []string{
		"inspect",
		"init",
		"check",
		"fix",
		"collection",
		"item",
		"schema",
		"skills",
		"check-types",
		"inspectors",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("top-level command order = %v, want %v", got, want)
	}
}

func TestCLIStyle_resourceNounsHaveNoDefaultAction(t *testing.T) {
	root := cmd.NewRootCmd()
	for _, command := range topLevelCommands(root) {
		if command.GroupID != "resources" {
			continue
		}
		if command.Run != nil || command.RunE != nil {
			t.Errorf("%s: resource noun must not run a default action", command.Name())
		}
	}
}

func TestCLIStyle_resourceNounsHaveListSubcommand(t *testing.T) {
	root := cmd.NewRootCmd()
	for _, command := range topLevelCommands(root) {
		if command.GroupID != "resources" {
			continue
		}
		if command.CommandPath() == "" {
			t.Fatalf("%s: unexpected empty command path", command.Name())
		}
		if _, _, err := command.Find([]string{"list"}); err != nil {
			t.Errorf("%s: resource noun must expose a list subcommand or document a reason not to", command.Name())
		}
	}
}

func TestCLIStyle_topLevelShortHelpShape(t *testing.T) {
	root := cmd.NewRootCmd()
	for _, command := range topLevelCommands(root) {
		if command.Short == "" {
			t.Errorf("%s: Short help must be set", command.Name())
			continue
		}
		if strings.HasSuffix(command.Short, ".") {
			t.Errorf("%s: Short help must not end with a period", command.Name())
		}
		switch command.GroupID {
		case "verbs":
			if strings.HasPrefix(command.Short, "Commands to ") {
				t.Errorf("%s: verb Short help must describe the action directly", command.Name())
			}
		case "resources":
			if !strings.HasPrefix(command.Short, "Commands to ") {
				t.Errorf("%s: resource noun Short help must start with %q", command.Name(), "Commands to ")
			}
		}
	}
}

func TestCLIStyle_visibleShortHelpHasNoTrailingPeriods(t *testing.T) {
	root := cmd.NewRootCmd()
	for _, command := range visibleCommands(root) {
		if command.Short == "" {
			t.Errorf("%s: Short help must be set", command.CommandPath())
			continue
		}
		if strings.HasSuffix(command.Short, ".") {
			t.Errorf("%s: Short help must not end with a period", command.CommandPath())
		}
	}
}

func TestCLIStyle_runnableCommandsDeclareArgValidation(t *testing.T) {
	root := cmd.NewRootCmd()
	for _, command := range visibleCommands(root) {
		if command.Run == nil && command.RunE == nil {
			continue
		}
		if command.Args == nil {
			t.Errorf("%s: runnable command must declare Args, even when it accepts no args", command.CommandPath())
		}
	}
}

func TestCLIStyle_noArgCommandsUseStandardArityError(t *testing.T) {
	root := cmd.NewRootCmd()
	tests := []struct {
		path  []string
		usage string
	}{
		{path: []string{"init"}, usage: "init"},
		{path: []string{"collection", "list"}, usage: "collection list"},
		{path: []string{"schema", "list"}, usage: "schema list"},
		{path: []string{"skills", "list"}, usage: "skills list"},
		{path: []string{"check-types", "list"}, usage: "check-types list"},
		{path: []string{"inspectors", "list"}, usage: "inspectors list"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(strings.Join(tc.path, " "), func(t *testing.T) {
			command := findCommand(t, root, tc.path...)
			if command.Args == nil {
				t.Fatalf("%s: Args is nil", command.CommandPath())
			}
			err := command.Args(command, []string{"unexpected"})
			if got := exitCode(err); got != 2 {
				t.Fatalf("%s: exit code = %d, want 2 (err: %v)", command.CommandPath(), got, err)
			}
			msg := err.Error()
			if !strings.Contains(msg, "too many arguments") {
				t.Errorf("%s: expected too many arguments message, got %q", command.CommandPath(), msg)
			}
			if !strings.Contains(msg, "usage: katalyst "+tc.usage) {
				t.Errorf("%s: expected usage hint for %q, got %q", command.CommandPath(), tc.usage, msg)
			}
		})
	}
}

func TestCLIStyle_topLevelHelpHasSnapshots(t *testing.T) {
	root := cmd.NewRootCmd()
	for _, command := range topLevelCommands(root) {
		name := "help/" + command.Name() + ".txt"
		if _, _, err := matchSnapshot(name, ""); err != nil {
			t.Errorf("%s: missing top-level help snapshot %s", command.Name(), name)
		}
	}
}

func topLevelCommands(root *cobra.Command) []*cobra.Command {
	var commands []*cobra.Command
	for _, command := range root.Commands() {
		if command.Hidden {
			continue
		}
		commands = append(commands, command)
	}
	return commands
}

func visibleCommands(root *cobra.Command) []*cobra.Command {
	var commands []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		if command.Hidden {
			return
		}
		commands = append(commands, command)
		for _, child := range command.Commands() {
			walk(child)
		}
	}
	walk(root)
	return commands
}

func findCommand(t *testing.T, root *cobra.Command, args ...string) *cobra.Command {
	t.Helper()
	command, _, err := root.Find(args)
	if err != nil {
		t.Fatalf("find %v: %v", args, err)
	}
	if command == nil {
		t.Fatalf("find %v: nil command", args)
	}
	return command
}

func commandNames(commands []*cobra.Command) []string {
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		names = append(names, command.Name())
	}
	return names
}
