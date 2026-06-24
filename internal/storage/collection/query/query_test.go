package query_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/storage/collection/query"
)

func TestApply_grepRegions(t *testing.T) {
	recs := []query.Record{
		{ID: "a", Raw: []byte("---\ntitle: A\n---\nbody has TODO\n"), Frontmatter: []byte("title: A\n"), Body: []byte("body has TODO\n")},
		{ID: "b", Raw: []byte("---\ntitle: TODO\n---\nclean body\n"), Frontmatter: []byte("title: TODO\n"), Body: []byte("clean body\n")},
	}
	re := regexp.MustCompile("TODO")

	all, _ := query.Apply(recs, query.Options{Greps: []*regexp.Regexp{re}, GrepIn: query.RegionAll})
	if strings.Join(ids(all), ",") != "a,b" {
		t.Errorf("grep all = %v, want [a b]", ids(all))
	}
	body, _ := query.Apply(recs, query.Options{Greps: []*regexp.Regexp{re}, GrepIn: query.RegionBody})
	if strings.Join(ids(body), ",") != "a" {
		t.Errorf("grep body = %v, want [a]", ids(body))
	}
	fm, _ := query.Apply(recs, query.Options{Greps: []*regexp.Regexp{re}, GrepIn: query.RegionFrontmatter})
	if strings.Join(ids(fm), ",") != "b" {
		t.Errorf("grep frontmatter = %v, want [b]", ids(fm))
	}
}

func TestApply_grepAndFilterCombine(t *testing.T) {
	recs := []query.Record{
		{ID: "a", Meta: map[string]any{"status": "draft"}, Raw: []byte("TODO here")},
		{ID: "b", Meta: map[string]any{"status": "draft"}, Raw: []byte("nothing")},
		{ID: "c", Meta: map[string]any{"status": "published"}, Raw: []byte("TODO here")},
	}
	p, _ := query.ParseFilter("status=draft")
	out, _ := query.Apply(recs, query.Options{
		Filters: []query.Predicate{p},
		Greps:   []*regexp.Regexp{regexp.MustCompile("TODO")},
	})
	if strings.Join(ids(out), ",") != "a" {
		t.Errorf("filter AND grep = %v, want [a]", ids(out))
	}
}

func TestApply_skipAndLimitAfterSort(t *testing.T) {
	recs := []query.Record{
		{ID: "a", Meta: map[string]any{"year": 2001}},
		{ID: "b", Meta: map[string]any{"year": 2002}},
		{ID: "c", Meta: map[string]any{"year": 2003}},
		{ID: "d", Meta: map[string]any{"year": 2004}},
	}
	keys, _ := query.ParseSort("-year")
	out, _ := query.Apply(recs, query.Options{Sorts: keys, Skip: 1, Limit: 2})
	// -year: d,c,b,a → skip 1 → c,b,a → limit 2 → c,b
	if strings.Join(ids(out), ",") != "c,b" {
		t.Errorf("skip+limit after sort = %v, want [c b]", ids(out))
	}
}

func TestApply_limitZeroIsNoCap(t *testing.T) {
	recs := []query.Record{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	out, _ := query.Apply(recs, query.Options{Limit: 0})
	if len(out) != 3 {
		t.Errorf("limit 0 = %d records, want 3", len(out))
	}
}

func TestApply_skipBeyondEnd_emptyNotError(t *testing.T) {
	recs := []query.Record{{ID: "a"}, {ID: "b"}}
	out, err := query.Apply(recs, query.Options{Skip: 5})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("skip beyond end = %v, want empty", ids(out))
	}
}

func TestApply_emptyResultIsNotError(t *testing.T) {
	p, _ := query.ParseFilter("year=9999")
	out, err := query.Apply(books(), query.Options{Filters: []query.Predicate{p}})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty result, got %v", ids(out))
	}
}
