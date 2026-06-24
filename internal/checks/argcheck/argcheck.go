// Package argcheck holds generic validators that check-type parsers use to
// produce uniform error phrasing without a central per-kind switch. The helpers
// carry no knowledge of any specific check; each takes the kind name and field
// name so the message reads the same across every check that distributes its own
// config parsing.
package argcheck

import (
	"fmt"
	"strings"
)

// RequireString errors when v is empty, with the canonical phrasing
// `<kind> requires "<field>"`.
func RequireString(kind, field, v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("%s requires %q", kind, field)
	}
	return nil
}

// RequireStrings errors when vs is empty, with `<kind> requires "<field>"`.
func RequireStrings(kind, field string, vs []string) error {
	if len(vs) == 0 {
		return fmt.Errorf("%s requires %q", kind, field)
	}
	return nil
}

// RequireOneOfFields errors when every named field is empty, with
// `<kind> requires "<a>" or "<b>"[ or "<c>"…]`. Use for checks that accept any
// of several alternative keys (e.g. name_affix's prefix/suffix).
func RequireOneOfFields(kind string, present bool, fields ...string) error {
	if present {
		return nil
	}
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = fmt.Sprintf("%q", f)
	}
	return fmt.Errorf("%s requires %s", kind, strings.Join(quoted, " or "))
}

// OneOf errors when v is not one of allowed, with
// `<kind>: "<field>" must be one of: a, b, c`. An empty v passes (pair with
// RequireString when the field is also mandatory); callers that default an
// empty value should validate the resolved value.
func OneOf(kind, field, v string, allowed ...string) error {
	if v == "" {
		return nil
	}
	for _, a := range allowed {
		if v == a {
			return nil
		}
	}
	return fmt.Errorf("%s: %q must be one of: %s", kind, field, strings.Join(allowed, ", "))
}
