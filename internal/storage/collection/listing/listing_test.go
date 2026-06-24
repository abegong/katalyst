package listing_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/storage/collection/listing"
	"github.com/abegong/katalyst/internal/storage/collection/predicate"
)

func ids(recs []listing.Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.ID
	}
	return out
}

func books() []listing.Record {
	return []listing.Record{
		{ID: "dune", Meta: map[string]any{"year": 1965, "status": "published", "tags": []any{"sci-fi", "classic"}, "title": "Dune"}},
		{ID: "hobbit", Meta: map[string]any{"year": 1937, "status": "published", "tags": []any{"fantasy"}, "title": "The Hobbit"}},
		{ID: "wip", Meta: map[string]any{"year": 2025, "status": "draft", "title": "Work in Progress"}},
	}
}

func TestApply_grepRegions(t *testing.T) {
	recs := []listing.Record{
		{ID: "a", Raw: []byte("---\ntitle: A\n---\nbody has TODO\n"), Frontmatter: []byte("title: A\n"), Body: []byte("body has TODO\n")},
		{ID: "b", Raw: []byte("---\ntitle: TODO\n---\nclean body\n"), Frontmatter: []byte("title: TODO\n"), Body: []byte("clean body\n")},
	}
	re := regexp.MustCompile("TODO")

	all, _ := listing.Apply(recs, listing.Options{Greps: []*regexp.Regexp{re}, GrepIn: listing.RegionAll})
	if strings.Join(ids(all), ",") != "a,b" {
		t.Errorf("grep all = %v, want [a b]", ids(all))
	}
	body, _ := listing.Apply(recs, listing.Options{Greps: []*regexp.Regexp{re}, GrepIn: listing.RegionBody})
	if strings.Join(ids(body), ",") != "a" {
		t.Errorf("grep body = %v, want [a]", ids(body))
	}
	fm, _ := listing.Apply(recs, listing.Options{Greps: []*regexp.Regexp{re}, GrepIn: listing.RegionFrontmatter})
	if strings.Join(ids(fm), ",") != "b" {
		t.Errorf("grep frontmatter = %v, want [b]", ids(fm))
	}
}

func TestApply_grepAndFilterCombine(t *testing.T) {
	recs := []listing.Record{
		{ID: "a", Meta: map[string]any{"status": "draft"}, Raw: []byte("TODO here")},
		{ID: "b", Meta: map[string]any{"status": "draft"}, Raw: []byte("nothing")},
		{ID: "c", Meta: map[string]any{"status": "published"}, Raw: []byte("TODO here")},
	}
	p, _ := predicate.Parse("status=draft")
	out, _ := listing.Apply(recs, listing.Options{
		Filters: []predicate.Predicate{p},
		Greps:   []*regexp.Regexp{regexp.MustCompile("TODO")},
	})
	if strings.Join(ids(out), ",") != "a" {
		t.Errorf("filter AND grep = %v, want [a]", ids(out))
	}
}

func TestApply_skipAndLimitAfterSort(t *testing.T) {
	recs := []listing.Record{
		{ID: "a", Meta: map[string]any{"year": 2001}},
		{ID: "b", Meta: map[string]any{"year": 2002}},
		{ID: "c", Meta: map[string]any{"year": 2003}},
		{ID: "d", Meta: map[string]any{"year": 2004}},
	}
	keys, _ := listing.ParseSort("-year")
	out, _ := listing.Apply(recs, listing.Options{Sorts: keys, Skip: 1, Limit: 2})
	// -year: d,c,b,a → skip 1 → c,b,a → limit 2 → c,b
	if strings.Join(ids(out), ",") != "c,b" {
		t.Errorf("skip+limit after sort = %v, want [c b]", ids(out))
	}
}

func TestApply_limitZeroIsNoCap(t *testing.T) {
	recs := []listing.Record{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	out, _ := listing.Apply(recs, listing.Options{Limit: 0})
	if len(out) != 3 {
		t.Errorf("limit 0 = %d records, want 3", len(out))
	}
}

func TestApply_skipBeyondEnd_emptyNotError(t *testing.T) {
	recs := []listing.Record{{ID: "a"}, {ID: "b"}}
	out, err := listing.Apply(recs, listing.Options{Skip: 5})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("skip beyond end = %v, want empty", ids(out))
	}
}

func TestApply_emptyResultIsNotError(t *testing.T) {
	p, _ := predicate.Parse("year=9999")
	out, err := listing.Apply(books(), listing.Options{Filters: []predicate.Predicate{p}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty result, got %v", ids(out))
	}
}
