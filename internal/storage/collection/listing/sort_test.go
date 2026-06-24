package listing_test

import (
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/storage/collection/listing"
)

// sortIDs parses the sort spec, applies it, and returns the ordered ids.
func sortIDs(t *testing.T, recs []listing.Record, spec, missing string) []string {
	t.Helper()
	keys, err := listing.ParseSort(spec)
	if err != nil {
		t.Fatalf("ParseSort(%q): %v", spec, err)
	}
	out, err := listing.Apply(recs, listing.Options{Sorts: keys, SortMissing: missing})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return ids(out)
}

func TestSort_defaultIsIDAscending(t *testing.T) {
	recs := []listing.Record{{ID: "c"}, {ID: "a"}, {ID: "b"}}
	out, err := listing.Apply(recs, listing.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ids(out), ","); got != "a,b,c" {
		t.Errorf("default order = %q, want a,b,c", got)
	}
}

func TestSort_ascendingAndDescending(t *testing.T) {
	if got := sortIDs(t, books(), "year", "last"); strings.Join(got, ",") != "hobbit,dune,wip" {
		t.Errorf("asc year = %v, want [hobbit dune wip]", got)
	}
	if got := sortIDs(t, books(), "-year", "last"); strings.Join(got, ",") != "wip,dune,hobbit" {
		t.Errorf("desc year = %v, want [wip dune hobbit]", got)
	}
}

func TestSort_multiKeyPrecedence(t *testing.T) {
	recs := []listing.Record{
		{ID: "a", Meta: map[string]any{"year": 2000, "title": "Z"}},
		{ID: "b", Meta: map[string]any{"year": 2000, "title": "A"}},
		{ID: "c", Meta: map[string]any{"year": 1999, "title": "M"}},
	}
	// Primary -year desc, secondary title asc.
	if got := sortIDs(t, recs, "-year,title", "last"); strings.Join(got, ",") != "b,a,c" {
		t.Errorf("multi-key = %v, want [b a c]", got)
	}
}

func TestSort_byStatus(t *testing.T) {
	recs := []listing.Record{
		{ID: "bad", Status: 2},
		{ID: "ok", Status: 0},
		{ID: "warn", Status: 1},
	}
	if got := sortIDs(t, recs, "status", "last"); strings.Join(got, ",") != "ok,warn,bad" {
		t.Errorf("status asc = %v, want [ok warn bad]", got)
	}
}

func TestSort_missingLast(t *testing.T) {
	recs := []listing.Record{
		{ID: "has", Meta: map[string]any{"year": 2000}},
		{ID: "none", Meta: map[string]any{}},
	}
	// "last": missing goes to the end in both directions.
	if got := sortIDs(t, recs, "year", "last"); strings.Join(got, ",") != "has,none" {
		t.Errorf("missing last asc = %v, want [has none]", got)
	}
	if got := sortIDs(t, recs, "-year", "last"); strings.Join(got, ",") != "has,none" {
		t.Errorf("missing last desc = %v, want [has none]", got)
	}
}

func TestSort_missingLowest(t *testing.T) {
	recs := []listing.Record{
		{ID: "has", Meta: map[string]any{"year": 2000}},
		{ID: "none", Meta: map[string]any{}},
	}
	// "lowest": missing is below any value, so first asc, last desc.
	if got := sortIDs(t, recs, "year", "lowest"); strings.Join(got, ",") != "none,has" {
		t.Errorf("missing lowest asc = %v, want [none has]", got)
	}
	if got := sortIDs(t, recs, "-year", "lowest"); strings.Join(got, ",") != "has,none" {
		t.Errorf("missing lowest desc = %v, want [has none]", got)
	}
}

func TestSort_tieBreakByID(t *testing.T) {
	recs := []listing.Record{
		{ID: "c", Meta: map[string]any{"year": 2000}},
		{ID: "a", Meta: map[string]any{"year": 2000}},
		{ID: "b", Meta: map[string]any{"year": 2000}},
	}
	// Equal sort key: ties break by id ascending, even under desc.
	if got := sortIDs(t, recs, "-year", "last"); strings.Join(got, ",") != "a,b,c" {
		t.Errorf("tie break = %v, want [a b c]", got)
	}
}

func TestParseSort_errors(t *testing.T) {
	for _, spec := range []string{"", "-", " , "} {
		if _, err := listing.ParseSort(spec); err == nil {
			t.Errorf("ParseSort(%q) expected error", spec)
		}
	}
}
