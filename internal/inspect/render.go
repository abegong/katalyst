package inspect

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderJSON serializes evidence as an indented JSON array — the machine form
// for callers that parse results. Each record is enriched with its one-line
// description from the registry. JSON is never truncated: it must stay complete
// and parseable. One of two projections of the same evidence; RenderMarkdown is
// the other.
func RenderJSON(evs []Evidence) ([]byte, error) {
	enriched := make([]Evidence, len(evs))
	for i, ev := range evs {
		if ev.Description == "" {
			ev.Description = Summary(ev.Inspector)
		}
		enriched[i] = ev
	}
	return json.MarshalIndent(enriched, "", "  ")
}

// RenderMarkdown projects evidence into a human-readable report, grouped by
// family in Families() order, each inspector prefixed with a one-line
// description of what its results mean. It is the default rendering: agents
// read Markdown well and humans read it for free.
//
// maxLines caps each inspector's data output: an inspector that would print
// more than maxLines lines is truncated with a notice, so one wide field can't
// drown the report. maxLines <= 0 disables truncation. The cap is per
// inspector, not for the whole report.
func RenderMarkdown(evs []Evidence, maxLines int) string {
	var b strings.Builder
	scope := ""
	if len(evs) > 0 {
		scope = evs[0].Scope
	}
	fmt.Fprintf(&b, "# Inspection report: %s\n", scope)

	for _, fam := range Families() {
		var inFamily []Evidence
		for _, ev := range evs {
			if familyOf(ev.Inspector) == fam.ID {
				inFamily = append(inFamily, ev)
			}
		}
		if len(inFamily) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n", fam.Title)
		for _, ev := range inFamily {
			fmt.Fprintf(&b, "\n### %s (n=%d)\n\n", ev.Inspector, ev.N)
			if s := Summary(ev.Inspector); s != "" {
				fmt.Fprintf(&b, "_%s_\n\n", s)
			}
			lines := dataLines(ev.Data)
			if maxLines > 0 && len(lines) > maxLines {
				hidden := len(lines) - maxLines
				lines = lines[:maxLines]
				for _, ln := range lines {
					b.WriteString(ln)
					b.WriteByte('\n')
				}
				fmt.Fprintf(&b, "- … %d more line(s) truncated (pass --max-lines 0 or -v to show all)\n", hidden)
				continue
			}
			for _, ln := range lines {
				b.WriteString(ln)
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

// dataLines renders one inspector's Data to individual Markdown lines, so the
// caller can count and truncate them.
func dataLines(data map[string]any) []string {
	var b strings.Builder
	for _, k := range sortedKeys(data) {
		renderKV(&b, 0, k, data[k])
	}
	s := strings.TrimRight(b.String(), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// familyOf looks up an inspector's family from the registry.
func familyOf(name string) string {
	for _, d := range Descriptors() {
		if d.Name == name {
			return d.Family
		}
	}
	return ""
}

// renderKV writes one evidence entry as a Markdown bullet, recursing into
// nested maps and listing slice elements compactly.
func renderKV(b *strings.Builder, indent int, key string, val any) {
	pad := strings.Repeat("  ", indent)
	switch v := val.(type) {
	case map[string]any:
		fmt.Fprintf(b, "%s- %s:\n", pad, key)
		for _, k := range sortedKeys(v) {
			renderKV(b, indent+1, k, v[k])
		}
	case []any:
		fmt.Fprintf(b, "%s- %s:\n", pad, key)
		for _, el := range v {
			fmt.Fprintf(b, "%s  - %s\n", pad, compact(el))
		}
	case []string:
		fmt.Fprintf(b, "%s- %s: [%s]\n", pad, key, strings.Join(v, ", "))
	default:
		fmt.Fprintf(b, "%s- %s: %v\n", pad, key, v)
	}
}

// compact renders a value on one line, used for slice elements.
func compact(v any) string {
	switch x := v.(type) {
	case map[string]any:
		parts := make([]string, 0, len(x))
		for _, k := range sortedKeys(x) {
			parts = append(parts, fmt.Sprintf("%s=%s", k, compact(x[k])))
		}
		return strings.Join(parts, " ")
	case []string:
		return "[" + strings.Join(x, ", ") + "]"
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = compact(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", x)
	}
}
