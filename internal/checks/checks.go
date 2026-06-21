package checks

import "github.com/katabase-ai/katalyst/internal/frontmatter"

// Context carries all data a check may need.
type Context struct {
	FilePath string
	// CollectionRoot is the absolute directory of the item's collection.
	// Path/filename targets that span directories (path-segments, path
	// depth) are resolved relative to it. Empty falls back to FilePath's
	// own directory.
	CollectionRoot string
	Doc            *frontmatter.Document
	Meta           map[string]any
}

// Violation is one failed check.
type Violation struct {
	Path    string
	Message string
	Line    int
}

// Check validates one concern against a document context.
type Check interface {
	Run(ctx Context) []Violation
}

// RunAll executes checks and returns a flattened violation list.
func RunAll(ctx Context, checkList []Check) []Violation {
	out := make([]Violation, 0)
	for _, check := range checkList {
		out = append(out, check.Run(ctx)...)
	}
	return out
}

func lookupLine(lines map[string]int, ptr string) int {
	for {
		if line, ok := lines[ptr]; ok {
			return line
		}
		if ptr == "" {
			return 0
		}
		i := -1
		for idx := len(ptr) - 1; idx >= 0; idx-- {
			if ptr[idx] == '/' {
				i = idx
				break
			}
		}
		if i < 0 {
			return 0
		}
		ptr = ptr[:i]
	}
}
