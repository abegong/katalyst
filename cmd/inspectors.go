package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/abegong/katalyst/internal/inspect"
	"github.com/spf13/cobra"
)

// newInspectorsCmd builds the `inspectors` resource noun: a read-only view of
// the inspector registry (inspect.Descriptors() / inspect.Families()) — the same
// catalog cmd/gendocs renders. As a resource noun (see cmd/AGENTS.md) it carries
// CRUD-shaped sub-verbs (list, show) rather than acting when invoked bare, so it
// matches the check-types/collection/item/schema nouns. It loads no project, so
// its sub-verbs run in any directory. The descriptive dual of `check-types`.
func newInspectorsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "inspectors",
		Short: "Inspect the inspectors the engine can run, grouped by family.",
		Long: `inspectors is a read-only view of the engine's inspector registry — the same
catalog cmd/gendocs renders and that the inspect command runs. List every
inspector grouped by family, or show one inspector's docs-style readout. It
reads no project, so it runs in any directory.`,
	}
	c.AddCommand(newInspectorsListCmd(), newInspectorsShowCmd())
	return c
}

func newInspectorsListCmd() *cobra.Command {
	var asJSON bool
	var family string
	c := &cobra.Command{
		Use:   "list",
		Short: "List inspectors grouped by family.",
		Long: `list prints the catalog of inspectors from the engine registry,
grouped by family. Narrow to one family with --family; --json emits
machine-readable descriptors.`,
		Args: maxArgs(0, "inspectors list"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspectorsList(cmd, family, asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Emit machine-readable JSON.")
	c.Flags().StringVar(&family, "family", "", "Limit the list to one family (structural, object, markdown, filesystem).")
	return c
}

func newInspectorsShowCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "show <inspector>",
		Short: "Show one inspector's family context, purpose, and siblings.",
		Long: `show prints a detailed, docs-style readout for one inspector: its
family context, purpose, and the other inspectors in its family. --json emits
the machine-readable descriptor.`,
		Args: exactArgs(1, "inspectors show <inspector>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspectorsDetail(cmd, args[0], asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Emit machine-readable JSON.")
	return c
}

func runInspectorsList(cmd *cobra.Command, family string, asJSON bool) error {
	descriptors := inspect.Descriptors()
	families := inspect.Families()
	if family != "" {
		fam, ok := findInspectorFamily(family)
		if !ok {
			return usageErr(fmt.Sprintf("--family: must be one of %s (got %q)",
				strings.Join(inspectorFamilyIDs(), ", "), family))
		}
		families = []inspect.Family{fam}
		descriptors = inspectorFamilyDescriptors(fam.ID)
	}

	if asJSON {
		return writeInspectorsJSON(cmd, descriptors)
	}

	byFamily := map[string][]inspect.Descriptor{}
	for _, d := range descriptors {
		byFamily[d.Family] = append(byFamily[d.Family], d)
	}

	out := cmd.OutOrStdout()
	for i, fam := range families {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, fam.Title)
		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "INSPECTOR\tPURPOSE")
		for _, d := range byFamily[fam.ID] {
			fmt.Fprintf(tw, "%s\t%s\n", d.Name, plainSummary(d.Summary))
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func runInspectorsDetail(cmd *cobra.Command, name string, asJSON bool) error {
	d, ok := findInspectorDescriptor(name)
	if !ok {
		return usageErr(fmt.Sprintf("unknown inspector %q (try `katalyst inspectors list`)", name))
	}
	if asJSON {
		return writeInspectorsJSON(cmd, d)
	}

	fam, _ := findInspectorFamily(d.Family)
	out := cmd.OutOrStdout()
	// Breadcrumb header, echoing how the docs nest family → inspector page.
	fmt.Fprintf(out, "%s › %s\n\n", fam.Title, d.Title)
	fmt.Fprintf(out, "inspector: %s\n", d.Name)
	fmt.Fprintf(out, "family:    %s\n", d.Family)
	fmt.Fprintf(out, "purpose:   %s\n", plainSummary(d.Summary))
	fmt.Fprintf(out, "\n%s\n", fam.Intro)

	// Inspectors take no configuration; they emit evidence over a corpus. Run
	// one with `katalyst inspect <path> --inspector %s` to see its evidence.
	fmt.Fprintf(out, "\nInspectors take no configuration. Run this one over a corpus to see its\nevidence:\n  katalyst inspect <path> --inspector %s\n", d.Name)

	if siblings := inspectorFamilySiblings(d); len(siblings) > 0 {
		fmt.Fprintf(out, "\nother %s inspectors:\n  %s\n", strings.ToLower(fam.Title), strings.Join(siblings, ", "))
	}
	return nil
}

// findInspectorDescriptor returns the descriptor whose Name equals name.
func findInspectorDescriptor(name string) (inspect.Descriptor, bool) {
	for _, d := range inspect.Descriptors() {
		if d.Name == name {
			return d, true
		}
	}
	return inspect.Descriptor{}, false
}

// findInspectorFamily returns the inspector family with the given id.
func findInspectorFamily(id string) (inspect.Family, bool) {
	for _, f := range inspect.Families() {
		if f.ID == id {
			return f, true
		}
	}
	return inspect.Family{}, false
}

// inspectorFamilyIDs returns the family ids in display order, for error messages.
func inspectorFamilyIDs() []string {
	fams := inspect.Families()
	ids := make([]string, len(fams))
	for i, f := range fams {
		ids[i] = f.ID
	}
	return ids
}

// inspectorFamilyDescriptors returns the descriptors in one family, in registry order.
func inspectorFamilyDescriptors(id string) []inspect.Descriptor {
	var out []inspect.Descriptor
	for _, d := range inspect.Descriptors() {
		if d.Family == id {
			out = append(out, d)
		}
	}
	return out
}

// inspectorFamilySiblings returns the other inspectors in d's family, in
// registry order.
func inspectorFamilySiblings(d inspect.Descriptor) []string {
	var out []string
	for _, o := range inspect.Descriptors() {
		if o.Family == d.Family && o.Name != d.Name {
			out = append(out, o.Name)
		}
	}
	return out
}

func writeInspectorsJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
