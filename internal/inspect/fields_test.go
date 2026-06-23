package inspect

import "testing"

func TestObjectFields_presenceTypesCardinalityValues(t *testing.T) {
	objs := []map[string]any{
		{"title": "A", "rating": 5, "status": "read"},
		{"title": "B", "rating": 3, "status": "read"},
		{"title": "C", "status": "reading"},
	}
	data := objectFields(objs)

	title := data["title"].(map[string]any)
	if title["present"].(int) != 3 {
		t.Errorf("title present = %v, want 3", title["present"])
	}

	rating := data["rating"].(map[string]any)
	if rating["present"].(int) != 2 {
		t.Errorf("rating present = %v, want 2", rating["present"])
	}

	status := data["status"].(map[string]any)
	if status["cardinality"].(int) != 2 {
		t.Errorf("status cardinality = %v, want 2", status["cardinality"])
	}
	vals, ok := status["values"].(map[string]any)
	if !ok {
		t.Fatalf("status values missing: %v", status)
	}
	if vals["read"].(int) != 2 {
		t.Errorf("status read count = %v, want 2", vals["read"])
	}
}

// A numeric value and a string value with the same textual form stay distinct:
// cardinality counts both, types shows the mix, and no merged value set ships.
func TestObjectFields_stringAndNumericDistinct(t *testing.T) {
	objs := []map[string]any{
		{"code": 5},
		{"code": "5"},
	}
	data := objectFields(objs)
	code := data["code"].(map[string]any)

	types := code["types"].(map[string]any)
	if len(types) != 2 {
		t.Errorf("code types = %v, want integer+string", types)
	}
	if code["cardinality"].(int) != 2 {
		t.Errorf("code cardinality = %v, want 2 (5 and \"5\" are distinct)", code["cardinality"])
	}
	if _, ok := code["values"]; ok {
		t.Errorf("a mixed-kind field must not emit a merged value set: %v", code)
	}
}

// Arrays and objects are typed but contribute no value set (#58).
func TestObjectFields_nonScalarNoValueSet(t *testing.T) {
	objs := []map[string]any{
		{"tags": []any{"a", "b"}},
		{"tags": []any{"a"}},
	}
	data := objectFields(objs)
	tags := data["tags"].(map[string]any)
	if _, ok := tags["values"]; ok {
		t.Errorf("array field must not emit a value set: %v", tags)
	}
	types := tags["types"].(map[string]any)
	if types["array"].(int) != 2 {
		t.Errorf("tags array type count = %v, want 2", types)
	}
}
