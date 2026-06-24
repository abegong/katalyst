package cmd

import (
	"fmt"
	"io"
	"strings"
)

// printSectionHeader prints a section title and underline divider.
func printSectionHeader(out io.Writer, title string) {
	fmt.Fprintln(out, title)
	fmt.Fprintln(out, strings.Repeat("-", len(title)))
}

// printListSectionHeader prints a count-bearing section header plus an
// underline divider for terminal list output.
func printListSectionHeader(out io.Writer, title string, count int) {
	printSectionHeader(out, fmt.Sprintf("%s (%d)", title, count))
}
