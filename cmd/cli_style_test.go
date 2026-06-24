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

func commandNames(commands []*cobra.Command) []string {
	names := make([]string, 0, len(commands))
	for _, command := range commands {
		names = append(names, command.Name())
	}
	return names
}
