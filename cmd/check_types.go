package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
	"github.com/spf13/cobra"
)

// newCheckTypesCmd builds the `check-types` resource noun: a read-only view of
// the check registry (checks.Descriptors() / checks.Families()), the same
// catalog cmd/gendocs renders. As a resource noun (see cmd/AGENTS.md) it carries
// CRUD-shaped sub-verbs (list, show) rather than acting when invoked bare, so
// it matches the collection/item/schema nouns. It loads no project, so its
// sub-verbs run in any directory. `rules` is kept as a deprecated alias.
func newCheckTypesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "check-types",
		Aliases: []string{"rules"},
		Short:   "Inspect the check types katalyst can enforce, grouped by family",
		Long: `check-types is a read-only view of katalyst's check registry, the same
catalog cmd/gendocs renders. List every check type grouped by family, or show one
check type's docs-style readout. It reads no project, so it runs in any directory.`,
	}
	c.AddCommand(newCheckTypesListCmd(), newCheckTypesShowCmd())
	return c
}

func newCheckTypesListCmd() *cobra.Command {
	var asJSON bool
	var family string
	c := &cobra.Command{
		Use:   "list",
		Short: "List check types grouped by family",
		Long: `list prints the catalog of check types from the check registry,
grouped by family. Narrow to one family with --family; --json emits
machine-readable descriptors.`,
		Args: maxArgs(0, "check-types list"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckTypesList(cmd, family, asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Emit machine-readable JSON.")
	c.Flags().StringVar(&family, "family", "", "Limit the list to one family (structuredObject, markdownBodyText, fileSystem, plainText).")
	return c
}

func newCheckTypesShowCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "show <check-type>",
		Short: "Show one check type's keys, example, and siblings",
		Long: `show prints a detailed, docs-style readout for one check type: its
family context, purpose, configuration keys, example, and the other
check types in its family. --json emits the machine-readable descriptor.`,
		Args: exactArgs(1, "check-types show <check-type>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckTypesDetail(cmd, args[0], asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Emit machine-readable JSON.")
	return c
}

func runCheckTypesList(cmd *cobra.Command, family string, asJSON bool) error {
	descriptors := checks.Descriptors()
	families := checks.Families()
	if family != "" {
		fam, ok := findFamily(family)
		if !ok {
			return usageErr(fmt.Sprintf("--family: must be one of %s (got %q)",
				strings.Join(familyIDs(), ", "), family))
		}
		families = []checks.Family{fam}
		descriptors = familyDescriptors(fam.ID)
	}

	if asJSON {
		return writeCheckTypesJSON(cmd, jsonDescriptors(descriptors))
	}

	byFamily := map[string][]checks.Descriptor{}
	for _, d := range descriptors {
		byFamily[d.Family] = append(byFamily[d.Family], d)
	}

	out := cmd.OutOrStdout()
	for i, fam := range families {
		if i > 0 {
			fmt.Fprintln(out)
		}
		ds := byFamily[fam.ID]
		printListSectionHeader(out, fam.Title, len(ds))
		for _, d := range ds {
			req, opt := splitFields(d.Fields)
			fmt.Fprintf(out, "- %s\n", d.CheckType)
			fmt.Fprintf(out, "  purpose: %s\n", plainSummary(d.Summary))
			fmt.Fprintf(out, "  required: %s\n", req)
			fmt.Fprintf(out, "  optional: %s\n", opt)
		}
	}
	return nil
}

func runCheckTypesDetail(cmd *cobra.Command, checkType string, asJSON bool) error {
	d, ok := findDescriptor(checkType)
	if !ok {
		return usageErr(fmt.Sprintf("unknown check type %q (try `katalyst check-types`)", checkType))
	}
	if asJSON {
		return writeCheckTypesJSON(cmd, jsonDescriptor(d))
	}

	fam, _ := findFamily(d.Family)
	out := cmd.OutOrStdout()
	// Breadcrumb header, echoing how the docs nest family → check-type page.
	printSectionHeader(out, fmt.Sprintf("%s › %s", fam.Title, d.Title))
	fmt.Fprintf(out, "- kind: %s\n", d.CheckType)
	fmt.Fprintf(out, "- family: %s\n", d.Family)
	fmt.Fprintf(out, "- configurableIn: %s\n", strings.Join(checks.DescriptorConfigurableIn(d), ", "))
	scope := d.Scope
	if scope == "" {
		scope = "item"
	}
	fmt.Fprintf(out, "- scope: %s\n", scope)
	sev := d.Severity
	if sev == "" {
		sev = "error"
	}
	fmt.Fprintf(out, "- severity: %s\n", sev)
	fmt.Fprintf(out, "- purpose: %s\n", plainSummary(d.Summary))

	fmt.Fprintln(out)
	printSectionHeader(out, "Family context")
	fmt.Fprintln(out, fam.Intro)

	fmt.Fprintln(out)
	if len(d.Fields) > 0 {
		printListSectionHeader(out, "Configuration keys", len(d.Fields))
		for _, f := range d.Fields {
			req := "no"
			if f.Required {
				req = "yes"
			}
			def := f.Default
			if def == "" {
				def = "-"
			}
			fmt.Fprintf(out, "- %s\n", f.Name)
			fmt.Fprintf(out, "  required: %s\n", req)
			fmt.Fprintf(out, "  default: %s\n", def)
			fmt.Fprintf(out, "  meaning: %s\n", plainSummary(f.Desc))
		}
	} else {
		printListSectionHeader(out, "Configuration keys", 0)
		fmt.Fprintln(out, "- none")
	}

	fmt.Fprintln(out)
	printSectionHeader(out, "Example")
	fmt.Fprintln(out, indentLines(d.ConfigExample, "  "))

	if siblings := familySiblings(d); len(siblings) > 0 {
		fmt.Fprintln(out)
		printListSectionHeader(out, "Other "+strings.ToLower(fam.Title), len(siblings))
		for _, s := range siblings {
			fmt.Fprintf(out, "- %s\n", s)
		}
	}
	return nil
}

// findDescriptor returns the descriptor whose CheckType equals checkType.
func findDescriptor(checkType string) (checks.Descriptor, bool) {
	for _, d := range checks.Descriptors() {
		if string(d.CheckType) == checkType {
			return d, true
		}
	}
	return checks.Descriptor{}, false
}

// findFamily returns the family with the given id.
func findFamily(id string) (checks.Family, bool) {
	for _, f := range checks.Families() {
		if f.ID == id {
			return f, true
		}
	}
	return checks.Family{}, false
}

// familyIDs returns the family ids in display order, for error messages.
func familyIDs() []string {
	fams := checks.Families()
	ids := make([]string, len(fams))
	for i, f := range fams {
		ids[i] = f.ID
	}
	return ids
}

// familyDescriptors returns the descriptors in one family, in registry order.
func familyDescriptors(id string) []checks.Descriptor {
	var out []checks.Descriptor
	for _, d := range checks.Descriptors() {
		if d.Family == id {
			out = append(out, d)
		}
	}
	return out
}

// familySiblings returns the other check types in d's family, in registry order.
func familySiblings(d checks.Descriptor) []string {
	var out []string
	for _, o := range checks.Descriptors() {
		if o.Family == d.Family && o.CheckType != d.CheckType {
			out = append(out, string(o.CheckType))
		}
	}
	return out
}

// splitFields partitions a check type's fields into comma-joined required and
// optional name lists, using a dash when a side is empty.
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
		return "-"
	}
	return strings.Join(names, ", ")
}

func writeCheckTypesJSON(cmd *cobra.Command, v any) error {
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
	d.ConfigurableIn = checks.DescriptorConfigurableIn(d)
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
