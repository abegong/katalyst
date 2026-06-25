package inspect

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/abegong/katalyst/internal/storage"
	"github.com/abegong/katalyst/internal/storage/collection/document"
)

type contentIssue struct {
	Path   string
	Kind   string
	Detail string
}

// FileContentShape profiles a selected set of source files by light content
// parsing. It is filesystem-specific through SourceView.
type FileContentShape struct{}

func (FileContentShape) Name() string { return "file_content_shape" }

func (FileContentShape) AppliesTo(t storage.StorageType) bool { return t == storage.Filesystem }

func (FileContentShape) Inspect(v SourceView, p Params) Evidence {
	data := buildFileContentShape(v, p.Selection)
	return Evidence{Inspector: "file_content_shape", Scope: v.root, N: asInt(data["file_count"]), Data: data}
}

func buildFileContentShape(v SourceView, sel Selection) map[string]any {
	if sel.Mode == "" {
		sel = ParseSelection("")
	}
	files, err := v.selectFiles(sel)
	if err != nil {
		return map[string]any{
			"selector":     sel.Label,
			"file_count":   0,
			"issues":       issuesToAny([]contentIssue{{Kind: "selection", Detail: err.Error()}}),
			"coherence":    "mixed",
			"summary_text": err.Error(),
		}
	}

	exts := map[string]int{}
	dirs := map[string]bool{}
	readable, unsupported := 0, 0
	var issues []contentIssue
	mdFiles, csvFiles, jsonFiles := 0, 0, 0
	mdKeys, mdSections := map[string]int{}, map[string]int{}
	mdH1 := 0
	csvColumns := map[string]int{}
	var csvRows []int
	jsonTop := map[string]int{}
	jsonKeys := map[string]int{}
	jsonObjects := 0

	for _, f := range files {
		exts[f.ext]++
		dirs[f.dir] = true
		src, err := v.readFile(f.rel)
		if err != nil {
			issues = append(issues, contentIssue{Path: f.rel, Kind: "read_failed", Detail: err.Error()})
			continue
		}
		readable++
		switch f.ext {
		case ".md":
			mdFiles++
			doc, err := document.Parse(src)
			if err != nil {
				issues = append(issues, contentIssue{Path: f.rel, Kind: "parse_failed", Detail: err.Error()})
				continue
			}
			if doc.HasFrontmatter {
				for k := range doc.Meta {
					mdKeys[k]++
				}
			}
			seenSections := map[string]bool{}
			hasH1 := false
			for _, h := range headings(doc.Body) {
				if h.level == 1 {
					hasH1 = true
				}
				if h.level >= 2 {
					seenSections[h.text] = true
				}
			}
			if hasH1 {
				mdH1++
			}
			for section := range seenSections {
				mdSections[section]++
			}
		case ".csv":
			csvFiles++
			r := csv.NewReader(bytes.NewReader(src))
			records, err := r.ReadAll()
			if err != nil {
				issues = append(issues, contentIssue{Path: f.rel, Kind: "parse_failed", Detail: err.Error()})
				continue
			}
			if len(records) == 0 {
				csvRows = append(csvRows, 0)
				continue
			}
			seen := map[string]bool{}
			for _, col := range records[0] {
				seen[col] = true
			}
			for col := range seen {
				csvColumns[col]++
			}
			csvRows = append(csvRows, len(records)-1)
		case ".json":
			jsonFiles++
			var val any
			dec := json.NewDecoder(bytes.NewReader(src))
			dec.UseNumber()
			if err := dec.Decode(&val); err != nil {
				issues = append(issues, contentIssue{Path: f.rel, Kind: "parse_failed", Detail: err.Error()})
				continue
			}
			shape := jsonShape(val)
			jsonTop[shape]++
			if obj, ok := val.(map[string]any); ok {
				jsonObjects++
				for k := range obj {
					jsonKeys[k]++
				}
			}
		default:
			unsupported++
			issues = append(issues, contentIssue{Path: f.rel, Kind: "unsupported", Detail: "no first-cut content parser for " + extensionLabel(f.ext)})
		}
	}

	common, variation := contentCommonVariation(len(files), mdFiles, mdKeys, mdSections, mdH1, csvFiles, csvColumns, csvRows, jsonFiles, jsonTop, jsonObjects, jsonKeys, unsupported)
	coherence := contentCoherence(len(files), mdFiles, csvFiles, jsonFiles, unsupported)
	return map[string]any{
		"selector":            sel.Label,
		"file_count":          len(files),
		"dir_count":           len(dirs),
		"extensions":          toAnyMap(exts),
		"readable_count":      readable,
		"unsupported_count":   unsupported,
		"parse_failure_count": countIssueKind(issues, "parse_failed"),
		"coherence":           coherence,
		"common_structure":    stringsToAny(common),
		"variation":           stringsToAny(variation),
		"markdown": map[string]any{
			"files":            mdFiles,
			"h1":               mdH1,
			"frontmatter_keys": toAnyMap(mdKeys),
			"sections":         toAnyMap(mdSections),
		},
		"csv": map[string]any{
			"files":      csvFiles,
			"columns":    toAnyMap(csvColumns),
			"row_counts": rowStats(csvRows),
		},
		"json": map[string]any{
			"files":              jsonFiles,
			"top_level_shapes":   toAnyMap(jsonTop),
			"object_files":       jsonObjects,
			"common_object_keys": toAnyMap(jsonKeys),
		},
		"issues": issuesToAny(issues),
	}
}

func jsonShape(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case json.Number, float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func contentCommonVariation(total, mdFiles int, mdKeys, mdSections map[string]int, mdH1 int, csvFiles int, csvColumns map[string]int, csvRows []int, jsonFiles int, jsonTop map[string]int, jsonObjects int, jsonKeys map[string]int, unsupported int) ([]string, []string) {
	var common, variation []string
	if mdFiles > 0 {
		if mdH1 == mdFiles {
			common = append(common, fmt.Sprintf("%d/%d Markdown files have an H1", mdH1, mdFiles))
		}
		for _, k := range highFrequency(mdKeys, mdFiles, 0.8) {
			common = append(common, fmt.Sprintf("%d/%d Markdown files have frontmatter key %s", mdKeys[k], mdFiles, k))
		}
		for _, s := range highFrequency(mdSections, mdFiles, 0.8) {
			common = append(common, fmt.Sprintf("%d/%d Markdown files have section %s", mdSections[s], mdFiles, s))
		}
		for _, k := range midFrequency(mdKeys, mdFiles) {
			variation = append(variation, fmt.Sprintf("frontmatter key %s appears in %d/%d Markdown files", k, mdKeys[k], mdFiles))
		}
	}
	if csvFiles > 0 {
		for _, c := range highFrequency(csvColumns, csvFiles, 0.8) {
			common = append(common, fmt.Sprintf("%d/%d CSV files have column %s", csvColumns[c], csvFiles, c))
		}
		if stats := rowStats(csvRows); stats["files"].(int) > 0 {
			common = append(common, fmt.Sprintf("CSV row count ranges from %d to %d, median %d", stats["min"].(int), stats["max"].(int), stats["median"].(int)))
		}
		for _, c := range midFrequency(csvColumns, csvFiles) {
			variation = append(variation, fmt.Sprintf("column %s appears in %d/%d CSV files", c, csvColumns[c], csvFiles))
		}
	}
	if jsonFiles > 0 {
		for _, shape := range sortedKeys(jsonTop) {
			common = append(common, fmt.Sprintf("%d/%d JSON files are top-level %ss", jsonTop[shape], jsonFiles, shape))
		}
		for _, k := range highFrequency(jsonKeys, jsonObjects, 0.8) {
			common = append(common, fmt.Sprintf("%d/%d JSON object files have key %s", jsonKeys[k], jsonObjects, k))
		}
	}
	if unsupported > 0 {
		variation = append(variation, fmt.Sprintf("%d/%d selected files are unsupported by first-cut parsers", unsupported, total))
	}
	return common, variation
}

func contentCoherence(total, mdFiles, csvFiles, jsonFiles, unsupported int) string {
	if total == 0 {
		return "mixed"
	}
	best := mdFiles
	if csvFiles > best {
		best = csvFiles
	}
	if jsonFiles > best {
		best = jsonFiles
	}
	if unsupported == 0 && best == total {
		return "coherent"
	}
	if best*100 >= total*60 {
		return "partly_coherent"
	}
	return "mixed"
}

func highFrequency(hist map[string]int, denom int, threshold float64) []string {
	if denom == 0 {
		return nil
	}
	var out []string
	for _, k := range sortedKeys(hist) {
		if float64(hist[k])/float64(denom) >= threshold {
			out = append(out, k)
		}
	}
	return out
}

func midFrequency(hist map[string]int, denom int) []string {
	if denom == 0 {
		return nil
	}
	var out []string
	for _, k := range sortedKeys(hist) {
		if hist[k] > 0 && hist[k] < denom {
			out = append(out, k)
		}
	}
	return out
}

func rowStats(rows []int) map[string]any {
	if len(rows) == 0 {
		return map[string]any{"files": 0, "min": 0, "median": 0, "max": 0}
	}
	sorted := append([]int(nil), rows...)
	sort.Ints(sorted)
	return map[string]any{
		"files":  len(sorted),
		"min":    sorted[0],
		"median": sorted[len(sorted)/2],
		"max":    sorted[len(sorted)-1],
	}
}

func countIssueKind(issues []contentIssue, kind string) int {
	n := 0
	for _, issue := range issues {
		if issue.Kind == kind {
			n++
		}
	}
	return n
}

func issuesToAny(issues []contentIssue) []any {
	out := make([]any, 0, len(issues))
	for _, issue := range issues {
		m := map[string]any{"kind": issue.Kind, "detail": issue.Detail}
		if issue.Path != "" {
			m["path"] = issue.Path
		}
		out = append(out, m)
	}
	return out
}
