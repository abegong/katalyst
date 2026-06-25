package inspect

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	SelectionAll       = "all"
	SelectionDir       = "dir"
	SelectionGlob      = "glob"
	SelectionExt       = "ext"
	SelectionPathUnder = "path_under"
)

// ParseSelection classifies the user-facing --select expression. It is
// path-only; content predicates belong to a later pass.
func ParseSelection(raw string) Selection {
	label := strings.TrimSpace(raw)
	if label == "" {
		return Selection{Label: "all files", Mode: SelectionAll}
	}
	if ext, ok := parseQuotedPredicate(label, "ext = "); ok {
		return Selection{Label: label, Mode: SelectionExt, Pattern: ext}
	}
	if prefix, ok := parseQuotedPredicate(label, "path under "); ok {
		return Selection{Label: label, Mode: SelectionPathUnder, Pattern: cleanSelectionPath(prefix)}
	}
	if strings.ContainsAny(label, "*?[") {
		return Selection{Label: label, Mode: SelectionGlob, Pattern: cleanSelectionPath(label)}
	}
	return Selection{Label: label, Mode: SelectionDir, Pattern: cleanSelectionPath(label)}
}

func parseQuotedPredicate(s, prefix string) (string, bool) {
	if !strings.HasPrefix(s, prefix) {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(s, prefix))
	if len(rest) >= 2 && rest[0] == '"' && rest[len(rest)-1] == '"' {
		return rest[1 : len(rest)-1], true
	}
	return rest, true
}

func cleanSelectionPath(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\\", "/"))
	s = strings.TrimPrefix(s, "./")
	return strings.Trim(s, "/")
}

func (v SourceView) selectFiles(sel Selection) ([]sourceFile, error) {
	if sel.Mode == "" || sel.Mode == SelectionAll {
		return sortedSourceFiles(v.files), nil
	}
	var out []sourceFile
	for _, f := range v.files {
		ok := false
		switch sel.Mode {
		case SelectionDir:
			prefix := strings.TrimSuffix(sel.Pattern, "/")
			ok = f.rel == prefix || strings.HasPrefix(f.rel, prefix+"/")
		case SelectionGlob:
			matched, err := doublestar.Match(sel.Pattern, f.rel)
			if err != nil {
				return nil, fmt.Errorf("select %q: %w", sel.Label, err)
			}
			ok = matched
		case SelectionExt:
			ok = f.ext == sel.Pattern
		case SelectionPathUnder:
			prefix := strings.TrimSuffix(sel.Pattern, "/")
			ok = f.rel == prefix || strings.HasPrefix(f.rel, prefix+"/")
		default:
			return nil, fmt.Errorf("select %q: unknown selection mode %q", sel.Label, sel.Mode)
		}
		if ok {
			out = append(out, f)
		}
	}
	return sortedSourceFiles(out), nil
}

func sortedSourceFiles(files []sourceFile) []sourceFile {
	out := append([]sourceFile(nil), files...)
	sortSourceFiles(out)
	return out
}

func sortSourceFiles(files []sourceFile) {
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
}
