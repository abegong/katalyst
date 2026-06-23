package inspect

import (
	"fmt"
	"testing"
)

func TestSummarize_classesAndOutliers(t *testing.T) {
	var profiles []Profile
	for i := 0; i < 190; i++ {
		profiles = append(profiles, Profile{Label: fmt.Sprintf("a%03d", i), Features: []string{"md", "kebab"}})
	}
	for i := 0; i < 7; i++ {
		profiles = append(profiles, Profile{Label: fmt.Sprintf("b%03d", i), Features: []string{"md", "snake"}})
	}
	profiles = append(profiles,
		Profile{Label: "x1", Features: []string{"png"}},
		Profile{Label: "x2", Features: []string{"pdf"}},
		Profile{Label: "x3", Features: []string{"csv"}},
	)

	p, _ := ParseParams("exact", -1, 0)
	out := summarize(profiles, p)

	classes := out["classes"].([]any)
	if len(classes) != 2 {
		t.Fatalf("classes = %d, want 2", len(classes))
	}
	if outliers := out["outliers"].([]any); len(outliers) != 3 {
		t.Errorf("outliers = %d, want 3", len(outliers))
	}
	top := classes[0].(map[string]any)
	if top["size"].(int) != 190 {
		t.Errorf("top class size = %v, want 190", top["size"])
	}
}

// Higher tolerance (lower threshold) collapses near-but-distinct profiles, so
// the class count drops.
func TestSummarize_higherToleranceFewerClasses(t *testing.T) {
	profiles := []Profile{
		{Label: "p1", Features: []string{"a", "b", "c"}},
		{Label: "p2", Features: []string{"a", "b", "d"}}, // Jaccard with p1 = 2/4 = 0.5
		{Label: "p3", Features: []string{"x", "y", "z"}},
	}
	exact, _ := ParseParams("exact", -1, 0) // threshold 1.0
	coarse, _ := ParseParams("", 0.5, 0)    // threshold 0.5

	nExact := classCount(summarize(profiles, exact))
	nCoarse := classCount(summarize(profiles, coarse))
	if nCoarse >= nExact {
		t.Errorf("expected fewer classes at higher tolerance: exact=%d coarse=%d", nExact, nCoarse)
	}
}

func TestSummarize_budgetCapsClasses(t *testing.T) {
	profiles := []Profile{
		{Label: "p1", Features: []string{"a"}},
		{Label: "p2", Features: []string{"b"}},
		{Label: "p3", Features: []string{"c"}},
		{Label: "p4", Features: []string{"d"}},
	}
	p, _ := ParseParams("", -1, 2)
	if got := classCount(summarize(profiles, p)); got > 2 {
		t.Errorf("budget 2 exceeded: %d classes", got)
	}
}

// classCount is the number of distinct classes, counting singleton outliers.
func classCount(out map[string]any) int {
	return len(out["classes"].([]any)) + len(out["outliers"].([]any))
}
