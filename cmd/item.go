package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/katabase-ai/katalyst/internal/frontmatter"
	"github.com/katabase-ai/katalyst/internal/project"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newItemCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "item",
		Short: "List, inspect, and mutate items within collections.",
	}
	c.AddCommand(
		newItemListCmd(),
		newItemGetCmd(),
		newItemAddCmd(),
		newItemUpdateCmd(),
		newItemDeleteCmd(),
	)
	return c
}

// itemSelector parses an arg that must be a <collection>/<item> selector.
func itemSelector(arg string) (project.Selector, error) {
	sel, err := project.ParseSelector(arg)
	if err != nil {
		return project.Selector{}, asUsageErr(err)
	}
	if !sel.IsItem() {
		return project.Selector{}, usageErr(fmt.Sprintf("expected <collection>/<item>, got %q", arg))
	}
	return sel, nil
}

func newItemListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <collection>",
		Short: "List items in a collection with their check status.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := newEngine("")
			if err != nil {
				return err
			}
			sel, err := project.ParseSelector(args[0])
			if err != nil {
				return asUsageErr(err)
			}
			if sel.IsItem() {
				return usageErr(fmt.Sprintf("item list expects <collection>, got item selector %q", args[0]))
			}
			c, ok := e.proj.Collection(sel.Collection)
			if !ok {
				return usageErr(fmt.Sprintf("unknown collection %q", sel.Collection))
			}
			items, err := e.proj.Items(c)
			if err != nil {
				return asUsageErr(err)
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, item := range items {
				n, err := itemStatus(e, c, item)
				status := statusLabel(n)
				if err != nil {
					status = "error: " + err.Error()
				}
				fmt.Fprintf(tw, "%s\t%s\n", item.ID, status)
			}
			return tw.Flush()
		},
	}
}

func newItemGetCmd() *cobra.Command {
	var frontmatterOnly, bodyOnly bool

	c := &cobra.Command{
		Use:   "get <collection>/<item>",
		Short: "Print an item (frontmatter and body by default).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if frontmatterOnly && bodyOnly {
				return usageErr("--frontmatter and --body are mutually exclusive")
			}
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			p := projectFor(cfg)
			sel, err := itemSelector(args[0])
			if err != nil {
				return err
			}
			item, err := p.ItemAt(sel.Collection, sel.Item)
			if err != nil {
				return asUsageErr(err)
			}

			out := cmd.OutOrStdout()
			switch {
			case frontmatterOnly:
				doc, err := frontmatter.Parse(mustRead(item.Path))
				if err != nil {
					return err
				}
				b, err := yaml.Marshal(doc.Meta)
				if err != nil {
					return err
				}
				_, err = out.Write(b)
				return err
			case bodyOnly:
				doc, err := frontmatter.Parse(mustRead(item.Path))
				if err != nil {
					return err
				}
				_, err = out.Write(doc.Body)
				return err
			default:
				_, err := out.Write(mustRead(item.Path))
				return err
			}
		},
	}

	c.Flags().BoolVar(&frontmatterOnly, "frontmatter", false, "Print only the parsed frontmatter")
	c.Flags().BoolVar(&bodyOnly, "body", false, "Print only the body")
	return c
}

func newItemAddCmd() *cobra.Command {
	var noValidate bool
	var schemaFlag string

	c := &cobra.Command{
		Use:   "add <collection>/<item> [key=value ...]",
		Short: "Create a new item with the given frontmatter and an empty body.",
		Long: `add writes a new item file with YAML frontmatter and an empty body.

Assignments are YAML-decoded, so numbers/booleans/arrays are supported:
  katalyst item add notes/dune title=Dune year=1965 tags='[sci-fi,classic]'

The result is validated before writing (use --no-validate to bypass).`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := newEngine(schemaFlag)
			if err != nil {
				return err
			}
			sel, err := itemSelector(args[0])
			if err != nil {
				return err
			}
			c, ok := e.proj.Collection(sel.Collection)
			if !ok {
				return usageErr(fmt.Sprintf("unknown collection %q", sel.Collection))
			}
			path := project.ItemPath(c, sel.Item)
			if _, err := os.Stat(path); err == nil {
				return usageErr(fmt.Sprintf("%s/%s already exists; refusing to overwrite", c.Name, sel.Item))
			}

			meta := map[string]any{}
			for _, a := range args[1:] {
				k, v, err := parseAssignment(a)
				if err != nil {
					return usageErr(err.Error())
				}
				meta[k] = v
			}

			out, err := composeMarkdown(meta, nil)
			if err != nil {
				return err
			}

			if !noValidate {
				if err := validateItemWrite(e, c, path, out); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
					return &exitError{code: exitValidationFail, msg: err.Error()}
				}
			}

			if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
				return err
			}
			if err := writeFileAtomic(path, out, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s/%s\n", c.Name, sel.Item)
			return nil
		},
	}

	c.Flags().BoolVar(&noValidate, "no-validate", false, "Skip schema validation before writing")
	c.Flags().StringVarP(&schemaFlag, "schema", "s", "", "Path to a JSON Schema file. Overrides config-based schema resolution")
	return c
}

func newItemUpdateCmd() *cobra.Command {
	var noValidate bool
	var schemaFlag string

	c := &cobra.Command{
		Use:   "update <collection>/<item> key=value [key=value...]",
		Short: "Set/merge frontmatter keys into an existing item; body untouched.",
		Long: `update merges top-level frontmatter keys into an existing item.

Values are YAML-decoded, so numbers/booleans/arrays are supported:
  katalyst item update notes/dune year=1965 published=true

The resulting document is validated before writing (use --no-validate to
bypass). Key removal (--unset) is out of scope for v0.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			e, err := newEngine(schemaFlag)
			if err != nil {
				return err
			}
			sel, err := itemSelector(args[0])
			if err != nil {
				return err
			}
			item, err := e.proj.ItemAt(sel.Collection, sel.Item)
			if err != nil {
				return asUsageErr(err)
			}
			c := item.Collection

			doc, err := parseItem(item.Path)
			if err != nil {
				return err
			}
			if !doc.HasFrontmatter {
				return usageErr(fmt.Sprintf("%s: no frontmatter found", item.Path))
			}

			meta := make(map[string]any, len(doc.Meta))
			for k, v := range doc.Meta {
				meta[k] = v
			}
			for _, a := range args[1:] {
				k, v, err := parseAssignment(a)
				if err != nil {
					return usageErr(err.Error())
				}
				meta[k] = v
			}

			out, err := composeMarkdown(meta, doc.Body)
			if err != nil {
				return err
			}

			if !noValidate {
				if err := validateItemWrite(e, c, item.Path, out); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), err.Error())
					return &exitError{code: exitValidationFail, msg: err.Error()}
				}
			}

			info, err := os.Stat(item.Path)
			if err != nil {
				return err
			}
			return writeFileAtomic(item.Path, out, info.Mode().Perm())
		},
	}

	c.Flags().BoolVar(&noValidate, "no-validate", false, "Skip schema validation before writing")
	c.Flags().StringVarP(&schemaFlag, "schema", "s", "", "Path to a JSON Schema file. Overrides config-based schema resolution")
	return c
}

func newItemDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <collection>/<item> [<collection>/<item> ...]",
		Short: "Delete one or more items.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfigFromCWD()
			if err != nil {
				return err
			}
			p := projectFor(cfg)

			// Resolve all targets first so a missing item aborts before we
			// delete anything.
			items := make([]project.Item, 0, len(args))
			for _, arg := range args {
				sel, err := itemSelector(arg)
				if err != nil {
					return err
				}
				item, err := p.ItemAt(sel.Collection, sel.Item)
				if err != nil {
					return asUsageErr(err)
				}
				items = append(items, item)
			}

			for _, item := range items {
				if err := os.Remove(item.Path); err != nil {
					return usageErr(fmt.Sprintf("delete: %v", err))
				}
			}
			return nil
		},
	}
}

func statusLabel(n int) string {
	switch n {
	case 0:
		return "ok"
	case 1:
		return "1 error"
	default:
		return fmt.Sprintf("%d errors", n)
	}
}

func mustRead(path string) []byte {
	b, _ := os.ReadFile(path)
	return b
}
