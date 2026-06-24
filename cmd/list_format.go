package cmd

import (
	"fmt"
	"io"
	"strings"
)

// printListSectionHeader prints a count-bearing section header plus an
// underline divider for terminal list output.
func printListSectionHeader(out io.Writer, title string, count int) {
	header := fmt.Sprintf("%s (%d)", title, count)
	fmt.Fprintln(out, header)
	fmt.Fprintln(out, strings.Repeat("-", len(header)))
}
