package cmd

import (
	"fmt"

	"github.com/abegong/katalyst/internal/project"
	"github.com/spf13/cobra"
)

func newCollectionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "collection",
		Short: "Inspect collections declared by bases under .katalyst/bases/",
	}
	c.AddCommand(newCollectionListCmd(), newCollectionGetCmd())
	return c
}

func newCollectionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List collections: name, directory, item count, schema",
		Args:  maxArgs(0, "collection list"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			p := projectFor(cfg)
			cols := p.Collections()
			out := cmd.OutOrStdout()
			printListSectionHeader(out, "Collections", len(cols))
			for _, c := range cols {
				items, err := p.Items(c)
				if err != nil {
					return asUsageErr(err)
				}
				fmt.Fprintf(out, "- %s\n", c.Name)
				fmt.Fprintf(out, "  directory: %s\n", c.Path)
				fmt.Fprintf(out, "  items: %d\n", len(items))
				fmt.Fprintf(out, "  schema: %s\n", schemaLabel(c.Schema))
			}
			return nil
		},
	}
}

func newCollectionGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <collection>",
		Short: "Show one collection's detail",
		Args:  exactArgs(1, "collection get <collection>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			p := projectFor(cfg)

			sel, err := project.ParseSelector(args[0])
			if err != nil {
				return asUsageErr(err)
			}
			if sel.IsItem() {
				return usageErr(fmt.Sprintf("expected <collection>, got item selector %q", args[0]))
			}
			c, ok := p.Collection(sel.Collection)
			if !ok {
				return unknownCollectionErr(sel.Collection)
			}
			items, err := p.Items(c)
			if err != nil {
				return asUsageErr(err)
			}

			out := cmd.OutOrStdout()
			printSectionHeader(out, "Collection "+c.Name)
			fmt.Fprintf(out, "- path: %s\n", c.Path)
			fmt.Fprintf(out, "- pattern: %s\n", c.Pattern)
			fmt.Fprintf(out, "- schema: %s\n", schemaLabel(c.Schema))
			fmt.Fprintf(out, "- items: %d\n", len(items))
			fmt.Fprintf(out, "- checks: %s\n", joinOrDash(checkTypes(c)))
			return nil
		},
	}
}

func schemaLabel(name string) string {
	if name == "" {
		return "(none)"
	}
	return name
}

func checkTypes(c project.Collection) []string {
	types := make([]string, 0, len(c.Checks))
	for _, cc := range c.Checks {
		types = append(types, string(cc.Kind))
	}
	return types
}
