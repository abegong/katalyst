// Package structuredobject holds the check types that validate structured
// frontmatter fields: schema validation and the field-shape checks. Each check
// type lives in its own file with its Descriptor and self-registration.
package structuredobject

// typeMatches reports whether v satisfies the named JSON type.
func typeMatches(v any, expected string) bool {
	switch expected {
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean", "bool":
		_, ok := v.(bool)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		return ok
	case "number":
		_, ok := toFloat(v)
		return ok
	case "integer", "int":
		return isInteger(v)
	default:
		return false
	}
}

func isInteger(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

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
