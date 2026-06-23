package inspect

import "strconv"

// objectFields builds a data dictionary over a set of objects (frontmatter
// maps). Per field it reports presence over n, an observed type histogram,
// scalar value cardinality, and — when the field is a single-scalar-type enum
// candidate — the value set with counts. String and numeric scalars are kept
// distinct: a numeric 5 and the string "5" are different values and never share
// a value set. Array and object values are typed but contribute no value set
// (deepening that is issue #58). This is the object_fields primitive; the five
// former object_field_* inspectors are columns of this one table.
func objectFields(objs []map[string]any) map[string]any {
	type acc struct {
		present int
		types   map[string]int
		// byKind partitions distinct scalar values by scalar kind so a string
		// value set and a numeric value set never merge.
		byKind    map[string]map[string]int
		nonScalar bool
	}
	accs := map[string]*acc{}
	for _, obj := range objs {
		for k, v := range obj {
			a := accs[k]
			if a == nil {
				a = &acc{types: map[string]int{}, byKind: map[string]map[string]int{}}
				accs[k] = a
			}
			a.present++
			a.types[jsonType(v)]++
			if kind, repr, ok := scalarKey(v); ok {
				if a.byKind[kind] == nil {
					a.byKind[kind] = map[string]int{}
				}
				a.byKind[kind][repr]++
			} else {
				a.nonScalar = true
			}
		}
	}
	data := make(map[string]any, len(accs))
	for k, a := range accs {
		cardinality := 0
		for _, set := range a.byKind {
			cardinality += len(set)
		}
		entry := map[string]any{
			"present":     a.present,
			"types":       toAnyMap(a.types),
			"cardinality": cardinality,
		}
		// Emit a value set only for a single-scalar-type field small enough to
		// be an enum candidate. A field mixing scalar kinds (or carrying
		// non-scalars) reports cardinality only — the type histogram already
		// shows the mix, and merging kinds would lump string with numeric.
		if !a.nonScalar && len(a.byKind) == 1 && cardinality > 0 && cardinality <= maxValueSet {
			for _, set := range a.byKind {
				entry["values"] = toAnyMap(set)
			}
		}
		data[k] = entry
	}
	return data
}

// scalarKey classifies a scalar value into a kind (string / number / boolean)
// and its stable string form. Non-scalars (arrays, objects, null) return false,
// so they never enter a value set. The kind keeps string and numeric values in
// separate buckets even when their string forms coincide ("5" vs 5).
func scalarKey(v any) (kind, repr string, ok bool) {
	switch x := v.(type) {
	case string:
		return "string", x, true
	case bool:
		if x {
			return "boolean", "true", true
		}
		return "boolean", "false", true
	default:
		if f, ok := toFloat(v); ok {
			// -1 precision yields the shortest exact form, so the integer 5 and
			// the value-set key "5" agree.
			return "number", strconv.FormatFloat(f, 'f', -1, 64), true
		}
		return "", "", false
	}
}

// toAnyMap copies a typed-value map into a map[string]any for Evidence payloads.
func toAnyMap[V any](m map[string]V) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
