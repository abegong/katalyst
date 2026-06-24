package filesystem

import "fmt"

// caseStyleNames and targetNames are the values the filesystem checks accept.
// They live here, with the checks that own them, rather than in config.
var caseStyleNames = map[string]bool{
	"kebab": true, "snake": true, "screaming-snake": true,
	"camel": true, "pascal": true, "point": true, "lower": true,
}

var targetNames = map[string]bool{
	"filename": true, "filename-ext": true, "parent-dir": true, "path-segments": true,
}

// validateTarget allows an empty target (the check defaults it) and rejects any
// non-empty value outside the known filesystem target set.
func validateTarget(kind, target string) error {
	if target == "" || targetNames[target] {
		return nil
	}
	return fmt.Errorf("%s: unknown target %q", kind, target)
}

// validateCaseStyle requires a known case style.
func validateCaseStyle(kind, style string) error {
	if style == "" {
		return fmt.Errorf("%s requires %q", kind, "style")
	}
	if !caseStyleNames[style] {
		return fmt.Errorf("%s: unknown style %q", kind, style)
	}
	return nil
}

// intPtr truncates a *float64 (the shared yaml min/max) to a *int for
// integer-bounded checks. Nil in, nil out.
func intPtr(f *float64) *int {
	if f == nil {
		return nil
	}
	n := int(*f)
	return &n
}
