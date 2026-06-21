package cmd

import (
	"fmt"
	"os"

	"github.com/katabase-ai/katalyst/internal/inspect"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var (
		jsonOut    bool
		outFile    string
		inspectors []string
		maxLines   int
		verbose    bool
	)

	c := &cobra.Command{
		Use:   "inspect <path>",
		Short: "Profile a directory of markdown files and report its shape.",
		Long: `inspect reads the markdown files under <path> and runs inspectors over
them, reporting the shape of the corpus: frontmatter field frequency and
types, markdown heading and section conventions, and filename conventions.

Inspectors describe; they never recommend. The output is evidence — counts
and distributions with the file count as denominator — for a human or agent
to judge. inspect writes no schema and mutates nothing under <path>.

Each inspector's results are prefixed with a one-line description of what they
mean. Long output is truncated per inspector to --max-lines (Markdown only;
--json is always complete); -v shows everything.

Output is Markdown by default; --json emits the same evidence as JSON.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if info, err := os.Stat(path); err != nil || !info.IsDir() {
				return usageErr(fmt.Sprintf("inspect: %q is not a readable directory", path))
			}

			selected, err := selectInspectors(inspectors)
			if err != nil {
				return err
			}

			corpus, err := inspect.Load(path)
			if err != nil {
				return usageErr(fmt.Sprintf("inspect: %v", err))
			}

			evidence := make([]inspect.Evidence, 0, len(selected))
			for _, ins := range selected {
				evidence = append(evidence, ins.Inspect(corpus))
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
					return usageErr(fmt.Sprintf("inspect: write %s: %v", outFile, err))
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
		"Run only the named inspector(s); repeatable. Default: all.")
	c.Flags().IntVar(&maxLines, "max-lines", 20,
		"Truncate each inspector's Markdown output to N lines (0 = no limit).")
	c.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Show full output; do not truncate (same as --max-lines 0).")
	return c
}

// selectInspectors resolves the --inspector names, defaulting to every
// inspector. An unknown name is a usage error.
func selectInspectors(names []string) ([]inspect.Inspector, error) {
	if len(names) == 0 {
		return inspect.All(), nil
	}
	out := make([]inspect.Inspector, 0, len(names))
	for _, n := range names {
		ins, ok := inspect.ByName(n)
		if !ok {
			return nil, usageErr(fmt.Sprintf("inspect: unknown inspector %q", n))
		}
		out = append(out, ins)
	}
	return out, nil
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
