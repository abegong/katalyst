package plaintext

import (
	"fmt"
	"regexp"
)

// textTargetNames are the span selectors a text rule accepts. Empty defaults to
// "body". They live here, with the checks that own them.
var textTargetNames = map[string]bool{
	"body": true, "line": true, "first-line": true, "matched-lines": true,
}

// validateTextTarget allows an empty target (defaults to body) and rejects any
// non-empty value outside the text span set.
func validateTextTarget(kind, target string) error {
	if target == "" || textTargetNames[target] {
		return nil
	}
	return fmt.Errorf("%s: unknown target %q", kind, target)
}

// validateSelect requires a select regex with target "matched-lines" and
// forbids it otherwise.
func validateSelect(kind, target, sel string) error {
	if target == "matched-lines" {
		if sel == "" {
			return fmt.Errorf(`%s: target "matched-lines" requires "select"`, kind)
		}
		if _, err := regexp.Compile(sel); err != nil {
			return fmt.Errorf("%s: invalid select %q: %w", kind, sel, err)
		}
		return nil
	}
	if sel != "" {
		return fmt.Errorf(`%s: "select" is only valid with target "matched-lines"`, kind)
	}
	return nil
}
