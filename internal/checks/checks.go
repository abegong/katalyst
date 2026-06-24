// Package checks is the core of the check engine: the shared types every
// check type is built on (Context, Check, CollectionCheck, Violation) plus the
// registry that check types self-register with.
//
// Each check type lives in a per-family subpackage: structuredobject,
// markdownbodytext, filesystem, plaintext, with one file per check type
// holding its struct, Run, Descriptor, and an init() that calls Register. The
// subpackages import this core; this core imports none of them. Callers wire
// every family in by blank-importing internal/checks/all.
package checks

import "github.com/abegong/katalyst/internal/storage/collection/document"

// Context carries all data a check may need.
type Context struct {
	FilePath string
	// CollectionRoot is the absolute directory of the item's collection.
	// Path/filename targets that span directories (path-segments, path
	// depth) are resolved relative to it. Empty falls back to FilePath's
	// own directory.
	CollectionRoot string
	Doc            *document.Document
	Meta           map[string]any
}

// Severity classifies how serious a violation is. The zero value is
// SeverityError, so any check that does not set it keeps failing the run;
// SeverityWarning is advisory, it is reported but never changes the exit
// code. Warnings exist for judgment-call checks (prose tells, style nits)
// where a human decides per instance rather than the build deciding for
// them.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

// String returns "error" or "warning".
func (s Severity) String() string {
	if s == SeverityWarning {
		return "warning"
	}
	return "error"
}

// Violation is one failed check.
type Violation struct {
	Path    string
	Message string
	Line    int
	// File names the offending file for collection-scoped violations, which
	// are not tied to the single item being processed. Empty for per-item
	// checks (the caller already knows the file).
	File string
	// Severity defaults to SeverityError (the zero value). Checks emitting
	// advisory findings set SeverityWarning.
	Severity Severity
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

// LookupLine resolves the 1-based source line for a JSON-pointer path, walking
// up to the nearest ancestor present in lines. It is the shared line-mapping
// helper every field-scoped check uses to point a violation at its key.
func LookupLine(lines map[string]int, ptr string) int {
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
