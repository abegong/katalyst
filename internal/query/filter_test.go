package query_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/abegong/katalyst/internal/query"
)

// filterIDs parses the given filter expressions, applies them, and returns
// the surviving record ids in order.
func filterIDs(t *testing.T, recs []query.Record, exprs ...string) []string {
	t.Helper()
	opts := query.Options{}
	for _, e := range exprs {
		p, err := query.ParseFilter(e)
		if err != nil {
			t.Fatalf("ParseFilter(%q): %v", e, err)
		}
		opts.Filters = append(opts.Filters, p)
	}
	out, err := query.Apply(recs, opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return ids(out)
}

func ids(recs []query.Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.ID
	}
	return out
}

func books() []query.Record {
	return []query.Record{
		{ID: "dune", Meta: map[string]any{"year": 1965, "status": "published", "tags": []any{"sci-fi", "classic"}, "title": "Dune"}},
		{ID: "hobbit", Meta: map[string]any{"year": 1937, "status": "published", "tags": []any{"fantasy"}, "title": "The Hobbit"}},
		{ID: "wip", Meta: map[string]any{"year": 2025, "status": "draft", "title": "Work in Progress"}},
	}
}

func TestFilter_comparisons(t *testing.T) {
	cases := []struct {
		expr string
		want []string
	}{
		{"year>=1965", []string{"dune", "wip"}},
		{"year>1965", []string{"wip"}},
		{"year<=1965", []string{"dune", "hobbit"}},
		{"year<1965", []string{"hobbit"}},
		{"year=1965", []string{"dune"}},
		{"year!=1965", []string{"hobbit", "wip"}},
		{"status=draft", []string{"wip"}},
		{"status=published", []string{"dune", "hobbit"}},
	}
	for _, c := range cases {
		if got := filterIDs(t, books(), c.expr); strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("filter %q = %v, want %v", c.expr, got, c.want)
		}
	}
}

func TestFilter_existsAndAbsent(t *testing.T) {
	if got := filterIDs(t, books(), "tags"); strings.Join(got, ",") != "dune,hobbit" {
		t.Errorf("exists 'tags' = %v, want [dune hobbit]", got)
	}
	if got := filterIDs(t, books(), "!tags"); strings.Join(got, ",") != "wip" {
		t.Errorf("absent '!tags' = %v, want [wip]", got)
	}
}

func TestFilter_inAndNin(t *testing.T) {
	// Scalar membership.
	if got := filterIDs(t, books(), "year=1965,1937"); strings.Join(got, ",") != "dune,hobbit" {
		t.Errorf("in = %v, want [dune hobbit]", got)
	}
	// Array membership: tags shares an element with the list.
	if got := filterIDs(t, books(), "tags=fantasy,horror"); strings.Join(got, ",") != "hobbit" {
		t.Errorf("array in = %v, want [hobbit]", got)
	}
	// Single = against an array field matches membership (Mongo-style).
	if got := filterIDs(t, books(), "tags=sci-fi"); strings.Join(got, ",") != "dune" {
		t.Errorf("array single-eq = %v, want [dune]", got)
	}
	// nin.
	if got := filterIDs(t, books(), "status=draft,published"); len(got) != 3 {
		t.Errorf("in all = %v, want all three", got)
	}
	if got := filterIDs(t, books(), "year!=1965,1937"); strings.Join(got, ",") != "wip" {
		t.Errorf("nin = %v, want [wip]", got)
	}
}

func TestFilter_regex(t *testing.T) {
	if got := filterIDs(t, books(), "title=~^The"); strings.Join(got, ",") != "hobbit" {
		t.Errorf("regex = %v, want [hobbit]", got)
	}
	if got := filterIDs(t, books(), "title=~(?i)^the"); strings.Join(got, ",") != "hobbit" {
		t.Errorf("inline case-insensitive regex = %v, want [hobbit]", got)
	}
}

func TestFilter_dotPath(t *testing.T) {
	recs := []query.Record{
		{ID: "a", Meta: map[string]any{"author": map[string]any{"name": "Herbert"}}},
		{ID: "b", Meta: map[string]any{"author": map[string]any{"name": "Tolkien"}}},
	}
	if got := filterIDs(t, recs, "author.name=Herbert"); strings.Join(got, ",") != "a" {
		t.Errorf("dot path = %v, want [a]", got)
	}
}

func TestFilter_multipleAreANDed(t *testing.T) {
	if got := filterIDs(t, books(), "status=published", "year>=1950"); strings.Join(got, ",") != "dune" {
		t.Errorf("AND = %v, want [dune]", got)
	}
}

func TestFilter_typeMismatch_skipByDefault(t *testing.T) {
	recs := []query.Record{
		{ID: "ok", Meta: map[string]any{"year": 2000}},
		{ID: "bad", Meta: map[string]any{"year": "twenty"}},
	}
	// Default skip: the string-year item simply does not match.
	if got := filterIDs(t, recs, "year>=1965"); strings.Join(got, ",") != "ok" {
		t.Errorf("mismatch skip = %v, want [ok]", got)
	}
}

func TestFilter_typeMismatch_errorMode(t *testing.T) {
	recs := []query.Record{{ID: "bad", Meta: map[string]any{"year": "twenty"}}}
	p, err := query.ParseFilter("year>=1965")
	if err != nil {
		t.Fatal(err)
	}
	_, err = query.Apply(recs, query.Options{Filters: []query.Predicate{p}, TypeMismatch: "error"})
	var tme *query.TypeMismatchError
	if !errors.As(err, &tme) {
		t.Fatalf("expected TypeMismatchError, got %v", err)
	}
}

func TestFilter_unparseableFrontmatter_matchesAbsent(t *testing.T) {
	// An item with nil Meta (parse failure) matches !FIELD and fails
	// positive predicates.
	recs := []query.Record{{ID: "broken", Meta: nil}}
	if got := filterIDs(t, recs, "!title"); strings.Join(got, ",") != "broken" {
		t.Errorf("absent on nil meta = %v, want [broken]", got)
	}
	if got := filterIDs(t, recs, "title=x"); len(got) != 0 {
		t.Errorf("positive on nil meta = %v, want none", got)
	}
}

func TestPredicate_Matches(t *testing.T) {
	meta := map[string]any{"kind": "section", "year": 1965}

	mustMatch := func(expr string, want bool) {
		t.Helper()
		p, err := query.ParseFilter(expr)
		if err != nil {
			t.Fatalf("ParseFilter(%q): %v", expr, err)
		}
		got, err := p.Matches(meta, "skip")
		if err != nil {
			t.Fatalf("Matches(%q): %v", expr, err)
		}
		if got != want {
			t.Errorf("Matches(%q) = %v, want %v", expr, got, want)
		}
	}

	mustMatch("kind=section", true)
	mustMatch("kind=page", false)
	mustMatch("year>=1965", true)
	mustMatch("!draft", true)   // absent field
	mustMatch("kind", true)     // existence

	// typeMismatch threads through to match: "error" surfaces the error,
	// "skip" reports a non-match.
	p, err := query.ParseFilter("year>=1965")
	if err != nil {
		t.Fatal(err)
	}
	strMeta := map[string]any{"year": "twenty"}
	if got, err := p.Matches(strMeta, "skip"); got || err != nil {
		t.Errorf("Matches skip on type mismatch = (%v, %v), want (false, nil)", got, err)
	}
	if _, err := p.Matches(strMeta, "error"); err == nil {
		t.Error("Matches error mode on type mismatch: expected error")
	}
}

func TestParseFilter_errors(t *testing.T) {
	for _, expr := range []string{"", "=value", "title=~("} {
		if _, err := query.ParseFilter(expr); err == nil {
			t.Errorf("ParseFilter(%q) expected error", expr)
		}
	}
}
