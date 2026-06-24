package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/inspect"
	"github.com/spf13/cobra"
)

// newInspectorsCmd builds the `inspectors` resource noun: a read-only view of
// the inspector registry (inspect.Descriptors() / inspect.Layers()), the same
// catalog cmd/gendocs renders. As a resource noun (see cmd/AGENTS.md) it carries
// CRUD-shaped sub-verbs (list, show) rather than acting when invoked bare, so it
// matches the check-types/collection/item/schema nouns. It loads no project, so
// its sub-verbs run in any directory. The descriptive dual of `check-types`.
func newInspectorsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "inspectors",
		Short: "Inspect the inspectors katalyst can run, grouped by layer.",
		Long: `inspectors is a read-only view of katalyst's inspector registry, the same
catalog cmd/gendocs renders and that the inspect command runs. List every
inspector grouped by layer (raw-source, collection), or show one inspector's
docs-style readout. It reads no project, so it runs in any directory.`,
	}
	c.AddCommand(newInspectorsListCmd(), newInspectorsShowCmd())
	return c
}

func newInspectorsListCmd() *cobra.Command {
	var asJSON bool
	var layer string
	c := &cobra.Command{
		Use:   "list",
		Short: "List inspectors grouped by layer.",
		Long: `list prints the catalog of inspectors from the inspector registry,
grouped by layer. Narrow to one layer with --layer; --json emits
machine-readable descriptors.`,
		Args: maxArgs(0, "inspectors list"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspectorsList(cmd, layer, asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Emit machine-readable JSON.")
	c.Flags().StringVar(&layer, "layer", "", "Limit the list to one layer (source, collection).")
	return c
}

func newInspectorsShowCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "show <inspector>",
		Short: "Show one inspector's layer context, purpose, and siblings.",
		Long: `show prints a detailed, docs-style readout for one inspector: its
layer context, purpose, and the other inspectors in its layer. --json emits
the machine-readable descriptor.`,
		Args: exactArgs(1, "inspectors show <inspector>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspectorsDetail(cmd, args[0], asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Emit machine-readable JSON.")
	return c
}

func runInspectorsList(cmd *cobra.Command, layer string, asJSON bool) error {
	descriptors := inspect.Descriptors()
	layers := inspect.Layers()
	if layer != "" {
		l, ok := findInspectorLayer(layer)
		if !ok {
			return usageErr(fmt.Sprintf("--layer: must be one of %s (got %q)",
				strings.Join(inspectorLayerIDs(), ", "), layer))
		}
		layers = []inspect.Layer{l}
		descriptors = inspectorLayerDescriptors(l.ID)
	}

	if asJSON {
		return writeInspectorsJSON(cmd, descriptors)
	}

	byLayer := map[string][]inspect.Descriptor{}
	for _, d := range descriptors {
		byLayer[d.Layer] = append(byLayer[d.Layer], d)
	}

	out := cmd.OutOrStdout()
	for i, l := range layers {
		if i > 0 {
			fmt.Fprintln(out)
		}
		ds := byLayer[l.ID]
		header := fmt.Sprintf("%s (%d)", l.Title, len(ds))
		fmt.Fprintln(out, header)
		fmt.Fprintln(out, strings.Repeat("-", len(header)))
		for _, d := range ds {
			fmt.Fprintf(out, "- %s\n", d.Name)
			fmt.Fprintf(out, "  %s\n", plainSummary(d.Summary))
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

	l, _ := findInspectorLayer(d.Layer)
	out := cmd.OutOrStdout()
	// Breadcrumb header, echoing how the docs nest layer → inspector page.
	fmt.Fprintf(out, "%s › %s\n\n", l.Title, d.Title)
	fmt.Fprintf(out, "inspector: %s\n", d.Name)
	fmt.Fprintf(out, "layer:     %s\n", d.Layer)
	fmt.Fprintf(out, "family:    %s\n", d.Family)
	fmt.Fprintf(out, "purpose:   %s\n", plainSummary(d.Summary))
	fmt.Fprintf(out, "\n%s\n", l.Intro)

	if siblings := inspectorLayerSiblings(d); len(siblings) > 0 {
		fmt.Fprintf(out, "\nother %s:\n  %s\n", strings.ToLower(l.Title), strings.Join(siblings, ", "))
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

// findInspectorLayer returns the inspector layer with the given id.
func findInspectorLayer(id string) (inspect.Layer, bool) {
	for _, l := range inspect.Layers() {
		if l.ID == id {
			return l, true
		}
	}
	return inspect.Layer{}, false
}

// inspectorLayerIDs returns the layer ids in display order, for error messages.
func inspectorLayerIDs() []string {
	layers := inspect.Layers()
	ids := make([]string, len(layers))
	for i, l := range layers {
		ids[i] = l.ID
	}
	return ids
}

// inspectorLayerDescriptors returns the descriptors in one layer, in registry order.
func inspectorLayerDescriptors(id string) []inspect.Descriptor {
	var out []inspect.Descriptor
	for _, d := range inspect.Descriptors() {
		if d.Layer == id {
			out = append(out, d)
		}
	}
	return out
}

// inspectorLayerSiblings returns the other inspectors in d's layer, in registry
// order.
func inspectorLayerSiblings(d inspect.Descriptor) []string {
	var out []string
	for _, o := range inspect.Descriptors() {
		if o.Layer == d.Layer && o.Name != d.Name {
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
