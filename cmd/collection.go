package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/katabase-ai/katalyst/internal/config"
	"github.com/katabase-ai/katalyst/internal/project"
	"github.com/spf13/cobra"
)

func newCollectionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "collection",
		Short: "Inspect collections defined under .katalyst/collections/.",
	}
	c.AddCommand(newCollectionListCmd(), newCollectionGetCmd())
	return c
}

func newCollectionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List collections: name, directory, item count, schema.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			p := projectFor(cfg)

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tDIRECTORY\tITEMS\tSCHEMA")
			for _, c := range p.Collections() {
				items, err := p.Items(c)
				if err != nil {
					return asUsageErr(err)
				}
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", c.Name, c.Path, len(items), schemaLabel(c.Schema))
			}
			return tw.Flush()
		},
	}
}

func newCollectionGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <collection>",
		Short: "Show one collection's detail.",
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
			fmt.Fprintf(out, "name:    %s\n", c.Name)
			fmt.Fprintf(out, "path:    %s\n", c.Path)
			fmt.Fprintf(out, "pattern: %s\n", c.Pattern)
			fmt.Fprintf(out, "schema:  %s\n", schemaLabel(c.Schema))
			fmt.Fprintf(out, "items:   %d\n", len(items))
			fmt.Fprintf(out, "checks:  %s\n", strings.Join(checkTypes(c), ", "))
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

func checkTypes(c config.Collection) []string {
	types := make([]string, 0, len(c.Checks))
	for _, ch := range c.Checks {
		types = append(types, string(ch.Type))
	}
	return types
}
