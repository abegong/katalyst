package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/katabase-ai/katalyst/internal/checks"
	"github.com/spf13/cobra"
)

// newRulesCmd builds `katalyst rules`, a read-only view of the check registry
// (checks.Descriptors() / checks.Families()) — the same catalog cmd/gendocs
// renders. It loads no project, so it runs in any directory.
func newRulesCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "rules [kind]",
		Short: "List the check kinds the engine can enforce, grouped by family.",
		Long: `rules prints the catalog of check kinds from the engine registry:
their purpose and configuration keys. It reads no project, so it runs in
any directory. With a kind argument it prints detail for that one kind;
with --json it emits a machine-readable descriptor array (or object).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return runRulesDetail(cmd, args[0], asJSON)
			}
			return runRulesList(cmd, asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Emit machine-readable JSON.")
	return c
}

func runRulesList(cmd *cobra.Command, asJSON bool) error {
	if asJSON {
		return writeRulesJSON(cmd, jsonDescriptors(checks.Descriptors()))
	}

	byFamily := map[string][]checks.Descriptor{}
	for _, d := range checks.Descriptors() {
		byFamily[d.Family] = append(byFamily[d.Family], d)
	}

	out := cmd.OutOrStdout()
	for i, fam := range checks.Families() {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, fam.Title)
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "KIND\tPURPOSE\tREQUIRED\tOPTIONAL")
		for _, d := range byFamily[fam.ID] {
			req, opt := splitFields(d.Fields)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", d.Kind, plainSummary(d.Summary), req, opt)
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func runRulesDetail(cmd *cobra.Command, kind string, asJSON bool) error {
	d, ok := findDescriptor(kind)
	if !ok {
		return usageErr(fmt.Sprintf("unknown check kind %q", kind))
	}
	if asJSON {
		return writeRulesJSON(cmd, jsonDescriptor(d))
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "kind:    %s\n", d.Kind)
	fmt.Fprintf(out, "family:  %s\n", d.Family)
	fmt.Fprintf(out, "purpose: %s\n", plainSummary(d.Summary))
	if len(d.Fields) > 0 {
		fmt.Fprint(out, "\nconfiguration keys:\n")
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "  FIELD\tREQUIRED\tDEFAULT\tMEANING")
		for _, f := range d.Fields {
			req := "no"
			if f.Required {
				req = "yes"
			}
			def := f.Default
			if def == "" {
				def = "—"
			}
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n", f.Name, req, def, plainSummary(f.Desc))
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "\nexample:\n%s\n", indentLines(d.ConfigExample, "  "))
	return nil
}

// findDescriptor returns the descriptor whose Kind equals kind.
func findDescriptor(kind string) (checks.Descriptor, bool) {
	for _, d := range checks.Descriptors() {
		if string(d.Kind) == kind {
			return d, true
		}
	}
	return checks.Descriptor{}, false
}

// splitFields partitions a check's fields into comma-joined required and
// optional name lists, using an em dash when a side is empty.
func splitFields(fields []checks.Field) (required, optional string) {
	var req, opt []string
	for _, f := range fields {
		if f.Required {
			req = append(req, f.Name)
		} else {
			opt = append(opt, f.Name)
		}
	}
	return joinOrDash(req), joinOrDash(opt)
}

func joinOrDash(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	return strings.Join(names, ", ")
}

func writeRulesJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// jsonDescriptor normalizes a descriptor for JSON: a nil Fields slice becomes
// [] so consumers iterate without a null-check.
func jsonDescriptor(d checks.Descriptor) checks.Descriptor {
	if d.Fields == nil {
		d.Fields = []checks.Field{}
	}
	return d
}

func jsonDescriptors(ds []checks.Descriptor) []checks.Descriptor {
	out := make([]checks.Descriptor, len(ds))
	for i, d := range ds {
		out[i] = jsonDescriptor(d)
	}
	return out
}

// plainSummary strips inline-code backticks so a summary reads cleanly in a
// terminal table, mirroring what gendocs does for its link lists.
func plainSummary(s string) string {
	return strings.ReplaceAll(s, "`", "")
}

// indentLines prefixes every line of s with prefix.
func indentLines(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = prefix + ln
	}
	return strings.Join(lines, "\n")
}
