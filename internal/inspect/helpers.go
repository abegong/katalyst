package inspect

import (
	"sort"
	"strconv"
)

// eachValue calls fn for every (key, value) pair across the frontmatter of
// every file in the corpus. Files without frontmatter contribute nothing.
func eachValue(c Corpus, fn func(key string, v any)) {
	for _, f := range c.Files {
		for k, v := range meta(f) {
			fn(k, v)
		}
	}
}

// jsonType names the JSON-shaped type of a decoded YAML value. yaml.v3 decodes
// integers as Go ints and floats as float64, so both map cleanly here.
func jsonType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return "unknown"
	}
}

// toFloat converts a numeric value to float64. Booleans and strings are not
// numeric and return false.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	default:
		return 0, false
	}
}

// scalarString renders a scalar value (string, bool, number) to a stable
// string for value-set counting. Arrays and objects are not scalars and
// return false, so they never enter an enum candidate set.
func scalarString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	default:
		if f, ok := toFloat(v); ok {
			// 'f' with -1 precision yields the shortest exact form, so the
			// integer 5 and the value-set key "5" agree.
			return strconv.FormatFloat(f, 'f', -1, 64), true
		}
		return "", false
	}
}

// sortedKeys returns a map's keys in sorted order.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
