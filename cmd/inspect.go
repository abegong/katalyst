package cmd

import (
	"fmt"
	"os"

	"github.com/abegong/katalyst/internal/config"
	"github.com/abegong/katalyst/internal/inspect"
	"github.com/abegong/katalyst/internal/project"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var (
		jsonOut    bool
		outFile    string
		inspectors []string
		maxLines   int
		verbose    bool
		detail     string
		similarity float64
		maxClasses int
	)

	c := &cobra.Command{
		Use:   "inspect <path-or-collection>",
		Short: "Profile a directory of markdown files and report its shape.",
		Long: `inspect runs inspectors over a target and reports the shape of what it
finds as evidence — counts and distributions, never recommendations.

The layer is inferred from the argument. Inside a katalyst project, a
configured collection name (e.g. notes) runs the collection inspectors over
that collection's items. Otherwise the argument is a filesystem path and the
raw-source inspectors profile the tree (the onboarding case: "what's here?").

Inspectors describe; they never recommend. inspect writes no schema and mutates
nothing. Output is Markdown by default; --json emits the same evidence as JSON.`,
		Args: exactArgs(1, "inspect <path-or-collection>"),
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := inspect.ParseParams(detail, similarity, maxClasses)
			if err != nil {
				return usageErr(err.Error())
			}

			evidence, err := runInspect(args[0], inspectors, params)
			if err != nil {
				return err
			}

			limit := maxLines
			if verbose {
				limit = 0
			}
			rendered, err := render(evidence, jsonOut, limit)
			if err != nil {
				return err
			}

			if outFile != "" {
				if err := os.WriteFile(outFile, rendered, 0o644); err != nil {
					return usageErr(fmt.Sprintf("write %s: %v", outFile, err))
				}
				return nil
			}
			_, err = cmd.OutOrStdout().Write(rendered)
			return err
		},
	}

	c.Flags().BoolVar(&jsonOut, "json", false, "Emit evidence as JSON instead of Markdown.")
	c.Flags().StringVarP(&outFile, "output", "o", "", "Write the report to a file instead of stdout.")
	c.Flags().StringArrayVar(&inspectors, "inspector", nil,
		"Run only the named inspector(s); repeatable. Default: all in the selected layer.")
	c.Flags().IntVar(&maxLines, "max-lines", 20,
		"Truncate each inspector's Markdown output to N lines (0 = no limit).")
	c.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Show full output; do not truncate (same as --max-lines 0).")
	c.Flags().StringVar(&detail, "detail", "",
		"Summarizer detail level: exact, grouped, or coarse (default grouped).")
	c.Flags().Float64Var(&similarity, "similarity", -1,
		"Summarizer similarity threshold (0–1). Mutually exclusive with --detail/--max-classes.")
	c.Flags().IntVar(&maxClasses, "max-classes", 0,
		"Cap the number of summarized classes. Mutually exclusive with --detail/--similarity.")
	return c
}

// runInspect selects the layer from the argument and runs its inspectors. A
// configured collection name runs the collection layer; anything else is a
// filesystem path for the raw-source layer.
func runInspect(arg string, names []string, params inspect.Params) ([]inspect.Evidence, error) {
	if proj, c, ok := resolveCollection(arg); ok {
		return runCollectionLayer(proj, c, names, params)
	}
	return runSourceLayer(arg, names, params)
}

// resolveCollection reports whether arg names a configured collection in a
// project rooted at or above the working directory.
func resolveCollection(arg string) (*project.Project, config.Collection, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, config.Collection{}, false
	}
	cfg, err := config.Load(wd)
	if err != nil {
		return nil, config.Collection{}, false // no project → raw path
	}
	sel, err := project.ParseSelector(arg)
	if err != nil {
		return nil, config.Collection{}, false
	}
	c, ok := cfg.Collection(sel.Collection)
	if !ok {
		return nil, config.Collection{}, false
	}
	return project.New(cfg), c, true
}

func runCollectionLayer(proj *project.Project, c config.Collection, names []string, params inspect.Params) ([]inspect.Evidence, error) {
	selected, err := selectCollectionInspectors(names)
	if err != nil {
		return nil, err
	}
	view, err := inspect.NewCollectionView(proj, c)
	if err != nil {
		return nil, usageErr(fmt.Sprintf("%s: %v", c.Name, err))
	}
	evidence := make([]inspect.Evidence, 0, len(selected))
	for _, ins := range selected {
		evidence = append(evidence, ins.Inspect(view, params))
	}
	return evidence, nil
}

func runSourceLayer(path string, names []string, params inspect.Params) ([]inspect.Evidence, error) {
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return nil, usageErr(fmt.Sprintf("%s: not a readable directory", path))
	}
	selected, err := selectSourceInspectors(names)
	if err != nil {
		return nil, err
	}
	view, err := inspect.NewSourceView(path)
	if err != nil {
		return nil, usageErr(fmt.Sprintf("%s: %v", path, err))
	}
	evidence := make([]inspect.Evidence, 0, len(selected))
	for _, ins := range selected {
		evidence = append(evidence, ins.Inspect(view, params))
	}
	return evidence, nil
}

// selectSourceInspectors resolves --inspector names against the source layer,
// defaulting to every source inspector.
func selectSourceInspectors(names []string) ([]inspect.SourceInspector, error) {
	if len(names) == 0 {
		return inspect.SourceInspectors(), nil
	}
	out := make([]inspect.SourceInspector, 0, len(names))
	for _, n := range names {
		ins, ok := inspect.SourceByName(n)
		if !ok {
			return nil, unknownInspectorErr(n)
		}
		out = append(out, ins)
	}
	return out, nil
}

// selectCollectionInspectors resolves --inspector names against the collection
// layer, defaulting to every collection inspector.
func selectCollectionInspectors(names []string) ([]inspect.CollectionInspector, error) {
	if len(names) == 0 {
		return inspect.CollectionInspectors(), nil
	}
	out := make([]inspect.CollectionInspector, 0, len(names))
	for _, n := range names {
		ins, ok := inspect.CollectionByName(n)
		if !ok {
			return nil, unknownInspectorErr(n)
		}
		out = append(out, ins)
	}
	return out, nil
}

func unknownInspectorErr(name string) error {
	return usageErr(fmt.Sprintf("unknown inspector %q (try `katalyst inspectors list`)", name))
}

// render produces the report bytes in the requested format. Both formats are
// projections of the same evidence, so -o and stdout write identical bytes.
// maxLines truncates Markdown only; JSON is always complete.
func render(evidence []inspect.Evidence, jsonOut bool, maxLines int) ([]byte, error) {
	if jsonOut {
		out, err := inspect.RenderJSON(evidence)
		if err != nil {
			return nil, err
		}
		return append(out, '\n'), nil
	}
	return []byte(inspect.RenderMarkdown(evidence, maxLines)), nil
}
