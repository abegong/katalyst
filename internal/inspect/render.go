package inspect

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// RenderJSON serializes evidence as an indented JSON array, the machine form
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
			if ev.Inspector == "file_tree" {
				for _, ln := range fileTreeMarkdownLines(ev.Data, maxLines <= 0) {
					b.WriteString(ln)
					b.WriteByte('\n')
				}
				continue
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

func fileTreeMarkdownLines(data map[string]any, expanded bool) []string {
	fileCount := asInt(data["file_count"])
	dirCount := asInt(data["dir_count"])
	maxDepth := asInt(data["max_depth"])
	extensions := anyMap(data["extensions"])
	lines := []string{"summary:"}
	lines = append(lines, alignRows([][]string{
		{"files", fmt.Sprintf("%d", fileCount)},
		{"directories", fmt.Sprintf("%d", dirCount)},
		{"max depth", fmt.Sprintf("%d", maxDepth)},
		{"dominant type", dominantExtensionSummary(fileCount, extensions)},
	}, "  ", ": ")...)

	tree := stringSlice(data["tree_entries"])
	if len(tree) > 0 {
		lines = appendSection(lines, "tree:")
		lines = append(lines, tree...)
	} else {
		regions := anySlice(data["top_level_regions"])
		limit := 5
		if expanded {
			limit = len(regions)
		}
		lines = appendSection(lines, "top-level regions:")
		rows := [][]string{{"REGION", "FILES", "TYPES"}}
		for i, region := range regions {
			if i >= limit {
				break
			}
			m := region.(map[string]any)
			path := m["path"].(string)
			count := asInt(m["file_count"])
			exts := anyMap(m["extensions"])
			rows = append(rows, []string{path, fmt.Sprintf("%d", count), topExtensions(exts, 2)})
		}
		lines = append(lines, alignTable(rows, "  ")...)
		if len(regions) > limit {
			lines = append(lines, fmt.Sprintf("  ... %d more top-level entries hidden; pass -v to show all", len(regions)-limit))
		}
	}

	extLimit := 5
	if expanded {
		extLimit = len(extensions)
	}
	lines = appendSection(lines, "file types:")
	lines = append(lines, histogramTableLines(extensions, extLimit, "TYPE", "FILES")...)
	if len(extensions) > extLimit {
		lines = append(lines, fmt.Sprintf("  ... %d more extensions hidden; pass -v to show all", len(extensions)-extLimit))
	}

	if naming := namingLines(anyMap(data["naming"]), expanded); len(naming) > 0 {
		lines = appendSection(lines, "naming:")
		lines = append(lines, naming...)
	}

	if expanded {
		lines = appendSection(lines, "directory density:")
		rows := [][]string{{"DIRECTORY", "FILES", "DIRECT", "NOTES"}}
		for _, item := range anySlice(data["directory_summaries"]) {
			dir := item.(map[string]any)
			label := dir["path"].(string)
			count := asInt(dir["descendant_file_count"])
			direct := asInt(dir["direct_file_count"])
			notes := "-"
			if heavy, _ := dir["markdown_heavy"].(bool); heavy {
				notes = "Markdown-heavy"
			}
			rows = append(rows, []string{label, fmt.Sprintf("%d", count), fmt.Sprintf("%d", direct), notes})
		}
		lines = append(lines, alignTable(rows, "  ")...)
		if deep := stringSlice(data["deep_paths"]); len(deep) > 0 {
			lines = appendSection(lines, "deep paths:")
			for _, rel := range deep {
				lines = append(lines, fmt.Sprintf("  %s", rel))
			}
		}
	}

	paths := stringSlice(data["representative_paths"])
	if len(paths) > 0 {
		limit := 5
		if expanded {
			limit = len(paths)
		}
		lines = appendSection(lines, "representative paths:")
		for i, rel := range paths {
			if i >= limit {
				break
			}
			lines = append(lines, fmt.Sprintf("  %s", rel))
		}
		if len(paths) > limit {
			lines = append(lines, fmt.Sprintf("  ... %d more representative paths hidden; pass -v to show all", len(paths)-limit))
		}
	}
	return lines
}

func appendSection(lines []string, label string) []string {
	return append(lines, "", "----------------------------------------", label)
}

func dominantExtensionSummary(fileCount int, exts map[string]any) string {
	if ext, n := dominantAnyExtension(exts); ext != "" && fileCount > 0 && n >= 3 && n*100 >= fileCount*60 {
		return fmt.Sprintf("%s (%d of %d files)", extensionLabel(ext), n, fileCount)
	}
	return "-"
}

func histogramTableLines(hist map[string]any, limit int, keyHeader, countHeader string) []string {
	items := sortedHistogram(hist)
	rows := [][]string{{keyHeader, countHeader}}
	for i, item := range items {
		if i >= limit {
			break
		}
		rows = append(rows, []string{extensionLabel(item.key), fmt.Sprintf("%d", item.n)})
	}
	return alignTable(rows, "  ")
}

func topExtensions(exts map[string]any, limit int) string {
	var parts []string
	for i, item := range sortedHistogram(exts) {
		if i >= limit {
			break
		}
		parts = append(parts, extensionLabel(item.key))
	}
	return joinStringsOrDash(parts)
}

func joinStringsOrDash(parts []string) string {
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func namingLines(naming map[string]any, expanded bool) []string {
	if len(naming) == 0 {
		return nil
	}
	scope := ""
	if raw, ok := naming["dominant_extension_scope"].(string); ok {
		scope = raw
	}
	active := naming
	if scope != "" {
		if byExt, ok := naming["by_extension"].(map[string]any); ok {
			if scoped, ok := byExt[scope].(map[string]any); ok {
				active = scoped
			}
		}
	}
	dominantBucket, _ := active["dominant_bucket"].(string)
	dominantCount := asInt(active["dominant_count"])
	comparableCount := asInt(active["comparable_count"])
	if dominantBucket == "" {
		if !expanded {
			return nil
		}
		lines := []string{"  no dominant filename bucket met the threshold"}
		return append(lines, bucketTableLines(anyMap(active["buckets"]))...)
	}

	subject := "filenames"
	if scope != "" {
		subject = fmt.Sprintf("%s filenames", extensionLabel(scope))
	}
	lines := []string{}
	lines = append(lines, alignRows([][]string{
		{"scope", subject},
		{"pattern", dominantBucket},
		{"matches", fmt.Sprintf("%d of %d files", dominantCount, comparableCount)},
	}, "  ", ": ")...)
	exceptions := anySlice(active["exceptions"])
	limit := 2
	if expanded {
		limit = len(exceptions)
	}
	if len(exceptions) > 0 {
		var examples []string
		for i, item := range exceptions {
			if i >= limit {
				break
			}
			m := item.(map[string]any)
			examples = append(examples, m["path"].(string))
		}
		lines = append(lines, "  exceptions:")
		for _, example := range examples {
			lines = append(lines, fmt.Sprintf("    %s", example))
		}
		if len(exceptions) > limit {
			lines = append(lines, fmt.Sprintf("    ... %d more naming exceptions hidden; pass -v to show all", len(exceptions)-limit))
		}
	}
	if expanded {
		lines = append(lines, bucketTableLines(anyMap(active["buckets"]))...)
	}
	return lines
}

func bucketTableLines(buckets map[string]any) []string {
	return append([]string{"", "  filename buckets:"}, histogramTableLines(buckets, len(buckets), "BUCKET", "FILES")...)
}

func alignRows(rows [][]string, prefix, sep string) []string {
	width := 0
	for _, row := range rows {
		if len(row) > 0 && len(row[0]) > width {
			width = len(row[0])
		}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s%-*s%s%s", prefix, width, row[0], sep, row[1]))
	}
	return lines
}

func alignTable(rows [][]string, prefix string) []string {
	if len(rows) == 0 {
		return nil
	}
	cols := 0
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	widths := make([]int, cols)
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		var b strings.Builder
		b.WriteString(prefix)
		for i := 0; i < cols; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			if i > 0 {
				b.WriteString("  ")
			}
			fmt.Fprintf(&b, "%-*s", widths[i], cell)
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}

type histogramItem struct {
	key string
	n   int
}

func sortedHistogram(hist map[string]any) []histogramItem {
	items := make([]histogramItem, 0, len(hist))
	for k, v := range hist {
		items = append(items, histogramItem{key: k, n: asInt(v)})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].n != items[j].n {
			return items[i].n > items[j].n
		}
		return items[i].key < items[j].key
	})
	return items
}

func dominantAnyExtension(hist map[string]any) (string, int) {
	items := sortedHistogram(hist)
	if len(items) == 0 {
		return "", 0
	}
	return items[0].key, items[0].n
}

func extensionLabel(ext string) string {
	if ext == "" {
		return "no extension"
	}
	return ext
}

func asInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	default:
		return 0
	}
}

func anyMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func anySlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

func stringSlice(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
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
