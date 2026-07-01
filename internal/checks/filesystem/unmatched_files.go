package filesystem

import (
	"fmt"
	slashpath "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/abegong/katalyst/internal/checks"
)

// UnmatchedFiles reports files in a filesystem scope that match neither
// include nor exclude patterns.
type UnmatchedFiles struct{}

func (UnmatchedFiles) RunCollection(ctx checks.CollectionContext) []checks.Violation {
	reports := unmatchedReports(ctx)
	out := make([]checks.Violation, 0, len(reports))
	for _, report := range reports {
		message := fmt.Sprintf("unmatched file (matches no include pattern %s and no exclude pattern %s)", patternList(ctx.Include), patternList(ctx.Exclude))
		if report.Count > 1 {
			message = fmt.Sprintf("unmatched files (%d files; matches no include pattern %s and no exclude pattern %s)", report.Count, patternList(ctx.Include), patternList(ctx.Exclude))
		}
		out = append(out, checks.Violation{
			File:    report.File,
			Message: message,
		})
	}
	return out
}

type unmatchedReport struct {
	File  string
	Count int
}

func unmatchedReports(ctx checks.CollectionContext) []unmatchedReport {
	unmatched := append([]string(nil), ctx.Unmatched...)
	sort.Strings(unmatched)
	if ctx.Verbose {
		reports := make([]unmatchedReport, 0, len(unmatched))
		for _, rel := range unmatched {
			reports = append(reports, unmatchedReport{File: rel, Count: 1})
		}
		return reports
	}

	selectedDirs := selectedSubtreeDirs(ctx)
	unmatchedCounts := unmatchedSubtreeCounts(unmatched)

	groups := map[string][]string{}
	var singles []string
	for _, rel := range unmatched {
		group := shallowestUnmatchedDir(rel, selectedDirs, unmatchedCounts)
		if group == "" {
			singles = append(singles, rel)
			continue
		}
		groups[group] = append(groups[group], rel)
	}

	reports := make([]unmatchedReport, 0, len(singles)+len(groups))
	for _, rel := range singles {
		reports = append(reports, unmatchedReport{File: rel, Count: 1})
	}
	for dir, members := range groups {
		if len(members) < 2 {
			reports = append(reports, unmatchedReport{File: members[0], Count: 1})
			continue
		}
		reports = append(reports, unmatchedReport{File: dir + "/", Count: len(members)})
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].File < reports[j].File
	})
	return reports
}

func selectedSubtreeDirs(ctx checks.CollectionContext) map[string]bool {
	out := map[string]bool{}
	for _, it := range ctx.Items {
		rel := relFromRoot(ctx.Root, it.FilePath)
		for _, dir := range ancestorDirs(rel) {
			out[dir] = true
		}
	}
	return out
}

func unmatchedSubtreeCounts(rels []string) map[string]int {
	out := map[string]int{}
	for _, rel := range rels {
		for _, dir := range ancestorDirs(rel) {
			out[dir]++
		}
	}
	return out
}

func shallowestUnmatchedDir(rel string, selectedDirs map[string]bool, unmatchedCounts map[string]int) string {
	for _, dir := range ancestorDirs(rel) {
		if !selectedDirs[dir] && unmatchedCounts[dir] > 1 {
			return dir
		}
	}
	return ""
}

func ancestorDirs(rel string) []string {
	dir := slashpath.Dir(slashpath.Clean(rel))
	if dir == "." || dir == "/" {
		return nil
	}
	parts := strings.Split(dir, "/")
	out := make([]string, 0, len(parts))
	for i := range parts {
		out = append(out, strings.Join(parts[:i+1], "/"))
	}
	return out
}

func relFromRoot(root, filePath string) string {
	if root != "" {
		rel, err := filepath.Rel(root, filePath)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(filePath)
}

func patternList(patterns []string) string {
	if len(patterns) == 0 {
		return "[]"
	}
	return "[" + strings.Join(patterns, ", ") + "]"
}

func init() {
	registerParsed(checks.Descriptor{
		CheckType:      checks.CheckFilesystemUnmatchedFiles,
		Family:         "fileSystem",
		ConfigurableIn: []string{checks.ConfigFilesystem},
		Slug:           "unmatched-files",
		Title:          "Unmatched files",
		Summary:        "Report regular files under a filesystem scope that match neither include nor exclude patterns.",
		Scope:          "collection",
		ConfigExample: `filesystemChecks:
  - path: docs
    include: ["**/*.md"]
    exclude: ["**/_generated/**"]
    checks:
      - kind: filesystem_unmatched_files`,
	}, checks.NoArgs, nil, func(any) checks.CollectionCheck {
		return UnmatchedFiles{}
	})
}
