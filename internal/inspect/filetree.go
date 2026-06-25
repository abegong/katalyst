package inspect

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	smallTreeFileLimit = 30
	smallTreeDirLimit  = 12
)

var (
	camelPattern  = regexp.MustCompile(`^[a-z][A-Za-z0-9]*[A-Z][A-Za-z0-9]*$`)
	pascalPattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

type fileTreeSummary struct {
	fileCount           int
	dirCount            int
	maxDepth            int
	extensions          map[string]int
	regions             []fileTreeRegion
	directories         []fileTreeDirectory
	naming              fileTreeNaming
	representativePaths []string
	deepPaths           []string
	treeLines           []string
	paths               []string
}

type fileTreeRegion struct {
	path        string
	fileCount   int
	extensions  map[string]int
	dominantExt string
}

type fileTreeDirectory struct {
	path                string
	depth               int
	directFileCount     int
	descendantFileCount int
	extensions          map[string]int
	markdownHeavy       bool
}

type fileTreeNaming struct {
	buckets          map[string]int
	dominantBucket   string
	dominantCount    int
	comparableCount  int
	exceptions       []fileTreeNamingException
	byExtension      map[string]fileTreeNaming
	dominantExtScope string
}

type fileTreeNamingException struct {
	path   string
	bucket string
	ext    string
}

// buildFileTreeSummary computes deterministic filesystem metadata for the
// file_tree inspector. It only uses SourceView's walked path metadata.
func buildFileTreeSummary(v SourceView) map[string]any {
	files := append([]sourceFile(nil), v.files...)
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })

	s := fileTreeSummary{
		fileCount:  len(files),
		extensions: map[string]int{},
		paths:      make([]string, 0, len(files)),
	}
	dirs := map[string]bool{".": true}
	regionExts := map[string]map[string]int{}
	regionCounts := map[string]int{}
	dirDirect := map[string]int{}
	dirDesc := map[string]int{}
	dirExts := map[string]map[string]int{}
	deepThreshold := 4

	for _, f := range files {
		s.paths = append(s.paths, f.rel)
		s.extensions[f.ext]++
		depth := pathDepth(f.rel)
		if depth > s.maxDepth {
			s.maxDepth = depth
		}
		if depth > deepThreshold {
			s.deepPaths = append(s.deepPaths, f.rel)
		}

		for _, dir := range ancestorDirs(f.rel) {
			dirs[dir] = true
			dirDesc[dir]++
			if dirExts[dir] == nil {
				dirExts[dir] = map[string]int{}
			}
			dirExts[dir][f.ext]++
		}
		dirDirect[f.dir]++

		region := topLevelRegion(f.rel)
		regionCounts[region]++
		if regionExts[region] == nil {
			regionExts[region] = map[string]int{}
		}
		regionExts[region][f.ext]++
	}

	s.dirCount = len(dirs)
	for _, region := range sortedKeys(regionCounts) {
		exts := regionExts[region]
		s.regions = append(s.regions, fileTreeRegion{
			path:        region,
			fileCount:   regionCounts[region],
			extensions:  exts,
			dominantExt: dominantExtension(exts, regionCounts[region]),
		})
	}
	sort.SliceStable(s.regions, func(i, j int) bool {
		if s.regions[i].fileCount != s.regions[j].fileCount {
			return s.regions[i].fileCount > s.regions[j].fileCount
		}
		return s.regions[i].path < s.regions[j].path
	})

	for _, dir := range sortedKeys(dirs) {
		exts := dirExts[dir]
		if exts == nil {
			exts = map[string]int{}
		}
		md := exts[".md"]
		desc := dirDesc[dir]
		s.directories = append(s.directories, fileTreeDirectory{
			path:                dir,
			depth:               dirDepth(dir),
			directFileCount:     dirDirect[dir],
			descendantFileCount: desc,
			extensions:          exts,
			markdownHeavy:       md >= 3 && desc > 0 && md*100 >= desc*60,
		})
	}
	sort.SliceStable(s.directories, func(i, j int) bool {
		if s.directories[i].descendantFileCount != s.directories[j].descendantFileCount {
			return s.directories[i].descendantFileCount > s.directories[j].descendantFileCount
		}
		return s.directories[i].path < s.directories[j].path
	})

	s.naming = summarizeNaming(files, "")
	s.naming.byExtension = map[string]fileTreeNaming{}
	for _, ext := range sortedKeys(s.extensions) {
		extNaming := summarizeNaming(files, ext)
		if extNaming.comparableCount > 0 {
			s.naming.byExtension[ext] = extNaming
		}
	}
	s.naming.dominantExtScope = dominantNamingScope(s.naming.byExtension)
	s.representativePaths = representativePaths(files, 10)
	if len(files) <= smallTreeFileLimit && s.dirCount <= smallTreeDirLimit {
		s.treeLines = asciiTree(files)
	}
	return s.toMap()
}

func (s fileTreeSummary) toMap() map[string]any {
	return map[string]any{
		"file_count":           s.fileCount,
		"dir_count":            s.dirCount,
		"max_depth":            s.maxDepth,
		"extensions":           toAnyMap(s.extensions),
		"top_level_regions":    regionsToAny(s.regions),
		"directory_summaries":  directoriesToAny(s.directories),
		"naming":               namingToAny(s.naming),
		"representative_paths": stringsToAny(s.representativePaths),
		"deep_paths":           stringsToAny(s.deepPaths),
		"tree_entries":         stringsToAny(s.treeLines),
		"paths":                stringsToAny(s.paths),
	}
}

func regionsToAny(regions []fileTreeRegion) []any {
	out := make([]any, 0, len(regions))
	for _, r := range regions {
		out = append(out, map[string]any{
			"path":               r.path,
			"file_count":         r.fileCount,
			"extensions":         toAnyMap(r.extensions),
			"dominant_extension": r.dominantExt,
		})
	}
	return out
}

func directoriesToAny(dirs []fileTreeDirectory) []any {
	out := make([]any, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, map[string]any{
			"path":                  d.path,
			"depth":                 d.depth,
			"direct_file_count":     d.directFileCount,
			"descendant_file_count": d.descendantFileCount,
			"extensions":            toAnyMap(d.extensions),
			"markdown_heavy":        d.markdownHeavy,
		})
	}
	return out
}

func namingToAny(n fileTreeNaming) map[string]any {
	out := map[string]any{
		"buckets":          toAnyMap(n.buckets),
		"dominant_bucket":  n.dominantBucket,
		"dominant_count":   n.dominantCount,
		"comparable_count": n.comparableCount,
		"exceptions":       namingExceptionsToAny(n.exceptions),
	}
	if n.dominantExtScope != "" {
		out["dominant_extension_scope"] = n.dominantExtScope
	}
	if len(n.byExtension) > 0 {
		byExt := map[string]any{}
		for _, ext := range sortedKeys(n.byExtension) {
			child := n.byExtension[ext]
			child.byExtension = nil
			byExt[ext] = namingToAny(child)
		}
		out["by_extension"] = byExt
	}
	return out
}

func namingExceptionsToAny(exceptions []fileTreeNamingException) []any {
	out := make([]any, 0, len(exceptions))
	for _, e := range exceptions {
		out = append(out, map[string]any{"path": e.path, "bucket": e.bucket, "extension": e.ext})
	}
	return out
}

func stringsToAny(in []string) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func pathDepth(rel string) int {
	if rel == "" || rel == "." {
		return 0
	}
	return strings.Count(rel, "/") + 1
}

func dirDepth(dir string) int {
	if dir == "." || dir == "" {
		return 0
	}
	return strings.Count(dir, "/") + 1
}

func ancestorDirs(rel string) []string {
	dir := path.Dir(rel)
	out := []string{"."}
	if dir == "." {
		return out
	}
	parts := strings.Split(dir, "/")
	for i := range parts {
		out = append(out, strings.Join(parts[:i+1], "/"))
	}
	return out
}

func topLevelRegion(rel string) string {
	if !strings.Contains(rel, "/") {
		return "."
	}
	return strings.Split(rel, "/")[0] + "/"
}

func dominantExtension(exts map[string]int, total int) string {
	ext, n := dominantInt(exts)
	if n >= 3 && total > 0 && n*100 >= total*60 {
		return ext
	}
	return ""
}

func dominantInt(hist map[string]int) (string, int) {
	best, bestN := "", -1
	for _, k := range sortedKeys(hist) {
		if hist[k] > bestN {
			best, bestN = k, hist[k]
		}
	}
	return best, bestN
}

func summarizeNaming(files []sourceFile, extFilter string) fileTreeNaming {
	n := fileTreeNaming{buckets: map[string]int{}}
	var comparable []struct {
		file   sourceFile
		bucket string
	}
	for _, f := range files {
		if extFilter != "" && f.ext != extFilter {
			continue
		}
		stem := strings.TrimSuffix(path.Base(f.rel), path.Ext(f.rel))
		if stem == "" {
			continue
		}
		bucket := namingBucket(stem)
		n.buckets[bucket]++
		n.comparableCount++
		comparable = append(comparable, struct {
			file   sourceFile
			bucket string
		}{file: f, bucket: bucket})
	}
	n.dominantBucket, n.dominantCount = dominantInt(n.buckets)
	if n.dominantCount >= 3 && n.comparableCount > 0 && n.dominantCount*100 >= n.comparableCount*80 {
		for _, item := range comparable {
			if item.bucket != n.dominantBucket {
				n.exceptions = append(n.exceptions, fileTreeNamingException{
					path:   item.file.rel,
					bucket: item.bucket,
					ext:    item.file.ext,
				})
			}
		}
	} else {
		n.dominantBucket = ""
		n.dominantCount = 0
	}
	return n
}

func dominantNamingScope(byExt map[string]fileTreeNaming) string {
	bestExt, bestN := "", 0
	for _, ext := range sortedKeys(byExt) {
		n := byExt[ext]
		if n.dominantBucket != "" && n.comparableCount > bestN {
			bestExt, bestN = ext, n.comparableCount
		}
	}
	return bestExt
}

func namingBucket(stem string) string {
	switch {
	case strings.Contains(stem, " "):
		return "title/spaces"
	case isAllDigits(stem):
		return "numeric"
	case isAllLetters(stem) && stem == strings.ToLower(stem):
		return "lowercase"
	case isAllLetters(stem) && stem == strings.ToUpper(stem):
		return "uppercase"
	case strings.Contains(stem, "-") && kebabPattern.MatchString(stem):
		return "kebab-case"
	case strings.Contains(stem, "_") && snakePattern.MatchString(stem):
		return "snake_case"
	case camelPattern.MatchString(stem):
		return "camelCase"
	case pascalPattern.MatchString(stem):
		return "PascalCase"
	default:
		return "mixed/other"
	}
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

func isAllLetters(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return s != ""
}

func representativePaths(files []sourceFile, cap int) []string {
	byRegion := map[string][]string{}
	for _, f := range files {
		byRegion[topLevelRegion(f.rel)] = append(byRegion[topLevelRegion(f.rel)], f.rel)
	}
	for _, paths := range byRegion {
		sort.Strings(paths)
	}
	var out []string
	regions := sortedKeys(byRegion)
	for len(out) < cap {
		added := false
		for _, region := range regions {
			paths := byRegion[region]
			if len(paths) == 0 {
				continue
			}
			out = append(out, paths[0])
			byRegion[region] = paths[1:]
			added = true
			if len(out) == cap {
				break
			}
		}
		if !added {
			break
		}
	}
	return out
}

func asciiTree(files []sourceFile) []string {
	type node struct {
		name     string
		file     bool
		children map[string]*node
	}
	root := &node{name: ".", children: map[string]*node{}}
	for _, f := range files {
		cur := root
		parts := strings.Split(f.rel, "/")
		for i, part := range parts {
			child := cur.children[part]
			if child == nil {
				child = &node{name: part, children: map[string]*node{}}
				cur.children[part] = child
			}
			if i == len(parts)-1 {
				child.file = true
			}
			cur = child
		}
	}
	lines := []string{"."}
	var walk func(*node, string)
	walk = func(n *node, prefix string) {
		names := sortedKeys(n.children)
		sort.SliceStable(names, func(i, j int) bool {
			a, b := n.children[names[i]], n.children[names[j]]
			if a.file != b.file {
				return !a.file
			}
			return names[i] < names[j]
		})
		for i, name := range names {
			child := n.children[name]
			last := i == len(names)-1
			connector := "+-- "
			nextPrefix := prefix + "|   "
			if last {
				connector = "+-- "
				nextPrefix = prefix + "    "
			}
			label := child.name
			if !child.file {
				label += "/"
			}
			lines = append(lines, fmt.Sprintf("%s%s%s", prefix, connector, label))
			if !child.file {
				walk(child, nextPrefix)
			}
		}
	}
	walk(root, "")
	return lines
}
