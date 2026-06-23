package inspect

import "testing"

func TestParseParams_defaultGrouped(t *testing.T) {
	p, err := ParseParams("", -1, 0)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	if p.mode != thresholdMode {
		t.Fatalf("mode = %v, want thresholdMode", p.mode)
	}
	if p.threshold != detailThresholds["grouped"] {
		t.Errorf("threshold = %v, want grouped %v", p.threshold, detailThresholds["grouped"])
	}
}

func TestParseParams_mutuallyExclusive(t *testing.T) {
	if _, err := ParseParams("coarse", 0.5, 0); err == nil {
		t.Error("expected error for detail + similarity")
	}
	if _, err := ParseParams("", 0.5, 3); err == nil {
		t.Error("expected error for similarity + max-classes")
	}
	if _, err := ParseParams("exact", -1, 4); err == nil {
		t.Error("expected error for detail + max-classes")
	}
}

func TestParseParams_eachForm(t *testing.T) {
	if p, err := ParseParams("exact", -1, 0); err != nil || p.threshold != 1.0 {
		t.Errorf("exact: p=%+v err=%v", p, err)
	}
	if p, err := ParseParams("", 0.25, 0); err != nil || p.mode != thresholdMode || p.threshold != 0.25 {
		t.Errorf("similarity: p=%+v err=%v", p, err)
	}
	if p, err := ParseParams("", -1, 5); err != nil || p.mode != budgetMode || p.maxClasses != 5 {
		t.Errorf("budget: p=%+v err=%v", p, err)
	}
}

func TestParseParams_invalid(t *testing.T) {
	if _, err := ParseParams("nope", -1, 0); err == nil {
		t.Error("expected error for unknown detail level")
	}
	if _, err := ParseParams("", 2.0, 0); err == nil {
		t.Error("expected error for similarity > 1")
	}
}
