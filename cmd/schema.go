package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abegong/katalyst/internal/config"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	s := &cobra.Command{
		Use:   "schema",
		Short: "Inspect schemas defined under .katalyst/schemas/.",
	}
	s.AddCommand(newSchemaListCmd(), newSchemaShowCmd())
	return s
}

func newSchemaListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List schemas registered in the config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, name := range cfg.SchemaNames() {
				rel, _ := filepath.Rel(cfg.Root, cfg.SchemaPath(name))
				if rel == "" {
					rel = cfg.SchemaPath(name)
				}
				fmt.Fprintf(out, "%s\t%s\n", name, rel)
			}
			return nil
		},
	}
}

// TODO: align the read verb with the other resource nouns, collection/item
// use `get`, schema uses `show`. Per the command-grammar work, pick one word
// for "read one" (likely `get`) and converge. See
// cmd/organization.md.
func newSchemaShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Print the contents of a registered schema (pretty-printed JSON).",
		Args:  exactArgs(1, "schema show <name>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			path := cfg.SchemaPath(name)
			if path == "" {
				return usageErr(fmt.Sprintf("unknown schema %q (try `katalyst schema list`)", name))
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}

			var pretty bytes.Buffer
			if err := json.Indent(&pretty, src, "", "  "); err != nil {
				// Fall back to the raw bytes if it isn't valid JSON;
				// `schema list` callers still benefit from seeing what's
				// actually in the file.
				cmd.OutOrStdout().Write(src)
				return nil
			}
			cmd.OutOrStdout().Write(pretty.Bytes())
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}
}

// loadConfigFromCWD finds the config relative to the current working
// directory, converting "not found" into a usage error so the CLI exits
// with code 2.
func loadConfigFromCWD() (*config.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(wd)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			return nil, usageErr("no .katalyst/ found in this directory or any ancestor (run `katalyst init`)")
		}
		return nil, err
	}
	return cfg, nil
}
