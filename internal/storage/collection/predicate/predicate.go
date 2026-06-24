// Package predicate parses and evaluates metadata predicates used by collection
// variants and listing filters.
package predicate

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type op int

const (
	opEq op = iota
	opNe
	opGt
	opGte
	opLt
	opLte
	opRegex
	opIn
	opNin
	opExists
	opAbsent
)

// Predicate is one metadata condition, evaluated against an item's metadata.
// Build it with Parse; its internals are opaque.
type Predicate struct {
	field string // dot path into the frontmatter
	op    op
	want  any            // scalar comparand for eq/ne/gt/gte/lt/lte
	wants []any          // comma list for in/nin
	re    *regexp.Regexp // compiled pattern for =~
}

// TypeMismatchError reports a predicate comparison against an incomparable type
// when filterTypeMismatch is "error".
type TypeMismatchError struct{ Field string }

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("filter on %q: incomparable types", e.Field)
}

// Parse parses one shorthand predicate expression. The operator is the
// first one found scanning left to right (two-char operators win at a given
// position): >= <= != =~ > < =. A bare field is an existence test; a leading
// ! is an absence test. A comma-separated RHS on = / != becomes in / nin.
func Parse(s string) (Predicate, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Predicate{}, fmt.Errorf("empty filter expression")
	}

	field, o, rhs, ok := splitOp(s)
	if !ok {
		// No operator: existence or absence.
		if strings.HasPrefix(s, "!") {
			f := strings.TrimSpace(s[1:])
			if f == "" {
				return Predicate{}, fmt.Errorf("invalid filter %q", s)
			}
			return Predicate{field: f, op: opAbsent}, nil
		}
		return Predicate{field: s, op: opExists}, nil
	}

	field = strings.TrimSpace(field)
	rhs = strings.TrimSpace(rhs)
	if field == "" {
		return Predicate{}, fmt.Errorf("invalid filter %q (empty field)", s)
	}

	switch o {
	case opRegex:
		re, err := regexp.Compile(rhs)
		if err != nil {
			return Predicate{}, fmt.Errorf("filter %q: %w", s, err)
		}
		return Predicate{field: field, op: opRegex, re: re}, nil
	case opEq, opNe:
		if strings.Contains(rhs, ",") {
			parts := strings.Split(rhs, ",")
			wants := make([]any, 0, len(parts))
			for _, p := range parts {
				wants = append(wants, scalar(strings.TrimSpace(p)))
			}
			setOp := opIn
			if o == opNe {
				setOp = opNin
			}
			return Predicate{field: field, op: setOp, wants: wants}, nil
		}
		return Predicate{field: field, op: o, want: scalar(rhs)}, nil
	default:
		return Predicate{field: field, op: o, want: scalar(rhs)}, nil
	}
}

// splitOp finds the first operator in s and splits around it. Two-char
// operators are checked before single-char ones at the same position so
// ">=" never reads as ">".
func splitOp(s string) (field string, o op, rhs string, ok bool) {
	for i := 0; i < len(s); i++ {
		switch {
		case strings.HasPrefix(s[i:], ">="):
			return s[:i], opGte, s[i+2:], true
		case strings.HasPrefix(s[i:], "<="):
			return s[:i], opLte, s[i+2:], true
		case strings.HasPrefix(s[i:], "!="):
			return s[:i], opNe, s[i+2:], true
		case strings.HasPrefix(s[i:], "=~"):
			return s[:i], opRegex, s[i+2:], true
		case s[i] == '>':
			return s[:i], opGt, s[i+1:], true
		case s[i] == '<':
			return s[:i], opLt, s[i+1:], true
		case s[i] == '=':
			return s[:i], opEq, s[i+1:], true
		}
	}
	return "", 0, "", false
}

// Matches evaluates the predicate against an item's metadata map. It is the
// exported, per-item evaluator reused by collection-variant discriminators and
// item listing. The typeMismatch argument behaves as in listing.Apply: "skip"
// reports a non-match on an incomparable comparison, "error" returns a
// *TypeMismatchError.
func (p Predicate) Matches(meta map[string]any, typeMismatch string) (bool, error) {
	return p.match(meta, typeMismatch)
}

// match evaluates the predicate against an item's frontmatter. A missing
// field never matches a comparison (only opExists/opAbsent observe absence).
func (p Predicate) match(meta map[string]any, typeMismatch string) (bool, error) {
	v, present := lookup(meta, p.field)

	switch p.op {
	case opExists:
		return present, nil
	case opAbsent:
		return !present, nil
	}

	if !present {
		return false, nil
	}

	switch p.op {
	case opEq:
		// Mongo-style: equality against an array field matches when the
		// array contains the value; against a scalar it is plain equality.
		return containsAny(v, []any{p.want}), nil
	case opNe:
		return !containsAny(v, []any{p.want}), nil
	case opRegex:
		return p.re.MatchString(stringOf(v)), nil
	case opIn:
		return containsAny(v, p.wants), nil
	case opNin:
		return !containsAny(v, p.wants), nil
	case opGt, opGte, opLt, opLte:
		c, ok := compare(v, p.want)
		if !ok {
			if typeMismatch == "error" {
				return false, &TypeMismatchError{Field: p.field}
			}
			return false, nil
		}
		switch p.op {
		case opGt:
			return c > 0, nil
		case opGte:
			return c >= 0, nil
		case opLt:
			return c < 0, nil
		default: // opLte
			return c <= 0, nil
		}
	}
	return false, nil
}

// lookup resolves a dot path into nested frontmatter maps.
func lookup(meta map[string]any, path string) (any, bool) {
	var cur any = meta
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[part]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// scalar YAML-decodes a filter RHS so numbers, booleans, and strings type
// the same way they do in `item add` assignments. Duplicated from cmd's
// parseAssignment rather than imported: cmd is not importable, and this is
// three lines.
func scalar(s string) any {
	if s == "" {
		return ""
	}
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	return v
}

// equal reports value equality: numbers numerically, strings and bools by
// identity. Incomparable kinds are never equal (and never panic).
func equal(a, b any) bool {
	if af, ok := toFloat(a); ok {
		bf, ok2 := toFloat(b)
		return ok2 && af == bf
	}
	switch av := a.(type) {
	case string:
		bs, ok := b.(string)
		return ok && av == bs
	case bool:
		bb, ok := b.(bool)
		return ok && av == bb
	}
	return false
}

// compare orders two scalars: numbers numerically, strings
// lexicographically. ok is false when the pair is not directly comparable.
func compare(a, b any) (int, bool) {
	if af, ok := toFloat(a); ok {
		bf, ok2 := toFloat(b)
		if !ok2 {
			return 0, false
		}
		switch {
		case af < bf:
			return -1, true
		case af > bf:
			return 1, true
		default:
			return 0, true
		}
	}
	if as, ok := a.(string); ok {
		if bs, ok2 := b.(string); ok2 {
			return strings.Compare(as, bs), true
		}
	}
	return 0, false
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	default:
		return 0, false
	}
}

// containsAny reports whether v (or, if v is an array, any of its elements)
// equals one of wants.
func containsAny(v any, wants []any) bool {
	if arr, ok := v.([]any); ok {
		for _, el := range arr {
			for _, w := range wants {
				if equal(el, w) {
					return true
				}
			}
		}
		return false
	}
	for _, w := range wants {
		if equal(v, w) {
			return true
		}
	}
	return false
}

// stringOf renders a frontmatter value to the text =~ tests against.
func stringOf(v any) string {
	return fmt.Sprintf("%v", v)
}
