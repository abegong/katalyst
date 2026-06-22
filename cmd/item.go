package cmd

import (
	"fmt"
	"os"
	"regexp"
	"text/tabwriter"

	"github.com/abegong/katalyst/internal/config"
	"github.com/abegong/katalyst/internal/frontmatter"
	"github.com/abegong/katalyst/internal/project"
	"github.com/abegong/katalyst/internal/query"
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
	var filters, greps, sorts []string
	var grepIn string
	var ignoreCase bool
	var skip, limit int
	var typeMismatch, sortMissing string

	c := &cobra.Command{
		Use:   "list <collection>",
		Short: "List items in a collection with their check status.",
		Long: `List items in a collection with their check status.

Narrow, search, and order the result (MongoDB find-inspired):
  --filter 'year>=1965'   field predicate (= != > >= < <= =~; comma RHS = in;
                          bare field = exists, !field = absent). Repeatable, ANDed.
  --grep TODO             regexp search; --grep-in all|body|frontmatter; -i.
  --sort -year,title      sort keys (id, status, or a field); leading - is desc.
  --skip N / --limit N    pagination, applied after sort.`,
		Args: exactArgs(1, "item list <collection>"),
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
				return usageErr(fmt.Sprintf("expected <collection>, got item selector %q", args[0]))
			}
			col, ok := e.proj.Collection(sel.Collection)
			if !ok {
				return unknownCollectionErr(sel.Collection)
			}
			items, err := e.proj.Items(col)
			if err != nil {
				return asUsageErr(err)
			}

			opts, err := buildQueryOptions(col, queryFlags{
				filters: filters, greps: greps, sorts: sorts,
				grepIn: grepIn, ignoreCase: ignoreCase,
				skip: skip, limit: limit,
				typeMismatch: typeMismatch, sortMissing: sortMissing,
			})
			if err != nil {
				return err
			}

			records := make([]query.Record, 0, len(items))
			statuses := make(map[string]string, len(items))
			for _, item := range items {
				rec, label := itemRecord(e, col, item)
				records = append(records, rec)
				statuses[item.ID] = label
			}

			out, err := query.Apply(records, opts)
			if err != nil {
				// The only error Apply returns is a filter type mismatch,
				// which the spec treats as a usage error (exit 2).
				return usageErr(err.Error())
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, rec := range out {
				fmt.Fprintf(tw, "%s\t%s\n", rec.ID, statuses[rec.ID])
			}
			return tw.Flush()
		},
	}

	c.Flags().StringArrayVar(&filters, "filter", nil, "Keep items matching a field predicate (repeatable, ANDed)")
	c.Flags().StringArrayVar(&greps, "grep", nil, "Keep items whose text matches a regexp (repeatable, ANDed)")
	c.Flags().StringVar(&grepIn, "grep-in", "all", "Region --grep searches: all, body, or frontmatter")
	c.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "Make --grep patterns case-insensitive")
	c.Flags().StringArrayVar(&sorts, "sort", nil, "Sort by key(s); leading - is descending (e.g. -year,title)")
	c.Flags().IntVar(&skip, "skip", 0, "Drop the first N results (after sorting)")
	c.Flags().IntVar(&limit, "limit", 0, "Keep at most N results (0 = no cap)")
	c.Flags().StringVar(&typeMismatch, "on-type-mismatch", "", "Override filterTypeMismatch: skip or error")
	c.Flags().StringVar(&sortMissing, "sort-missing", "", "Override sortMissing: last or lowest")
	return c
}

// queryFlags collects the raw --filter/--grep/--sort flag values for the
// item list query.
type queryFlags struct {
	filters, greps, sorts     []string
	grepIn                    string
	ignoreCase                bool
	skip, limit               int
	typeMismatch, sortMissing string
}

// buildQueryOptions parses and validates the query flags into a
// query.Options, resolving the configurable defaults flag-over-collection.
// Any parse or validation failure is a usage error (exit 2).
func buildQueryOptions(col config.Collection, f queryFlags) (query.Options, error) {
	opts := query.Options{}

	for _, expr := range f.filters {
		p, err := query.ParseFilter(expr)
		if err != nil {
			return query.Options{}, usageErr(err.Error())
		}
		opts.Filters = append(opts.Filters, p)
	}

	for _, pat := range f.greps {
		if f.ignoreCase {
			pat = "(?i)" + pat
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return query.Options{}, usageErr(fmt.Sprintf("--grep: %v", err))
		}
		opts.Greps = append(opts.Greps, re)
	}

	switch f.grepIn {
	case "", "all":
		opts.GrepIn = query.RegionAll
	case "body":
		opts.GrepIn = query.RegionBody
	case "frontmatter":
		opts.GrepIn = query.RegionFrontmatter
	default:
		return query.Options{}, usageErr(fmt.Sprintf("--grep-in: must be all, body, or frontmatter (got %q)", f.grepIn))
	}

	for _, spec := range f.sorts {
		keys, err := query.ParseSort(spec)
		if err != nil {
			return query.Options{}, usageErr(err.Error())
		}
		opts.Sorts = append(opts.Sorts, keys...)
	}

	if f.skip < 0 {
		return query.Options{}, usageErr("--skip: must not be negative")
	}
	if f.limit < 0 {
		return query.Options{}, usageErr("--limit: must not be negative")
	}
	opts.Skip = f.skip
	opts.Limit = f.limit

	opts.TypeMismatch = col.Query.FilterTypeMismatch
	if f.typeMismatch != "" {
		if f.typeMismatch != "skip" && f.typeMismatch != "error" {
			return query.Options{}, usageErr(fmt.Sprintf("--on-type-mismatch: must be skip or error (got %q)", f.typeMismatch))
		}
		opts.TypeMismatch = f.typeMismatch
	}

	opts.SortMissing = col.Query.SortMissing
	if f.sortMissing != "" {
		if f.sortMissing != "last" && f.sortMissing != "lowest" {
			return query.Options{}, usageErr(fmt.Sprintf("--sort-missing: must be last or lowest (got %q)", f.sortMissing))
		}
		opts.SortMissing = f.sortMissing
	}

	return opts, nil
}

// itemRecord assembles a query.Record for one item and its display status
// label. A parse error still yields a record (raw bytes for --grep, empty
// Meta) so the listing stays robust; the label reports the error.
func itemRecord(e *engine, col config.Collection, item project.Item) (query.Record, string) {
	raw := mustRead(item.Path)
	rec := query.Record{ID: item.ID, Raw: raw, Body: raw}

	if doc, err := parseItem(item.Path); err == nil && doc != nil {
		rec.Meta = doc.Meta
		rec.Body = doc.Body
		rec.Frontmatter = doc.Frontmatter
	}

	n, err := itemStatus(e, col, item)
	if err != nil {
		// Sort errored items after clean ones; surface the error in the label.
		rec.Status = 1 << 30
		return rec, "error: " + err.Error()
	}
	rec.Status = n
	return rec, statusLabel(n)
}

func newItemGetCmd() *cobra.Command {
	var frontmatterOnly, bodyOnly bool

	c := &cobra.Command{
		Use:   "get <collection>/<item>",
		Short: "Print an item (frontmatter and body by default).",
		Args:  exactArgs(1, "item get <collection>/<item>"),
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
		Args: minArgs(1, "item add <collection>/<item> [key=value ...]"),
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
				return unknownCollectionErr(sel.Collection)
			}
			path := project.ItemPath(c, sel.Item)
			if _, err := os.Stat(path); err == nil {
				return usageErr(fmt.Sprintf("%q already exists; refusing to overwrite", c.Name+"/"+sel.Item))
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
		Args: minArgs(2, "item update <collection>/<item> key=value [key=value ...]"),
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
		Args:  minArgs(1, "item delete <collection>/<item> ..."),
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
					return usageErr(fmt.Sprintf("delete %s: %v", item.Path, err))
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
